use crate::{
    binary::{
        BlockType, Code, ElementMode, ExportKind, FloatInstruction, HirInstruction, ImportKind,
        NameCustomSection, TableType, WasmBinary,
    },
    host::get_host_impl,
    memory::Memory,
    merkle::{Merkle, MerkleType},
    reinterpret::{ReinterpretAsSigned, ReinterpretAsUnsigned},
    utils::Bytes32,
    value::{FunctionType, IntegerValType, ProgramCounter, Value, ValueType},
    wavm::{pack_cross_module_call, unpack_cross_module_call, FloatingPointImpls, Instruction},
    wavm::{FunctionCodegenState, IBinOpType, IRelOpType, IUnOpType, Opcode},
};
use digest::Digest;
use fnv::FnvHashMap as HashMap;
use num::{traits::PrimInt, Zero};
use rayon::prelude::*;
use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use std::{
    borrow::Cow,
    convert::TryFrom,
    fs::File,
    io::{BufReader, BufWriter, Write},
    num::Wrapping,
    ops::{Deref, DerefMut},
    path::Path,
    sync::Arc,
};

fn hash_call_indirect_data(table: u32, ty: &FunctionType) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update("Call indirect:");
    h.update(&(table as u64).to_be_bytes());
    h.update(ty.hash());
    h.finalize().into()
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum InboxIdentifier {
    Sequencer = 0,
    Delayed,
}

pub fn argument_data_to_inbox(argument_data: u64) -> Result<InboxIdentifier, ()> {
    match argument_data {
        0x0 => Ok(InboxIdentifier::Sequencer),
        0x1 => Ok(InboxIdentifier::Delayed),
        _ => Err(()),
    }
}

#[derive(Clone, Debug)]
pub struct Function {
    code: Vec<Instruction>,
    ty: FunctionType,
    code_merkle: Merkle,
    local_types: Vec<ValueType>,
}

impl Function {
    pub fn new(
        code: Code,
        func_ty: FunctionType,
        func_block_ty: BlockType,
        module_types: &[FunctionType],
        fp_impls: &FloatingPointImpls,
    ) -> Function {
        let locals_with_params: Vec<ValueType> =
            func_ty.inputs.iter().cloned().chain(code.locals).collect();
        let mut insts = Vec::new();
        let empty_local_hashes = locals_with_params
            .iter()
            .cloned()
            .map(Value::default_of_type)
            .map(Value::hash)
            .collect::<Vec<_>>();
        insts.push(Instruction {
            opcode: Opcode::InitFrame,
            argument_data: 0,
            proving_argument_data: Some(Merkle::new(MerkleType::Value, empty_local_hashes).root()),
        });
        // Fill in parameters
        for i in (0..func_ty.inputs.len()).rev() {
            insts.push(Instruction {
                opcode: Opcode::LocalSet,
                argument_data: i as u64,
                proving_argument_data: None,
            });
        }
        insts.push(Instruction::simple(Opcode::PushStackBoundary));
        let codegen_state = FunctionCodegenState::new(func_ty.outputs.len(), fp_impls);

        Instruction::extend_from_hir(
            &mut insts,
            codegen_state,
            crate::binary::HirInstruction::Block(func_block_ty, code.expr),
        );

        Instruction::extend_from_hir(
            &mut insts,
            codegen_state,
            crate::binary::HirInstruction::Simple(Opcode::Return),
        );

        // Insert missing proving argument data
        for inst in insts.iter_mut() {
            if inst.opcode == Opcode::CallIndirect {
                let (table, ty) = crate::wavm::unpack_call_indirect(inst.argument_data);
                let ty = &module_types[usize::try_from(ty).unwrap()];
                inst.proving_argument_data = Some(hash_call_indirect_data(table, ty));
            }
        }

        Function::new_from_wavm(insts, func_ty, locals_with_params)
    }

    fn new_from_wavm(
        code: Vec<Instruction>,
        ty: FunctionType,
        local_types: Vec<ValueType>,
    ) -> Function {
        assert!(
            u32::try_from(code.len()).is_ok(),
            "Function instruction count doesn't fit in a u32",
        );
        let code_merkle = Merkle::new(
            MerkleType::Instruction,
            code.par_iter().map(|i| i.hash()).collect(),
        );

        Function {
            code,
            ty,
            code_merkle,
            local_types,
        }
    }

    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Function:");
        h.update(self.code_merkle.root());
        h.finalize().into()
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
struct StackFrame {
    return_ref: Value,
    locals: Vec<Value>,
    caller_module: u32,
    caller_module_internals: u32,
}

impl StackFrame {
    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Stack frame:");
        h.update(&self.return_ref.hash());
        h.update(
            Merkle::new(
                MerkleType::Value,
                self.locals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );
        h.update(self.caller_module.to_be_bytes());
        h.update(self.caller_module_internals.to_be_bytes());
        h.finalize().into()
    }

    fn serialize_for_proof(&self) -> Vec<u8> {
        let mut data = Vec::new();
        data.extend(self.return_ref.serialize_for_proof());
        data.extend(
            Merkle::new(
                MerkleType::Value,
                self.locals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );
        data.extend(self.caller_module.to_be_bytes());
        data.extend(self.caller_module_internals.to_be_bytes());
        data
    }
}

#[derive(Clone, Debug)]
struct TableElement {
    func_ty: FunctionType,
    val: Value,
}

impl Default for TableElement {
    fn default() -> Self {
        TableElement {
            func_ty: FunctionType::default(),
            val: Value::RefNull,
        }
    }
}

impl TableElement {
    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Table element:");
        h.update(self.func_ty.hash());
        h.update(self.val.hash());
        h.finalize().into()
    }
}

#[derive(Clone, Debug)]
struct Table {
    ty: TableType,
    elems: Vec<TableElement>,
    elems_merkle: Merkle,
}

impl Table {
    fn serialize_for_proof(&self) -> Vec<u8> {
        let mut data = Vec::new();
        data.push(Into::<ValueType>::into(self.ty.ty).serialize());
        data.extend(&(self.elems.len() as u64).to_be_bytes());
        data.extend(self.elems_merkle.root());
        data
    }

    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Table:");
        h.update(&[Into::<ValueType>::into(self.ty.ty).serialize()]);
        h.update(&(self.elems.len() as u64).to_be_bytes());
        h.update(self.elems_merkle.root());
        h.finalize().into()
    }
}

fn make_internal_func(opcode: Opcode, ty: FunctionType) -> Function {
    let mut wavm = Vec::new();
    wavm.push(Instruction::simple(Opcode::InitFrame));
    wavm.push(Instruction::simple(opcode));
    wavm.push(Instruction::simple(Opcode::Return));
    Function::new_from_wavm(wavm, ty, Vec::new())
}

#[derive(Clone, Debug)]
struct AvailableImport {
    ty: FunctionType,
    module: u32,
    func: u32,
}

#[derive(Clone, Debug, Default)]
struct Module {
    globals: Vec<Value>,
    memory: Memory,
    tables: Vec<Table>,
    tables_merkle: Merkle,
    funcs: Arc<Vec<Function>>,
    funcs_merkle: Arc<Merkle>,
    types: Arc<Vec<FunctionType>>,
    internals_offset: u32,
    names: Arc<NameCustomSection>,
    host_call_hooks: Arc<Vec<Option<(String, String)>>>,
    start_function: Option<u32>,
    func_types: Arc<Vec<FunctionType>>,
    exports: Arc<HashMap<String, u32>>,
}

impl Module {
    fn from_binary(
        bin: WasmBinary,
        available_imports: &HashMap<String, AvailableImport>,
        floating_point_impls: &FloatingPointImpls,
        allow_hostapi: bool,
    ) -> Module {
        let mut code = Vec::new();
        let mut func_type_idxs: Vec<u32> = Vec::new();
        let mut memory = Memory::default();
        let mut exports = HashMap::default();
        let mut tables = Vec::new();
        let mut host_call_hooks = Vec::new();
        for import in bin.imports {
            if let ImportKind::Function(ty) = import.kind {
                let mut qualified_name = format!("{}__{}", import.module, import.name);
                qualified_name = qualified_name.replace(&['/', '.'] as &[char], "_");
                let func;
                if let Some(import) = available_imports.get(&qualified_name) {
                    assert_eq!(
                        import.ty, bin.types[ty as usize],
                        "Import has different function signature than host function",
                    );
                    let mut wavm = Vec::new();
                    wavm.push(Instruction::simple(Opcode::InitFrame));
                    wavm.push(Instruction::with_data(
                        Opcode::CrossModuleCall,
                        pack_cross_module_call(import.func, import.module),
                    ));
                    wavm.push(Instruction::simple(Opcode::Return));
                    func = Function::new_from_wavm(wavm, import.ty.clone(), Vec::new());
                } else {
                    func = get_host_impl(
                        &import.module,
                        &import.name,
                        BlockType::TypeIndex(ty as u32),
                    );
                    assert_eq!(
                        func.ty, bin.types[ty as usize],
                        "Import has different function signature than host function",
                    );
                    assert!(
                        allow_hostapi,
                        "Calling hostapi directly is not allowed. Function {}",
                        import.name,
                    );
                }
                func_type_idxs.push(ty);
                code.push(func);
                host_call_hooks.push(Some((import.module, import.name)));
            } else {
                panic!("Unsupport import kind {:?}", import);
            }
        }
        func_type_idxs.extend(bin.functions.into_iter());
        let types = &bin.types;
        let mut func_types: Vec<FunctionType> = func_type_idxs
            .iter()
            .map(|i| types[*i as usize].clone())
            .collect();
        for c in bin.code {
            let idx = code.len();
            code.push(Function::new(
                c,
                func_types[idx].clone(),
                BlockType::TypeIndex(func_type_idxs[idx]),
                &bin.types,
                floating_point_impls,
            ));
            host_call_hooks.push(None);
        }
        assert!(
            bin.memories.len() <= 1,
            "Multiple memories are not supported"
        );
        if let Some(limits) = bin.memories.get(0) {
            // We ignore the maximum size
            let size = usize::try_from(limits.minimum_size)
                .ok()
                .and_then(|x| x.checked_mul(Memory::PAGE_SIZE))
                .expect("Memory size is too large");
            memory = Memory::new(size);
        }
        let globals = bin
            .globals
            .into_iter()
            .map(|g| {
                if let [insn] = g.initializer.as_slice() {
                    if let Some(val) = insn.get_const_output() {
                        return val;
                    }
                }
                panic!("Global initializer isn't a constant");
            })
            .collect();
        for export in bin.exports {
            if let ExportKind::Function(idx) = export.kind {
                exports.insert(export.name, idx);
            }
        }
        for data in bin.datas {
            if let Some(loc) = data.active_location {
                assert_eq!(loc.memory, 0, "Attempted to write to nonexistant memory");
                let mut offset = None;
                if let [insn] = loc.offset.as_slice() {
                    if let Some(Value::I32(x)) = insn.get_const_output() {
                        offset = Some(x);
                    }
                }
                let offset =
                    usize::try_from(offset.expect("Non-constant data offset expression")).unwrap();
                if !matches!(
                    offset.checked_add(data.data.len()),
                    Some(x) if (x as u64) < memory.size() as u64,
                ) {
                    panic!(
                        "Out-of-bounds data memory init with offset {} and size {}",
                        offset,
                        data.data.len(),
                    );
                }
                memory.set_range(offset, &data.data);
            }
        }
        for table in bin.tables {
            tables.push(Table {
                elems: vec![
                    TableElement::default();
                    usize::try_from(table.limits.minimum_size).unwrap()
                ],
                ty: table,
                elems_merkle: Merkle::default(),
            });
        }
        for elem in bin.elements {
            if let ElementMode::Active(t, o) = elem.mode {
                let mut offset = None;
                if let [insn] = o.as_slice() {
                    if let Some(Value::I32(x)) = insn.get_const_output() {
                        offset = Some(x);
                    }
                }
                let offset =
                    usize::try_from(offset.expect("Non-constant data offset expression")).unwrap();
                let t = usize::try_from(t).unwrap();
                assert_eq!(tables[t].ty.ty, elem.ty);
                let contents: Vec<_> = elem
                    .init
                    .into_iter()
                    .map(|i| {
                        let insn = match i.as_slice() {
                            [x] => x,
                            _ => panic!("Element initializer isn't one instruction: {:?}", o),
                        };
                        match insn.get_const_output() {
                            Some(v @ Value::RefNull) => TableElement {
                                func_ty: FunctionType::default(),
                                val: v,
                            },
                            Some(Value::FuncRef(x)) => TableElement {
                                func_ty: func_types[usize::try_from(x).unwrap()].clone(),
                                val: Value::FuncRef(x),
                            },
                            _ => panic!("Invalid element initializer {:?}", insn),
                        }
                    })
                    .collect();
                let len = contents.len();
                tables[t].elems[offset..][..len].clone_from_slice(&contents);
            }
        }
        assert!(
            code.len() < (1usize << 31),
            "Module function count must be under 2^31",
        );
        assert!(!code.is_empty(), "Module has no code");

        // Make internal functions
        let internals_offset = code.len() as u32;
        let mut memory_load_internal_type = FunctionType::default();
        memory_load_internal_type.inputs.push(ValueType::I32);
        memory_load_internal_type.outputs.push(ValueType::I32);
        func_types.push(memory_load_internal_type.clone());
        code.push(make_internal_func(
            Opcode::MemoryLoad {
                ty: ValueType::I32,
                bytes: 1,
                signed: false,
            },
            memory_load_internal_type.clone(),
        ));
        func_types.push(memory_load_internal_type.clone());
        code.push(make_internal_func(
            Opcode::MemoryLoad {
                ty: ValueType::I32,
                bytes: 4,
                signed: false,
            },
            memory_load_internal_type,
        ));
        let mut memory_store_internal_type = FunctionType::default();
        memory_store_internal_type.inputs.push(ValueType::I32);
        memory_store_internal_type.inputs.push(ValueType::I32);
        func_types.push(memory_store_internal_type.clone());
        code.push(make_internal_func(
            Opcode::MemoryStore {
                ty: ValueType::I32,
                bytes: 1,
            },
            memory_store_internal_type.clone(),
        ));
        func_types.push(memory_store_internal_type.clone());
        code.push(make_internal_func(
            Opcode::MemoryStore {
                ty: ValueType::I32,
                bytes: 4,
            },
            memory_store_internal_type,
        ));

        Module {
            memory,
            globals,
            tables_merkle: Merkle::new(MerkleType::Table, tables.iter().map(Table::hash).collect()),
            tables,
            funcs_merkle: Arc::new(Merkle::new(
                MerkleType::Function,
                code.iter().map(|f| f.hash()).collect(),
            )),
            funcs: Arc::new(code),
            types: Arc::new(bin.types),
            internals_offset,
            names: Arc::new(bin.names),
            host_call_hooks: Arc::new(host_call_hooks),
            start_function: bin.start,
            func_types: Arc::new(func_types),
            exports: Arc::new(exports),
        }
    }

    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Module:");
        h.update(
            Merkle::new(
                MerkleType::Value,
                self.globals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );
        h.update(self.memory.hash());
        h.update(self.tables_merkle.root());
        h.update(self.funcs_merkle.root());
        h.update(self.internals_offset.to_be_bytes());
        h.finalize().into()
    }

    fn serialize_for_proof(&self, mem_merkle: &Merkle) -> Vec<u8> {
        let mut data = Vec::new();

        data.extend(
            Merkle::new(
                MerkleType::Value,
                self.globals.iter().map(|v| v.hash()).collect(),
            )
            .root(),
        );

        data.extend(self.memory.size().to_be_bytes());
        data.extend(mem_merkle.root());

        data.extend(self.tables_merkle.root());
        data.extend(self.funcs_merkle.root());

        data.extend(self.internals_offset.to_be_bytes());

        data
    }
}

// Globalstate holds:
// bytes32 - lastblockhash
// uint64 - inbox_position
// uint64 - position_within_message
pub const GLOBAL_STATE_BYTES32_NUM: usize = 1;
pub const GLOBAL_STATE_U64_NUM: usize = 2;

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[repr(C)]
pub struct GlobalState {
    pub bytes32_vals: [Bytes32; GLOBAL_STATE_BYTES32_NUM],
    pub u64_vals: [u64; GLOBAL_STATE_U64_NUM],
}

impl GlobalState {
    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Global state:");
        for item in self.bytes32_vals {
            h.update(item)
        }
        for item in self.u64_vals {
            h.update(item.to_be_bytes())
        }
        h.finalize().into()
    }

    fn serialize(&self) -> Vec<u8> {
        let mut data = Vec::new();
        for item in self.bytes32_vals {
            data.extend(item)
        }
        for item in self.u64_vals {
            data.extend(item.to_be_bytes())
        }
        data
    }
}

struct LazyModuleMerkle<'a>(&'a mut Module, Option<&'a mut Merkle>, usize);

impl Deref for LazyModuleMerkle<'_> {
    type Target = Module;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for LazyModuleMerkle<'_> {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl Drop for LazyModuleMerkle<'_> {
    fn drop(&mut self) {
        if let Some(merkle) = &mut self.1 {
            merkle.set(self.2, self.0.hash());
        }
    }
}

/// cbindgen:ignore
#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[repr(u8)]
pub enum MachineStatus {
    Running,
    Finished,
    Errored,
    TooFar,
}

#[derive(Clone, Serialize, Deserialize)]
pub struct ModuleState<'a> {
    globals: Cow<'a, Vec<Value>>,
    memory: Cow<'a, Memory>,
}

#[derive(Serialize, Deserialize)]
pub struct MachineState<'a> {
    steps: u64, // Not part of machine hash
    status: MachineStatus,
    value_stack: Cow<'a, Vec<Value>>,
    internal_stack: Cow<'a, Vec<Value>>,
    block_stack: Cow<'a, Vec<usize>>,
    frame_stack: Cow<'a, Vec<StackFrame>>,
    modules: Vec<ModuleState<'a>>,
    global_state: GlobalState,
    pc: ProgramCounter,
    stdio_output: Cow<'a, Vec<u8>>,
    initial_hash: Bytes32,
}

#[derive(Clone)]
pub struct Machine {
    steps: u64, // Not part of machine hash
    status: MachineStatus,
    value_stack: Vec<Value>,
    internal_stack: Vec<Value>,
    block_stack: Vec<usize>,
    frame_stack: Vec<StackFrame>,
    modules: Vec<Module>,
    modules_merkle: Option<Merkle>,
    global_state: GlobalState,
    pc: ProgramCounter,
    stdio_output: Vec<u8>,
    inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
    preimages: HashMap<Bytes32, Vec<u8>>,
    initial_hash: Bytes32,
}

fn hash_stack<I, D>(stack: I, prefix: &str) -> Bytes32
where
    I: IntoIterator<Item = D>,
    D: AsRef<[u8]>,
{
    let mut hash = Bytes32::default();
    for item in stack.into_iter() {
        let mut h = Keccak256::new();
        h.update(prefix);
        h.update(item.as_ref());
        h.update(&hash);
        hash = h.finalize().into();
    }
    hash
}

fn hash_value_stack(stack: &[Value]) -> Bytes32 {
    hash_stack(stack.iter().map(|v| v.hash()), "Value stack:")
}

fn hash_pc_stack(pcs: &[usize]) -> Bytes32 {
    hash_stack(
        pcs.iter().map(|pc| (*pc as u32).to_be_bytes()),
        "Program counter stack:",
    )
}

fn hash_stack_frame_stack(frames: &[StackFrame]) -> Bytes32 {
    hash_stack(frames.iter().map(|f| f.hash()), "Stack frame stack:")
}

#[must_use]
fn prove_window<T, F, D, G>(items: &[T], stack_hasher: F, encoder: G) -> Vec<u8>
where
    F: Fn(&[T]) -> Bytes32,
    D: AsRef<[u8]>,
    G: Fn(&T) -> D,
{
    let mut data = Vec::with_capacity(33);
    if items.is_empty() {
        data.extend(Bytes32::default());
        data.push(0);
    } else {
        let last_idx = items.len() - 1;
        data.extend(stack_hasher(&items[..last_idx]));
        data.push(1);
        data.extend(encoder(&items[last_idx]).as_ref());
    }
    data
}

#[must_use]
fn prove_stack<T, F, D, G>(
    items: &[T],
    proving_depth: usize,
    stack_hasher: F,
    encoder: G,
) -> Vec<u8>
where
    F: Fn(&[T]) -> Bytes32,
    D: AsRef<[u8]>,
    G: Fn(&T) -> D,
{
    let mut data = Vec::with_capacity(33);
    let unproven_stack_depth = items.len().saturating_sub(proving_depth);
    data.extend(stack_hasher(&items[..unproven_stack_depth]));
    data.extend(Bytes32::from(items.len() - unproven_stack_depth));
    for val in &items[unproven_stack_depth..] {
        data.extend(encoder(val).as_ref());
    }
    data
}

#[must_use]
fn exec_ibin_op<T>(a: T, b: T, op: IBinOpType) -> T
where
    Wrapping<T>: ReinterpretAsSigned,
    T: Zero,
{
    let a = Wrapping(a);
    let b = Wrapping(b);
    if matches!(
        op,
        IBinOpType::DivS | IBinOpType::DivU | IBinOpType::RemS | IBinOpType::RemU,
    ) {
        if b.is_zero() {
            return T::zero();
        }
    }
    let res = match op {
        IBinOpType::Add => a + b,
        IBinOpType::Sub => a - b,
        IBinOpType::Mul => a * b,
        IBinOpType::DivS => (a.cast_signed() / b.cast_signed()).cast_unsigned(),
        IBinOpType::DivU => a / b,
        IBinOpType::RemS => (a.cast_signed() % b.cast_signed()).cast_unsigned(),
        IBinOpType::RemU => a % b,
        IBinOpType::And => a & b,
        IBinOpType::Or => a | b,
        IBinOpType::Xor => a ^ b,
        IBinOpType::Shl => a << b.cast_usize(),
        IBinOpType::ShrS => (a.cast_signed() >> b.cast_usize()).cast_unsigned(),
        IBinOpType::ShrU => a >> b.cast_usize(),
        IBinOpType::Rotl => a.rotl(b.cast_usize()),
        IBinOpType::Rotr => a.rotr(b.cast_usize()),
    };
    res.0
}

#[must_use]
fn exec_iun_op<T>(a: T, op: IUnOpType) -> u32
where
    T: PrimInt,
{
    match op {
        IUnOpType::Clz => a.leading_zeros(),
        IUnOpType::Ctz => a.trailing_zeros(),
        IUnOpType::Popcnt => a.count_ones(),
    }
}

fn exec_irel_op<T>(a: T, b: T, op: IRelOpType) -> Value
where
    T: Ord,
{
    let res = match op {
        IRelOpType::Eq => a == b,
        IRelOpType::Ne => a != b,
        IRelOpType::Lt => a < b,
        IRelOpType::Gt => a > b,
        IRelOpType::Le => a <= b,
        IRelOpType::Ge => a >= b,
    };
    Value::I32(res as u32)
}

impl Machine {
    pub const MAX_STEPS: u64 = 1 << 43;

    pub fn from_binary(
        libraries: Vec<WasmBinary>,
        bin: WasmBinary,
        always_merkleize: bool,
        allow_hostapi_from_main: bool,
        global_state: GlobalState,
        inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
        preimages: HashMap<Bytes32, Vec<u8>>,
    ) -> Machine {
        // `modules` starts out with the entrypoint module, which will be initialized later
        let mut modules = vec![Module::default()];
        let mut available_imports = HashMap::default();
        let mut floating_point_impls = HashMap::default();

        for export in &bin.exports {
            if let ExportKind::Function(f) = export.kind {
                if let Some(ty_idx) = usize::try_from(f).unwrap().checked_sub(bin.imports.len()) {
                    let ty = bin.functions[ty_idx];
                    let ty = &bin.types[usize::try_from(ty).unwrap()];
                    let module = u32::try_from(modules.len() + libraries.len()).unwrap();
                    available_imports.insert(
                        format!("env__wavm_guest_call__{}", export.name),
                        AvailableImport {
                            ty: ty.clone(),
                            module,
                            func: f,
                        },
                    );
                }
            }
        }

        for lib in libraries {
            let module = Module::from_binary(lib, &available_imports, &floating_point_impls, true);
            for (name, &func) in &*module.exports {
                let ty = module.func_types[func as usize].clone();
                available_imports.insert(
                    name.clone(),
                    AvailableImport {
                        module: modules.len() as u32,
                        func: func,
                        ty: ty.clone(),
                    },
                );
                if let Ok(op) = name.parse::<FloatInstruction>() {
                    let mut sig = op.signature();
                    // wavm codegen takes care of effecting this type change at callsites
                    for ty in sig.inputs.iter_mut().chain(sig.outputs.iter_mut()) {
                        if *ty == ValueType::F32 {
                            *ty = ValueType::I32;
                        } else if *ty == ValueType::F64 {
                            *ty = ValueType::I64;
                        }
                    }
                    assert_eq!(ty, sig, "Wrong type for floating point impl {:?}", name);
                    floating_point_impls.insert(op, (modules.len() as u32, func));
                }
            }
            modules.push(module);
        }

        // Shouldn't be necessary, but to safe, don't allow the binary to import its own guest calls
        available_imports.retain(|_, i| i.module as usize != modules.len());
        modules.push(Module::from_binary(
            bin,
            &available_imports,
            &floating_point_impls,
            allow_hostapi_from_main,
        ));

        // Build the entrypoint module
        let mut entrypoint = Vec::new();
        for (i, module) in modules.iter().enumerate() {
            if let Some(s) = module.start_function {
                assert!(
                    module.func_types[s as usize] == FunctionType::default(),
                    "Start function takes inputs or outputs",
                );
                entrypoint.push(HirInstruction::CrossModuleCall(
                    u32::try_from(i).unwrap(),
                    s,
                ));
            }
        }
        let main_module_idx = modules.len() - 1;
        let main_module = &modules[main_module_idx];
        // Rust support
        if let Some(&f) = main_module.exports.get("main") {
            let mut expected_type = FunctionType::default();
            expected_type.inputs.push(ValueType::I32); // argc
            expected_type.inputs.push(ValueType::I32); // argv
            expected_type.outputs.push(ValueType::I32); // ret
            assert!(
                main_module.func_types[f as usize] == expected_type,
                "Main function doesn't match expected signature of [argc, argv] -> [ret]",
            );
            entrypoint.push(HirInstruction::I32Const(0));
            entrypoint.push(HirInstruction::I32Const(0));
            entrypoint.push(HirInstruction::CrossModuleCall(
                u32::try_from(main_module_idx).unwrap(),
                f,
            ));
            entrypoint.push(HirInstruction::Simple(Opcode::Drop));
            entrypoint.push(HirInstruction::Simple(Opcode::HaltAndSetFinished));
        }
        // Go support
        if let Some(&f) = main_module.exports.get("run") {
            let mut expected_type = FunctionType::default();
            expected_type.inputs.push(ValueType::I32); // argc
            expected_type.inputs.push(ValueType::I32); // argv
            assert!(
                main_module.func_types[f as usize] == expected_type,
                "Run function doesn't match expected signature of [argc, argv]",
            );
            // Go's flags library panics if the argument list is empty.
            // To pass in the program name argument, we need to put it in memory.
            // The Go linker guarantees a section of memory starting at byte 4096 is available for this purpose.
            // https://github.com/golang/go/blob/252324e879e32f948d885f787decf8af06f82be9/misc/wasm/wasm_exec.js#L520
            // These memory stores also assume that the Go module's memory is large enough to begin with.
            // That's also handled by the Go compiler. Go 1.17.5 in the compilation of the arbitrator go test case
            // initializes its memory to 272 pages long (about 18MB), much larger than the required space.
            let free_memory_base = 4096;
            let name_str_ptr = free_memory_base;
            let argv_ptr = name_str_ptr + 8;
            assert!(main_module.internals_offset != 0);
            let main_module_idx = u32::try_from(main_module_idx).unwrap();
            let main_module_store32 =
                HirInstruction::CrossModuleCall(main_module_idx, main_module.internals_offset + 3);
            // Write "js\0" to name_str_ptr, to match what the actual JS environment does
            entrypoint.push(HirInstruction::I32Const(name_str_ptr));
            entrypoint.push(HirInstruction::I32Const(0x736a)); // b"js\0"
            entrypoint.push(main_module_store32.clone());
            entrypoint.push(HirInstruction::I32Const(name_str_ptr + 4));
            entrypoint.push(HirInstruction::I32Const(0));
            entrypoint.push(main_module_store32.clone());
            // Write name_str_ptr to argv_ptr
            entrypoint.push(HirInstruction::I32Const(argv_ptr));
            entrypoint.push(HirInstruction::I32Const(name_str_ptr));
            entrypoint.push(main_module_store32.clone());
            entrypoint.push(HirInstruction::I32Const(argv_ptr + 4));
            entrypoint.push(HirInstruction::I32Const(0));
            entrypoint.push(main_module_store32);
            // Launch main with an argument count of 1 and argv_ptr
            entrypoint.push(HirInstruction::I32Const(1));
            entrypoint.push(HirInstruction::I32Const(argv_ptr));
            entrypoint.push(HirInstruction::CrossModuleCall(main_module_idx, f));
            if let Some(i) = available_imports.get("wavm__go_after_run") {
                assert!(
                    i.ty == FunctionType::default(),
                    "Resume function has non-empty function signature",
                );
                entrypoint.push(HirInstruction::CrossModuleCall(i.module, i.func));
            }
        }
        let entrypoint_types = vec![FunctionType::default()];
        let mut entrypoint_names = NameCustomSection {
            module: "entry".into(),
            functions: HashMap::default(),
            locals: HashMap::default(),
        };
        entrypoint_names
            .functions
            .insert(0, "wavm_entrypoint".into());
        let entrypoint_funcs = vec![Function::new(
            Code {
                locals: Vec::new(),
                expr: entrypoint,
            },
            FunctionType::default(),
            BlockType::TypeIndex(0),
            &entrypoint_types,
            &floating_point_impls,
        )];
        let entrypoint = Module {
            globals: Vec::new(),
            memory: Memory::default(),
            tables: Vec::new(),
            tables_merkle: Merkle::default(),
            funcs_merkle: Arc::new(Merkle::new(
                MerkleType::Function,
                entrypoint_funcs.iter().map(Function::hash).collect(),
            )),
            funcs: Arc::new(entrypoint_funcs),
            types: Arc::new(entrypoint_types),
            names: Arc::new(entrypoint_names),
            internals_offset: 0,
            host_call_hooks: Arc::new(vec![None]),
            start_function: None,
            func_types: Arc::new(vec![FunctionType::default()]),
            exports: Arc::new(HashMap::default()),
        };
        modules[0] = entrypoint;

        // Merkleize things if requested
        for module in &mut modules {
            for table in module.tables.iter_mut() {
                table.elems_merkle = Merkle::new(
                    MerkleType::TableElement,
                    table.elems.iter().map(TableElement::hash).collect(),
                );
            }
            module.tables_merkle = Merkle::new(
                MerkleType::Table,
                module.tables.iter().map(Table::hash).collect(),
            );

            if always_merkleize {
                module.memory.cache_merkle_tree();
            }
        }
        let mut modules_merkle = None;
        if always_merkleize {
            modules_merkle = Some(Merkle::new(
                MerkleType::Module,
                modules.iter().map(Module::hash).collect(),
            ));
        }

        let mut mach = Machine {
            status: MachineStatus::Running,
            steps: 0,
            value_stack: vec![Value::RefNull, Value::I32(0), Value::I32(0)],
            internal_stack: Vec::new(),
            block_stack: Vec::new(),
            frame_stack: Vec::new(),
            modules,
            modules_merkle,
            global_state,
            pc: ProgramCounter::default(),
            stdio_output: Vec::new(),
            inbox_contents,
            preimages,
            initial_hash: Bytes32::default(),
        };
        mach.initial_hash = mach.hash();
        mach
    }

    pub fn serialize_state<P: AsRef<Path>>(&self, path: P) -> eyre::Result<()> {
        let mut f = File::create(path)?;
        let mut writer = BufWriter::new(&mut f);
        let modules = self
            .modules
            .iter()
            .map(|m| ModuleState {
                globals: Cow::Borrowed(&m.globals),
                memory: Cow::Borrowed(&m.memory),
            })
            .collect();
        let state = MachineState {
            steps: self.steps,
            status: self.status,
            value_stack: Cow::Borrowed(&self.value_stack),
            internal_stack: Cow::Borrowed(&self.internal_stack),
            block_stack: Cow::Borrowed(&self.block_stack),
            frame_stack: Cow::Borrowed(&self.frame_stack),
            modules,
            global_state: self.global_state.clone(),
            pc: self.pc,
            stdio_output: Cow::Borrowed(&self.stdio_output),
            initial_hash: self.initial_hash,
        };
        bincode::serialize_into(&mut writer, &state)?;
        writer.flush()?;
        drop(writer);
        f.sync_data()?;
        Ok(())
    }

    // Requires that this is the same base machine. If this returns an error, it has not mutated `self`.
    pub fn deserialize_and_replace_state<P: AsRef<Path>>(&mut self, path: P) -> eyre::Result<()> {
        let reader = BufReader::new(File::open(path)?);
        let new_state: MachineState = bincode::deserialize_from(reader)?;
        if self.initial_hash != new_state.initial_hash {
            eyre::bail!(
                "attempted to load deserialize machine with initial hash {} into machine with initial hash {}",
                new_state.initial_hash, self.initial_hash,
            );
        }
        assert_eq!(self.modules.len(), new_state.modules.len());

        // Start mutating the machine. We must not return an error past this point.
        for (module, new_module_state) in self.modules.iter_mut().zip(new_state.modules.into_iter())
        {
            module.globals = new_module_state.globals.into_owned();
            module.memory = new_module_state.memory.into_owned();
        }
        self.steps = new_state.steps;
        self.status = new_state.status;
        self.value_stack = new_state.value_stack.into_owned();
        self.internal_stack = new_state.internal_stack.into_owned();
        self.block_stack = new_state.block_stack.into_owned();
        self.frame_stack = new_state.frame_stack.into_owned();
        self.global_state = new_state.global_state;
        self.pc = new_state.pc;
        self.stdio_output = new_state.stdio_output.into_owned();
        Ok(())
    }

    pub fn get_next_instruction(&self) -> Option<Instruction> {
        if self.is_halted() {
            return None;
        }
        self.modules[self.pc.module].funcs[self.pc.func]
            .code
            .get(self.pc.inst)
            .cloned()
    }

    pub fn get_pc(&self) -> Option<ProgramCounter> {
        if self.is_halted() {
            return None;
        }
        Some(self.pc)
    }

    fn test_next_instruction(module: &Module, pc: &ProgramCounter) {
        assert!(module.funcs[pc.func].code.len() > pc.inst);
    }

    pub fn get_steps(&self) -> u64 {
        self.steps
    }

    pub fn step_n(&mut self, n: u64) {
        for _ in 0..n {
            if self.is_halted() {
                break;
            }
            self.step();
        }
    }

    pub fn step(&mut self) {
        if self.is_halted() {
            return;
        }
        // It's infeasible to overflow steps without halting
        self.steps += 1;
        if self.steps == Self::MAX_STEPS {
            self.status = MachineStatus::Errored;
            return;
        }

        if self.pc.inst == 1 {
            if let Some(hook) = self.modules[self.pc.module]
                .host_call_hooks
                .get(self.pc.func)
                .cloned()
                .and_then(|x| x)
            {
                if let Err(err) = &self.host_call_hook(&hook.0, &hook.1) {
                    eprintln!(
                        "Failed to process host call hook for host call {:?} {:?}: {}",
                        hook.0, hook.1, err,
                    );
                }
            }
        }

        // Updates the modules_merkle on drop
        let mut module = LazyModuleMerkle(
            &mut self.modules[self.pc.module],
            self.modules_merkle.as_mut(),
            self.pc.module,
        );
        let func = &module.funcs[self.pc.func];
        let code = &func.code;
        let inst = code[self.pc.inst];
        self.pc.inst += 1;
        match inst.opcode {
            Opcode::Unreachable => {
                self.status = MachineStatus::Errored;
            }
            Opcode::Nop => {}
            Opcode::Block => {
                let idx = inst.argument_data as usize;
                self.block_stack.push(idx);
                self.pc.block_depth += 1;
                assert!(module.funcs[self.pc.func].code.len() > idx);
            }
            Opcode::EndBlock => {
                assert!(self.pc.block_depth > 0);
                self.pc.block_depth -= 1;
                self.block_stack.pop();
            }
            Opcode::EndBlockIf => {
                let x = self.value_stack.last().unwrap();
                if !x.is_i32_zero() {
                    assert!(self.pc.block_depth > 0);
                    self.pc.block_depth -= 1;
                    self.block_stack.pop().unwrap();
                }
            }
            Opcode::InitFrame => {
                let caller_module_internals = self.value_stack.pop().unwrap().assume_u32();
                let caller_module = self.value_stack.pop().unwrap().assume_u32();
                let return_ref = self.value_stack.pop().unwrap();
                self.frame_stack.push(StackFrame {
                    return_ref,
                    locals: func
                        .local_types
                        .iter()
                        .cloned()
                        .map(Value::default_of_type)
                        .collect(),
                    caller_module,
                    caller_module_internals,
                });
            }
            Opcode::ArbitraryJumpIf => {
                let x = self.value_stack.pop().unwrap();
                if !x.is_i32_zero() {
                    self.pc.inst = inst.argument_data as usize;
                    Machine::test_next_instruction(&module, &self.pc);
                }
            }
            Opcode::Branch => {
                assert!(self.pc.block_depth > 0);
                self.pc.block_depth -= 1;
                self.pc.inst = self.block_stack.pop().unwrap();
                Machine::test_next_instruction(&module, &self.pc);
            }
            Opcode::BranchIf => {
                let x = self.value_stack.pop().unwrap();
                if !x.is_i32_zero() {
                    assert!(self.pc.block_depth > 0);
                    self.pc.block_depth -= 1;
                    self.pc.inst = self.block_stack.pop().unwrap();
                    Machine::test_next_instruction(&module, &self.pc);
                }
            }
            Opcode::Return => {
                let frame = self.frame_stack.pop().unwrap();
                match frame.return_ref {
                    Value::RefNull => {
                        self.status = MachineStatus::Errored;
                    }
                    Value::InternalRef(pc) => self.pc = pc,
                    v => panic!("Attempted to return into an invalid reference: {:?}", v),
                }
            }
            Opcode::Call => {
                let current_frame = self.frame_stack.last().unwrap();
                self.value_stack.push(Value::InternalRef(self.pc));
                self.value_stack
                    .push(Value::I32(current_frame.caller_module));
                self.value_stack
                    .push(Value::I32(current_frame.caller_module_internals));
                self.pc.func = inst.argument_data as usize;
                self.pc.inst = 0;
                self.pc.block_depth = 0;
            }
            Opcode::CrossModuleCall => {
                self.value_stack.push(Value::InternalRef(self.pc));
                self.value_stack.push(Value::I32(self.pc.module as u32));
                self.value_stack.push(Value::I32(module.internals_offset));
                let (func, module) = unpack_cross_module_call(inst.argument_data as u64);
                self.pc.module = module as usize;
                self.pc.func = func as usize;
                self.pc.inst = 0;
                self.pc.block_depth = 0;
            }
            Opcode::CallerModuleInternalCall => {
                self.value_stack.push(Value::InternalRef(self.pc));
                self.value_stack.push(Value::I32(self.pc.module as u32));
                self.value_stack.push(Value::I32(module.internals_offset));

                let current_frame = self.frame_stack.last().unwrap();
                if current_frame.caller_module_internals > 0 {
                    let func_idx = u32::try_from(inst.argument_data)
                        .ok()
                        .and_then(|o| current_frame.caller_module_internals.checked_add(o))
                        .expect("Internal call function index overflow");
                    self.pc.module = current_frame.caller_module as usize;
                    self.pc.func = func_idx as usize;
                    self.pc.inst = 0;
                    self.pc.block_depth = 0;
                } else {
                    // The caller module has no internals
                    self.status = MachineStatus::Errored;
                }
            }
            Opcode::CallIndirect => {
                let (table, ty) = crate::wavm::unpack_call_indirect(inst.argument_data);
                let idx = match self.value_stack.pop() {
                    Some(Value::I32(i)) => usize::try_from(i).unwrap(),
                    x => panic!(
                        "WASM validation failed: top of stack before call_indirect is {:?}",
                        x,
                    ),
                };
                let ty = &module.types[usize::try_from(ty).unwrap()];
                let elems = &module.tables[usize::try_from(table).unwrap()].elems;
                if let Some(elem) = elems.get(idx).filter(|e| &e.func_ty == ty) {
                    match elem.val {
                        Value::FuncRef(func) => {
                            let current_frame = self.frame_stack.last().unwrap();
                            self.value_stack.push(Value::InternalRef(self.pc));
                            self.value_stack
                                .push(Value::I32(current_frame.caller_module));
                            self.value_stack
                                .push(Value::I32(current_frame.caller_module_internals));
                            self.pc.func = func as usize;
                            self.pc.inst = 0;
                            self.pc.block_depth = 0;
                        }
                        Value::RefNull => {
                            self.status = MachineStatus::Errored;
                        }
                        v => panic!("Invalid table element value {:?}", v),
                    }
                } else {
                    self.status = MachineStatus::Errored;
                }
            }
            Opcode::LocalGet => {
                let val = self.frame_stack.last().unwrap().locals[inst.argument_data as usize];
                self.value_stack.push(val);
            }
            Opcode::LocalSet => {
                let val = self.value_stack.pop().unwrap();
                self.frame_stack.last_mut().unwrap().locals[inst.argument_data as usize] = val;
            }
            Opcode::GlobalGet => {
                self.value_stack
                    .push(module.globals[inst.argument_data as usize]);
            }
            Opcode::GlobalSet => {
                let val = self.value_stack.pop().unwrap();
                module.globals[inst.argument_data as usize] = val;
            }
            Opcode::MemoryLoad { ty, bytes, signed } => {
                let base = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x,
                    x => panic!(
                        "WASM validation failed: top of stack before memory load is {:?}",
                        x,
                    ),
                };
                if let Some(idx) = inst.argument_data.checked_add(base.into()) {
                    let val = module.memory.get_value(idx, ty, bytes, signed);
                    if let Some(val) = val {
                        self.value_stack.push(val);
                    } else {
                        self.status = MachineStatus::Errored;
                    }
                } else {
                    self.status = MachineStatus::Errored;
                }
            }
            Opcode::MemoryStore { ty: _, bytes } => {
                let val = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x.into(),
                    Some(Value::I64(x)) => x,
                    Some(Value::F32(x)) => x.to_bits().into(),
                    Some(Value::F64(x)) => x.to_bits(),
                    x => panic!(
                        "WASM validation failed: attempted to memory store type {:?}",
                        x,
                    ),
                };
                let base = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x,
                    x => panic!(
                        "WASM validation failed: attempted to memory store with index type {:?}",
                        x,
                    ),
                };
                if let Some(idx) = inst.argument_data.checked_add(base.into()) {
                    if !module.memory.store_value(idx, val, bytes) {
                        self.status = MachineStatus::Errored;
                    }
                } else {
                    self.status = MachineStatus::Errored;
                }
            }
            Opcode::I32Const => {
                self.value_stack.push(Value::I32(inst.argument_data as u32));
            }
            Opcode::I64Const => {
                self.value_stack.push(Value::I64(inst.argument_data));
            }
            Opcode::F32Const => {
                self.value_stack
                    .push(Value::F32(f32::from_bits(inst.argument_data as u32)));
            }
            Opcode::F64Const => {
                self.value_stack
                    .push(Value::F64(f64::from_bits(inst.argument_data)));
            }
            Opcode::FuncRefConst => {
                self.value_stack
                    .push(Value::FuncRef(inst.argument_data as u32));
            }
            Opcode::I32Eqz => {
                let val = self.value_stack.pop().unwrap();
                self.value_stack.push(Value::I32(val.is_i32_zero() as u32));
            }
            Opcode::I64Eqz => {
                let val = self.value_stack.pop().unwrap();
                self.value_stack.push(Value::I32(val.is_i64_zero() as u32));
            }
            Opcode::IRelOp(t, op, signed) => {
                let vb = self.value_stack.pop();
                let va = self.value_stack.pop();
                match t {
                    IntegerValType::I32 => {
                        if let (Some(Value::I32(a)), Some(Value::I32(b))) = (va, vb) {
                            if signed {
                                self.value_stack.push(exec_irel_op(a as i32, b as i32, op));
                            } else {
                                self.value_stack.push(exec_irel_op(a, b, op));
                            }
                        } else {
                            panic!("WASM validation failed: wrong types for i32relop");
                        }
                    }
                    IntegerValType::I64 => {
                        if let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) {
                            if signed {
                                self.value_stack.push(exec_irel_op(a as i64, b as i64, op));
                            } else {
                                self.value_stack.push(exec_irel_op(a, b, op));
                            }
                        } else {
                            panic!("WASM validation failed: wrong types for i64relop");
                        }
                    }
                }
            }
            Opcode::Drop => {
                self.value_stack.pop().unwrap();
            }
            Opcode::Select => {
                let selector_zero = self.value_stack.pop().unwrap().is_i32_zero();
                let val2 = self.value_stack.pop().unwrap();
                let val1 = self.value_stack.pop().unwrap();
                if selector_zero {
                    self.value_stack.push(val2);
                } else {
                    self.value_stack.push(val1);
                }
            }
            Opcode::MemorySize => {
                let pages = u32::try_from(module.memory.size() / Memory::PAGE_SIZE as u64)
                    .expect("Memory pages grew past a u32");
                self.value_stack.push(Value::I32(pages));
            }
            Opcode::MemoryGrow => {
                let old_size = module.memory.size();
                let adding_pages = match self.value_stack.pop() {
                    Some(Value::I32(x)) => x,
                    v => panic!("WASM validation failed: bad value for memory.grow {:?}", v),
                };
                let new_size = (|| {
                    let old_size = u64::try_from(old_size).ok()?;
                    let adding_size =
                        u64::from(adding_pages).checked_mul(Memory::PAGE_SIZE as u64)?;
                    let new_size = old_size.checked_add(adding_size)?;
                    // Note: we require the size remain *below* 2^32, meaning the actual limit is 2^32-PAGE_SIZE
                    if new_size < (1 << 32) {
                        Some(new_size)
                    } else {
                        None
                    }
                })();
                if let Some(new_size) = new_size {
                    module.memory.resize(usize::try_from(new_size).unwrap());
                    // Push the old number of pages
                    let old_pages = u32::try_from(old_size / Memory::PAGE_SIZE as u64).unwrap();
                    self.value_stack.push(Value::I32(old_pages));
                } else {
                    // Push -1
                    self.value_stack.push(Value::I32(u32::MAX));
                }
            }
            Opcode::IUnOp(w, op) => {
                let va = self.value_stack.pop();
                match w {
                    IntegerValType::I32 => {
                        if let Some(Value::I32(a)) = va {
                            self.value_stack.push(Value::I32(exec_iun_op(a, op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i32unop");
                        }
                    }
                    IntegerValType::I64 => {
                        if let Some(Value::I64(a)) = va {
                            self.value_stack.push(Value::I64(exec_iun_op(a, op) as u64));
                        } else {
                            panic!("WASM validation failed: wrong types for i64unop");
                        }
                    }
                }
            }
            Opcode::IBinOp(w, op) => {
                let vb = self.value_stack.pop();
                let va = self.value_stack.pop();
                match w {
                    IntegerValType::I32 => {
                        if let (Some(Value::I32(a)), Some(Value::I32(b))) = (va, vb) {
                            self.value_stack.push(Value::I32(exec_ibin_op(a, b, op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i32binop");
                        }
                    }
                    IntegerValType::I64 => {
                        if let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) {
                            self.value_stack.push(Value::I64(exec_ibin_op(a, b, op)));
                        } else {
                            panic!("WASM validation failed: wrong types for i64binop");
                        }
                    }
                }
            }
            Opcode::I32WrapI64 => {
                let x = match self.value_stack.pop() {
                    Some(Value::I64(x)) => x,
                    v => panic!(
                        "WASM validation failed: wrong type for i32.wrapi64: {:?}",
                        v,
                    ),
                };
                self.value_stack.push(Value::I32(x as u32));
            }
            Opcode::I64ExtendI32(signed) => {
                let x = self.value_stack.pop().unwrap().assume_u32();
                let x64 = match signed {
                    true => x as i32 as i64 as u64,
                    false => x as u32 as u64,
                };
                self.value_stack.push(Value::I64(x64));
            }
            Opcode::Reinterpret(dest, source) => {
                let val = match self.value_stack.pop() {
                    Some(Value::I32(x)) if source == ValueType::I32 => {
                        assert_eq!(dest, ValueType::F32, "Unsupported reinterpret");
                        Value::F32(f32::from_bits(x))
                    }
                    Some(Value::I64(x)) if source == ValueType::I64 => {
                        assert_eq!(dest, ValueType::F64, "Unsupported reinterpret");
                        Value::F64(f64::from_bits(x))
                    }
                    Some(Value::F32(x)) if source == ValueType::F32 => {
                        assert_eq!(dest, ValueType::I32, "Unsupported reinterpret");
                        Value::I32(x.to_bits())
                    }
                    Some(Value::F64(x)) if source == ValueType::F64 => {
                        assert_eq!(dest, ValueType::I64, "Unsupported reinterpret");
                        Value::I64(x.to_bits())
                    }
                    v => panic!("Bad reinterpret: val {:?} source {:?}", v, source),
                };
                self.value_stack.push(val);
            }
            Opcode::I32ExtendS(b) => {
                let mut x = self.value_stack.pop().unwrap().assume_u32();
                let mask = (1u32 << b) - 1;
                x &= mask;
                if x & (1 << (b - 1)) != 0 {
                    x |= !mask;
                }
                self.value_stack.push(Value::I32(x));
            }
            Opcode::I64ExtendS(b) => {
                let mut x = self.value_stack.pop().unwrap().assume_u64();
                let mask = (1u64 << b) - 1;
                x &= mask;
                if x & (1 << (b - 1)) != 0 {
                    x |= !mask;
                }
                self.value_stack.push(Value::I64(x));
            }
            Opcode::PushStackBoundary => {
                self.value_stack.push(Value::StackBoundary);
            }
            Opcode::MoveFromStackToInternal => {
                self.internal_stack.push(self.value_stack.pop().unwrap());
            }
            Opcode::MoveFromInternalToStack => {
                self.value_stack.push(self.internal_stack.pop().unwrap());
            }
            Opcode::IsStackBoundary => {
                let val = self.value_stack.pop().unwrap();
                self.value_stack
                    .push(Value::I32((val == Value::StackBoundary) as u32));
            }
            Opcode::Dup => {
                let val = self.value_stack.last().cloned().unwrap();
                self.value_stack.push(val);
            }
            Opcode::GetGlobalStateBytes32 => {
                let ptr = self.value_stack.pop().unwrap().assume_u32();
                let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                if idx > self.global_state.bytes32_vals.len() {
                    self.status = MachineStatus::Errored;
                } else if !module
                    .memory
                    .store_slice_aligned(ptr.into(), &*self.global_state.bytes32_vals[idx])
                {
                    self.status = MachineStatus::Errored;
                }
            }
            Opcode::SetGlobalStateBytes32 => {
                let ptr = self.value_stack.pop().unwrap().assume_u32();
                let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                if idx > self.global_state.bytes32_vals.len() {
                    self.status = MachineStatus::Errored;
                } else if let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) {
                    self.global_state.bytes32_vals[idx] = hash;
                } else {
                    self.status = MachineStatus::Errored;
                }
            }
            Opcode::GetGlobalStateU64 => {
                let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                if idx > self.global_state.u64_vals.len() {
                    self.status = MachineStatus::Errored;
                } else {
                    self.value_stack
                        .push(Value::I64(self.global_state.u64_vals[idx]));
                }
            }
            Opcode::SetGlobalStateU64 => {
                let val = self.value_stack.pop().unwrap().assume_u64();
                let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                if idx > self.global_state.u64_vals.len() {
                    self.status = MachineStatus::Errored;
                } else {
                    self.global_state.u64_vals[idx] = val
                }
            }
            Opcode::ReadPreImage => {
                let offset = self.value_stack.pop().unwrap().assume_u32();
                let ptr = self.value_stack.pop().unwrap().assume_u32();
                if let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) {
                    if let Some(preimage) = self.preimages.get(&hash) {
                        let offset = usize::try_from(offset).unwrap();
                        let len = std::cmp::min(32, preimage.len().saturating_sub(offset));
                        let read = preimage.get(offset..(offset + len)).unwrap_or_default();
                        let success = module.memory.store_slice_aligned(ptr.into(), read);
                        assert!(success, "Failed to write to previously read memory");
                        self.value_stack.push(Value::I32(len as u32));
                    } else {
                        panic!("Missing requested preimage for hash {}", hash);
                    }
                } else {
                    self.status = MachineStatus::Errored;
                }
            }
            Opcode::ReadInboxMessage => {
                let offset = self.value_stack.pop().unwrap().assume_u32();
                let ptr = self.value_stack.pop().unwrap().assume_u32();
                let msg_num = self.value_stack.pop().unwrap().assume_u64();
                if ptr as u64 + 32 > module.memory.size() {
                    self.status = MachineStatus::Errored;
                } else {
                    assert!(
                        inst.argument_data <= (InboxIdentifier::Delayed as u64),
                        "Bad inbox identifier"
                    );
                    let inbox_identifier = argument_data_to_inbox(inst.argument_data).unwrap();
                    if let Some(message) = self.inbox_contents.get(&(inbox_identifier, msg_num)) {
                        let offset = usize::try_from(offset).unwrap();
                        let len = std::cmp::min(32, message.len().saturating_sub(offset));
                        let read = message.get(offset..(offset + len)).unwrap_or_default();
                        if module.memory.store_slice_aligned(ptr.into(), read) {
                            self.value_stack.push(Value::I32(len as u32));
                        } else {
                            self.status = MachineStatus::Errored;
                        }
                    } else {
                        self.status = MachineStatus::TooFar;
                    }
                }
            }
            Opcode::HaltAndSetFinished => {
                self.status = MachineStatus::Finished;
            }
        }
    }

    fn host_call_hook(&mut self, module_name: &str, name: &str) -> eyre::Result<()> {
        let module = &mut self.modules[self.pc.module];
        macro_rules! pull_arg {
            ($offset:expr, $t:ident) => {
                self.value_stack
                    .get(self.value_stack.len().wrapping_sub($offset + 1))
                    .and_then(|v| match v {
                        Value::$t(x) => Some(*x),
                        _ => None,
                    })
                    .ok_or_else(|| eyre::eyre!("Exit code not on top of stack"))?
            };
        }
        macro_rules! read_u32_ptr {
            ($ptr:expr) => {
                module
                    .memory
                    .get_u32($ptr.into())
                    .ok_or_else(|| eyre::eyre!("Pointer out of bounds"))?
            };
        }
        macro_rules! read_bytes_segment {
            ($ptr:expr, $size:expr) => {
                module
                    .memory
                    .get_range($ptr as usize, $size as usize)
                    .ok_or_else(|| eyre::eyre!("Bytes segment out of bounds"))?
            };
        }
        match (module_name, name) {
            ("wasi_snapshot_preview1", "proc_exit") | ("env", "exit") => {
                let exit_code = pull_arg!(0, I32);
                println!(
                    "\x1b[31mWASM exiting\x1b[0m with exit code \x1b[31m{}\x1b[0m",
                    exit_code,
                );
                Ok(())
            }
            ("wasi_snapshot_preview1", "fd_write") => {
                let fd = pull_arg!(3, I32);
                if fd != 1 && fd != 2 {
                    // Not stdout or stderr, ignore
                    return Ok(());
                }
                let mut data = Vec::new();
                let iovecs_ptr = pull_arg!(2, I32);
                let iovecs_len = pull_arg!(1, I32);
                for offset in 0..iovecs_len {
                    let offset = offset.wrapping_mul(8);
                    let data_ptr_ptr = iovecs_ptr.wrapping_add(offset);
                    let data_size_ptr = data_ptr_ptr.wrapping_add(4);

                    let data_ptr = read_u32_ptr!(data_ptr_ptr);
                    let data_size = read_u32_ptr!(data_size_ptr);
                    data.extend_from_slice(read_bytes_segment!(data_ptr, data_size));
                }
                println!("WASM says: {:?}", String::from_utf8_lossy(&data));
                self.stdio_output.extend(data);
                Ok(())
            }
            _ => Ok(()),
        }
    }

    pub fn is_halted(&self) -> bool {
        self.status != MachineStatus::Running
    }

    pub fn get_status(&self) -> MachineStatus {
        self.status
    }

    fn get_modules_merkle(&self) -> Cow<Merkle> {
        if let Some(merkle) = &self.modules_merkle {
            Cow::Borrowed(merkle)
        } else {
            Cow::Owned(Merkle::new(
                MerkleType::Module,
                self.modules.iter().map(Module::hash).collect(),
            ))
        }
    }

    pub fn get_modules_root(&self) -> Bytes32 {
        self.get_modules_merkle().root()
    }

    pub fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        match self.status {
            MachineStatus::Running => {
                h.update(b"Machine running:");
                h.update(&hash_value_stack(&self.value_stack));
                h.update(&hash_value_stack(&self.internal_stack));
                h.update(&hash_pc_stack(&self.block_stack));
                h.update(hash_stack_frame_stack(&self.frame_stack));
                h.update(self.global_state.hash());
                h.update(&(self.pc.module as u32).to_be_bytes());
                h.update(&(self.pc.func as u32).to_be_bytes());
                h.update(&(self.pc.inst as u32).to_be_bytes());
                h.update(self.get_modules_root());
            }
            MachineStatus::Finished => {
                h.update("Machine finished:");
                h.update(self.global_state.hash());
            }
            MachineStatus::Errored => {
                h.update("Machine errored:");
            }
            MachineStatus::TooFar => {
                h.update("Machine too far:");
            }
        }
        h.finalize().into()
    }

    pub fn serialize_proof(&self) -> Vec<u8> {
        // Could be variable, but not worth it yet
        const STACK_PROVING_DEPTH: usize = 3;

        let mut data = Vec::new();

        data.push(self.status as u8);

        data.extend(prove_stack(
            &self.value_stack,
            STACK_PROVING_DEPTH,
            hash_value_stack,
            |v| v.serialize_for_proof(),
        ));

        data.extend(prove_stack(
            &self.internal_stack,
            1,
            hash_value_stack,
            |v| v.serialize_for_proof(),
        ));

        data.extend(prove_stack(&self.block_stack, 1, hash_pc_stack, |pc| {
            (*pc as u32).to_be_bytes()
        }));

        data.extend(prove_window(
            &self.frame_stack,
            hash_stack_frame_stack,
            StackFrame::serialize_for_proof,
        ));

        data.extend(self.global_state.hash());

        data.extend(&(self.pc.module as u32).to_be_bytes());
        data.extend(&(self.pc.func as u32).to_be_bytes());
        data.extend(&(self.pc.inst as u32).to_be_bytes());
        let mod_merkle = self.get_modules_merkle();
        data.extend(mod_merkle.root());

        // End machine serialization, serialize module

        let module = &self.modules[self.pc.module];
        let mem_merkle = module.memory.merkelize();
        data.extend(module.serialize_for_proof(&mem_merkle));

        // Prove module is in modules merkle tree

        data.extend(
            mod_merkle
                .prove(self.pc.module)
                .expect("Failed to prove module"),
        );

        if self.is_halted() {
            return data;
        }

        // Begin next instruction proof

        let func = &module.funcs[self.pc.func];
        data.extend(func.code[self.pc.inst].serialize_for_proof());
        data.extend(
            func.code_merkle
                .prove(self.pc.inst)
                .expect("Failed to prove against code merkle"),
        );
        data.extend(
            module
                .funcs_merkle
                .prove(self.pc.func)
                .expect("Failed to prove against function merkle"),
        );

        // End next instruction proof, begin instruction specific serialization

        if let Some(next_inst) = func.code.get(self.pc.inst) {
            if matches!(
                next_inst.opcode,
                Opcode::GetGlobalStateBytes32
                    | Opcode::SetGlobalStateBytes32
                    | Opcode::GetGlobalStateU64
                    | Opcode::SetGlobalStateU64
            ) {
                data.extend(self.global_state.serialize());
            }
            if matches!(next_inst.opcode, Opcode::LocalGet | Opcode::LocalSet) {
                let locals = &self.frame_stack.last().unwrap().locals;
                let idx = next_inst.argument_data as usize;
                data.extend(locals[idx].serialize_for_proof());
                let locals_merkle =
                    Merkle::new(MerkleType::Value, locals.iter().map(|v| v.hash()).collect());
                data.extend(
                    locals_merkle
                        .prove(idx)
                        .expect("Out of bounds local access"),
                );
            } else if matches!(next_inst.opcode, Opcode::GlobalGet | Opcode::GlobalSet) {
                let idx = next_inst.argument_data as usize;
                data.extend(module.globals[idx].serialize_for_proof());
                let locals_merkle = Merkle::new(
                    MerkleType::Value,
                    module.globals.iter().map(|v| v.hash()).collect(),
                );
                data.extend(
                    locals_merkle
                        .prove(idx)
                        .expect("Out of bounds global access"),
                );
            } else if matches!(
                next_inst.opcode,
                Opcode::MemoryLoad { .. } | Opcode::MemoryStore { .. },
            ) {
                let is_store = matches!(next_inst.opcode, Opcode::MemoryStore { .. });
                let stack_idx_offset = if is_store {
                    // The index is one item below the top stack item for a memory store
                    1
                } else {
                    0
                };
                let base = match self
                    .value_stack
                    .get(self.value_stack.len() - 1 - stack_idx_offset)
                {
                    Some(Value::I32(x)) => *x,
                    x => panic!("WASM validation failed: memory index type is {:?}", x),
                };
                if let Some(mut idx) = u64::from(base)
                    .checked_add(next_inst.argument_data)
                    .and_then(|x| usize::try_from(x).ok())
                {
                    // Prove the leaf this index is in, and the next one, if they are within the memory's size.
                    idx /= Memory::LEAF_SIZE;
                    data.extend(module.memory.get_leaf_data(idx));
                    data.extend(mem_merkle.prove(idx).unwrap_or_default());
                    // Now prove the next leaf too, in case it's accessed.
                    let next_leaf_idx = idx.saturating_add(1);
                    data.extend(module.memory.get_leaf_data(next_leaf_idx));
                    let second_mem_merkle = if is_store {
                        // For stores, prove the second merkle against a state after the first leaf is set.
                        // This state also happens to have the second leaf set, but that's irrelevant.
                        let mut copy = self.clone();
                        copy.step();
                        copy.modules[self.pc.module].memory.merkelize().into_owned()
                    } else {
                        mem_merkle.into_owned()
                    };
                    data.extend(second_mem_merkle.prove(next_leaf_idx).unwrap_or_default());
                }
            } else if next_inst.opcode == Opcode::CallIndirect {
                let (table, ty) = crate::wavm::unpack_call_indirect(next_inst.argument_data);
                let idx = match self.value_stack.last() {
                    Some(Value::I32(i)) => *i,
                    x => panic!(
                        "WASM validation failed: top of stack before call_indirect is {:?}",
                        x,
                    ),
                };
                let ty = &module.types[usize::try_from(ty).unwrap()];
                data.extend(&(table as u64).to_be_bytes());
                data.extend(ty.hash());
                let table_usize = usize::try_from(table).unwrap();
                let table = &module.tables[table_usize];
                data.extend(table.serialize_for_proof());
                data.extend(
                    module
                        .tables_merkle
                        .prove(table_usize)
                        .expect("Failed to prove tables merkle"),
                );
                let idx_usize = usize::try_from(idx).unwrap();
                if let Some(elem) = table.elems.get(idx_usize) {
                    data.extend(elem.func_ty.hash());
                    data.extend(elem.val.serialize_for_proof());
                    data.extend(
                        table
                            .elems_merkle
                            .prove(idx_usize)
                            .expect("Failed to prove elements merkle"),
                    );
                }
            } else if matches!(
                next_inst.opcode,
                Opcode::GetGlobalStateBytes32 | Opcode::SetGlobalStateBytes32,
            ) {
                let ptr = self.value_stack.last().unwrap().assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    // Prove the leaf this index is in
                    idx /= Memory::LEAF_SIZE;
                    data.extend(module.memory.get_leaf_data(idx));
                    data.extend(mem_merkle.prove(idx).unwrap_or_default());
                }
            } else if matches!(
                next_inst.opcode,
                Opcode::ReadPreImage | Opcode::ReadInboxMessage,
            ) {
                let ptr = self
                    .value_stack
                    .get(self.value_stack.len() - 2)
                    .unwrap()
                    .assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    // Prove the leaf this index is in
                    idx /= Memory::LEAF_SIZE;
                    let prev_data = module.memory.get_leaf_data(idx);
                    data.extend(prev_data);
                    data.extend(mem_merkle.prove(idx).unwrap_or_default());
                    if next_inst.opcode == Opcode::ReadPreImage {
                        let hash = Bytes32(prev_data);
                        let preimage = match self.preimages.get(&hash) {
                            Some(b) => b,
                            None => panic!("Missing requested preimage for hash {}", hash),
                        };
                        data.push(0); // preimage proof type
                        data.extend(preimage);
                    } else if next_inst.opcode == Opcode::ReadInboxMessage {
                        let msg_idx = self
                            .value_stack
                            .get(self.value_stack.len() - 3)
                            .unwrap()
                            .assume_u64();
                        let inbox_identifier =
                            argument_data_to_inbox(next_inst.argument_data).unwrap();
                        if let Some(msg_data) =
                            self.inbox_contents.get(&(inbox_identifier, msg_idx))
                        {
                            data.push(0); // inbox proof type
                            data.extend(msg_data);
                        }
                    } else {
                        panic!("Should never ever get here");
                    }
                }
            }
        }

        data
    }

    pub fn get_data_stack(&self) -> &[Value] {
        &self.value_stack
    }

    pub fn get_global_state(&self) -> GlobalState {
        self.global_state.clone()
    }

    pub fn set_global_state(&mut self, gs: GlobalState) {
        self.global_state = gs;
    }

    pub fn add_preimage(&mut self, key: Bytes32, val: Vec<u8>) {
        self.preimages.insert(key, val);
    }

    pub fn add_inbox_msg(&mut self, identifier: InboxIdentifier, index: u64, data: Vec<u8>) {
        self.inbox_contents.insert((identifier, index), data);
    }

    pub fn get_module_names(&self, module: usize) -> Option<&NameCustomSection> {
        self.modules.get(module).map(|m| &*m.names)
    }

    pub fn get_backtrace(&self) -> Vec<(String, String, usize)> {
        let mut res = Vec::new();
        let mut push_pc = |pc: ProgramCounter| {
            let names = &self.modules[pc.module].names;
            let func = names
                .functions
                .get(&(pc.func as u32))
                .cloned()
                .unwrap_or_else(|| format!("{}", pc.func));
            let mut module = names.module.clone();
            if module.is_empty() {
                module = format!("{}", pc.module);
            }
            res.push((module, func, pc.inst));
        };
        push_pc(self.pc);
        for frame in self.frame_stack.iter().rev() {
            match frame.return_ref {
                Value::InternalRef(pc) => {
                    push_pc(pc);
                }
                _ => {}
            }
        }
        res
    }

    pub fn get_stdio_output(&self) -> &[u8] {
        &self.stdio_output
    }
}
