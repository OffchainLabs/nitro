// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    binary::{parse, FloatInstruction, Local, NameCustomSection, WasmBinary},
    host,
    kzg::prove_kzg_preimage,
    memory::Memory,
    merkle::{Merkle, MerkleType},
    reinterpret::{ReinterpretAsSigned, ReinterpretAsUnsigned},
    utils::{file_bytes, Bytes32, CBytes, RemoteTableType},
    value::{ArbValueType, FunctionType, IntegerValType, ProgramCounter, Value},
    wavm::{
        pack_cross_module_call, unpack_cross_module_call, wasm_to_wavm, FloatingPointImpls,
        IBinOpType, IRelOpType, IUnOpType, Instruction, Opcode,
    },
};
use arbutil::{Color, PreimageType};
use c_kzg::BYTES_PER_BLOB;
use digest::Digest;
use eyre::{bail, ensure, eyre, Result, WrapErr};
use fnv::FnvHashMap as HashMap;
use num::{traits::PrimInt, Zero};
use rayon::prelude::*;
use serde::{Deserialize, Serialize};
use serde_with::serde_as;
use sha3::Keccak256;
use smallvec::SmallVec;
use std::{
    borrow::Cow,
    convert::{TryFrom, TryInto},
    fmt::{self, Display},
    fs::File,
    io::{BufReader, BufWriter, Write},
    num::Wrapping,
    path::{Path, PathBuf},
    sync::Arc,
};
use wasmparser::{DataKind, ElementItem, ElementKind, ExternalKind, Operator, TableType, TypeRef};

fn hash_call_indirect_data(table: u32, ty: &FunctionType) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update("Call indirect:");
    h.update((table as u64).to_be_bytes());
    h.update(ty.hash());
    h.finalize().into()
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum InboxIdentifier {
    Sequencer = 0,
    Delayed,
}

pub fn argument_data_to_inbox(argument_data: u64) -> Option<InboxIdentifier> {
    match argument_data {
        0x0 => Some(InboxIdentifier::Sequencer),
        0x1 => Some(InboxIdentifier::Delayed),
        _ => None,
    }
}

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct Function {
    code: Vec<Instruction>,
    ty: FunctionType,
    #[serde(skip)]
    code_merkle: Merkle,
    local_types: Vec<ArbValueType>,
}

impl Function {
    pub fn new<F: FnOnce(&mut Vec<Instruction>) -> Result<()>>(
        locals: &[Local],
        add_body: F,
        func_ty: FunctionType,
        module_types: &[FunctionType],
    ) -> Result<Function> {
        let mut locals_with_params = func_ty.inputs.clone();
        locals_with_params.extend(locals.iter().map(|x| x.value));

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

        add_body(&mut insts)?;
        insts.push(Instruction::simple(Opcode::Return));

        // Insert missing proving argument data
        for inst in insts.iter_mut() {
            if inst.opcode == Opcode::CallIndirect {
                let (table, ty) = crate::wavm::unpack_call_indirect(inst.argument_data);
                let ty = &module_types[usize::try_from(ty).unwrap()];
                inst.proving_argument_data = Some(hash_call_indirect_data(table, ty));
            }
        }

        Ok(Function::new_from_wavm(insts, func_ty, locals_with_params))
    }

    pub fn new_from_wavm(
        code: Vec<Instruction>,
        ty: FunctionType,
        local_types: Vec<ArbValueType>,
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
    locals: SmallVec<[Value; 16]>,
    caller_module: u32,
    caller_module_internals: u32,
}

impl StackFrame {
    fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Stack frame:");
        h.update(self.return_ref.hash());
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

#[derive(Clone, Debug, Serialize, Deserialize)]
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

#[serde_as]
#[derive(Clone, Debug, Serialize, Deserialize)]
struct Table {
    #[serde(with = "RemoteTableType")]
    ty: TableType,
    elems: Vec<TableElement>,
    #[serde(skip)]
    elems_merkle: Merkle,
}

impl Table {
    fn serialize_for_proof(&self) -> Result<Vec<u8>> {
        let mut data = vec![ArbValueType::try_from(self.ty.element_type)?.serialize()];
        data.extend((self.elems.len() as u64).to_be_bytes());
        data.extend(self.elems_merkle.root());
        Ok(data)
    }

    fn hash(&self) -> Result<Bytes32> {
        let mut h = Keccak256::new();
        h.update("Table:");
        h.update([ArbValueType::try_from(self.ty.element_type)?.serialize()]);
        h.update((self.elems.len() as u64).to_be_bytes());
        h.update(self.elems_merkle.root());
        Ok(h.finalize().into())
    }
}

#[derive(Clone, Debug)]
struct AvailableImport {
    ty: FunctionType,
    module: u32,
    func: u32,
}

#[derive(Clone, Debug, Default, Serialize, Deserialize)]
struct Module {
    globals: Vec<Value>,
    memory: Memory,
    tables: Vec<Table>,
    #[serde(skip)]
    tables_merkle: Merkle,
    funcs: Arc<Vec<Function>>,
    #[serde(skip)]
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
        bin: &WasmBinary,
        available_imports: &HashMap<String, AvailableImport>,
        floating_point_impls: &FloatingPointImpls,
        allow_hostapi: bool,
    ) -> Result<Module> {
        let mut code = Vec::new();
        let mut func_type_idxs: Vec<u32> = Vec::new();
        let mut memory = Memory::default();
        let mut exports = HashMap::default();
        let mut tables = Vec::new();
        let mut host_call_hooks = Vec::new();
        for import in &bin.imports {
            if let TypeRef::Func(ty) = import.ty {
                let mut qualified_name = format!("{}__{}", import.module, import.name);
                qualified_name = qualified_name.replace(&['/', '.'] as &[char], "_");
                let have_ty = &bin.types[ty as usize];
                let func;
                if let Some(import) = available_imports.get(&qualified_name) {
                    ensure!(
                        &import.ty == have_ty,
                        "Import has different function signature than host function. Expected {:?} but got {:?}",
                        import.ty, have_ty,
                    );
                    let wavm = vec![
                        Instruction::simple(Opcode::InitFrame),
                        Instruction::with_data(
                            Opcode::CrossModuleCall,
                            pack_cross_module_call(import.module, import.func),
                        ),
                        Instruction::simple(Opcode::Return),
                    ];
                    func = Function::new_from_wavm(wavm, import.ty.clone(), Vec::new());
                } else {
                    func = host::get_impl(import.module, import.name)?;
                    ensure!(
                        &func.ty == have_ty,
                        "Import has different function signature than host function. Expected {:?} but got {:?}",
                        func.ty, have_ty,
                    );
                    ensure!(
                        allow_hostapi,
                        "Calling hostapi directly is not allowed. Function {}",
                        import.name,
                    );
                }
                func_type_idxs.push(ty);
                code.push(func);
                host_call_hooks.push(Some((import.module.into(), import.name.into())));
            } else {
                bail!("Unsupport import kind {:?}", import);
            }
        }
        func_type_idxs.extend(bin.functions.iter());

        let internals = host::new_internal_funcs();
        let internals_offset = (code.len() + bin.codes.len()) as u32;
        let internals_types = internals.iter().map(|f| f.ty.clone());

        let mut types = bin.types.clone();
        let mut func_types: Vec<_> = func_type_idxs
            .iter()
            .map(|i| types[*i as usize].clone())
            .collect();

        func_types.extend(internals_types.clone());
        types.extend(internals_types);

        for c in &bin.codes {
            let idx = code.len();
            let func_ty = func_types[idx].clone();
            code.push(Function::new(
                &c.locals,
                |code| {
                    wasm_to_wavm(
                        &c.expr,
                        code,
                        floating_point_impls,
                        &func_types,
                        &types,
                        func_type_idxs[idx],
                        internals_offset,
                    )
                },
                func_ty.clone(),
                &types,
            )?);
        }
        code.extend(internals);
        ensure!(
            code.len() < (1usize << 31),
            "Module function count must be under 2^31",
        );

        ensure!(
            bin.memories.len() <= 1,
            "Multiple memories are not supported"
        );
        if let Some(limits) = bin.memories.first() {
            let page_size = Memory::PAGE_SIZE;
            let initial = limits.initial; // validate() checks this is less than max::u32
            let allowed = u32::MAX as u64 / Memory::PAGE_SIZE - 1; // we require the size remain *below* 2^32

            let max_size = match limits.maximum {
                Some(pages) => u64::min(allowed, pages),
                _ => allowed,
            };
            if initial > max_size {
                bail!(
                    "Memory inits to a size larger than its max: {} vs {}",
                    limits.initial,
                    max_size
                );
            }
            let size = initial * page_size;

            memory = Memory::new(size as usize, max_size);
        }

        let mut globals = vec![];
        for global in &bin.globals {
            let mut init = global.init_expr.get_operators_reader();

            let value = match (init.read()?, init.read()?, init.eof()) {
                (op, Operator::End, true) => crate::binary::op_as_const(op)?,
                _ => bail!("Non-constant global initializer"),
            };
            globals.push(value);
        }

        for export in &bin.exports {
            if let ExternalKind::Func = export.kind {
                exports.insert(export.name.to_owned(), export.index);
            }
        }

        for data in &bin.datas {
            let (memory_index, mut init) = match data.kind {
                DataKind::Active {
                    memory_index,
                    init_expr,
                } => (memory_index, init_expr.get_operators_reader()),
                _ => continue,
            };
            ensure!(
                memory_index == 0,
                "Attempted to write to nonexistant memory"
            );

            let offset = match (init.read()?, init.read()?, init.eof()) {
                (Operator::I32Const { value }, Operator::End, true) => value as usize,
                x => bail!("Non-constant element segment offset expression {:?}", x),
            };
            if !matches!(
                offset.checked_add(data.data.len()),
                Some(x) if (x as u64) <= memory.size(),
            ) {
                bail!(
                    "Out-of-bounds data memory init with offset {} and size {}",
                    offset,
                    data.data.len(),
                );
            }
            memory.set_range(offset, data.data);
        }

        for table in &bin.tables {
            tables.push(Table {
                elems: vec![TableElement::default(); usize::try_from(table.initial).unwrap()],
                ty: *table,
                elems_merkle: Merkle::default(),
            });
        }

        for elem in &bin.elements {
            let (t, mut init) = match elem.kind {
                ElementKind::Active {
                    table_index,
                    init_expr,
                } => (table_index, init_expr.get_operators_reader()),
                _ => continue,
            };
            let offset = match (init.read()?, init.read()?, init.eof()) {
                (Operator::I32Const { value }, Operator::End, true) => value as usize,
                x => bail!("Non-constant element segment offset expression {:?}", x),
            };
            let table = match tables.get_mut(t as usize) {
                Some(t) => t,
                None => bail!("Element segment for non-exsistent table {}", t),
            };
            let expected_ty = table.ty.element_type;
            ensure!(
                expected_ty == elem.ty,
                "Element type expected to be of table type {:?} but of type {:?}",
                expected_ty,
                elem.ty
            );

            let mut contents = vec![];
            let mut item_reader = elem.items.get_items_reader()?;
            for _ in 0..item_reader.get_count() {
                let item = item_reader.read()?;
                let index = match item {
                    ElementItem::Func(index) => index,
                    ElementItem::Expr(_) => {
                        bail!("Non-constant element initializers are not supported")
                    }
                };
                let func_ty = func_types[index as usize].clone();
                contents.push(TableElement {
                    val: Value::FuncRef(index),
                    func_ty,
                })
            }

            let len = contents.len();
            ensure!(
                offset.saturating_add(len) <= table.elems.len(),
                "Out of bounds element segment at offset {} and length {} for table of length {}",
                offset,
                len,
                table.elems.len(),
            );
            table.elems[offset..][..len].clone_from_slice(&contents);
        }

        let tables_hashes: Result<_, _> = tables.iter().map(Table::hash).collect();

        Ok(Module {
            memory,
            globals,
            tables_merkle: Merkle::new(MerkleType::Table, tables_hashes?),
            tables,
            funcs_merkle: Arc::new(Merkle::new(
                MerkleType::Function,
                code.iter().map(|f| f.hash()).collect(),
            )),
            funcs: Arc::new(code),
            types: Arc::new(types.to_owned()),
            internals_offset,
            names: Arc::new(bin.names.to_owned()),
            host_call_hooks: Arc::new(host_call_hooks),
            start_function: bin.start,
            func_types: Arc::new(func_types),
            exports: Arc::new(exports),
        })
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
        data.extend(self.memory.max_size.to_be_bytes());
        data.extend(mem_merkle.root());

        data.extend(self.tables_merkle.root());
        data.extend(self.funcs_merkle.root());

        data.extend(self.internals_offset.to_be_bytes());

        data
    }
}

// Globalstate holds:
// bytes32 - last_block_hash
// bytes32 - send_root
// uint64 - inbox_position
// uint64 - position_within_message
// uint64 - espresso hotshot height
pub const GLOBAL_STATE_BYTES32_NUM: usize = 2;
pub const GLOBAL_STATE_U64_NUM: usize = 3;

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

#[derive(Serialize)]
pub struct ProofInfo {
    pub before: String,
    pub proof: String,
    pub after: String,
}

impl ProofInfo {
    pub fn new(before: String, proof: String, after: String) -> Self {
        Self {
            before,
            proof,
            after,
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

impl Display for MachineStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Running => write!(f, "running"),
            Self::Finished => write!(f, "finished"),
            Self::Errored => write!(f, "errored"),
            Self::TooFar => write!(f, "too far"),
        }
    }
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
    frame_stack: Cow<'a, Vec<StackFrame>>,
    modules: Vec<ModuleState<'a>>,
    global_state: GlobalState,
    pc: ProgramCounter,
    stdio_output: Cow<'a, Vec<u8>>,
    initial_hash: Bytes32,
}

pub type PreimageResolver = Arc<dyn Fn(u64, PreimageType, Bytes32) -> Option<CBytes> + Send + Sync>;

/// Wraps a preimage resolver to provide an easier API
/// and cache the last preimage retrieved.
#[derive(Clone)]
struct PreimageResolverWrapper {
    resolver: PreimageResolver,
    last_resolved: Option<(Bytes32, CBytes)>,
}

impl fmt::Debug for PreimageResolverWrapper {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "resolver...")
    }
}

impl PreimageResolverWrapper {
    pub fn new(resolver: PreimageResolver) -> PreimageResolverWrapper {
        PreimageResolverWrapper {
            resolver,
            last_resolved: None,
        }
    }

    pub fn get(&mut self, context: u64, ty: PreimageType, hash: Bytes32) -> Option<&[u8]> {
        // TODO: this is unnecessarily complicated by the rust borrow checker.
        // This will probably be simplifiable when Polonius is shipped.
        if matches!(&self.last_resolved, Some(r) if r.0 != hash) {
            self.last_resolved = None;
        }
        match &mut self.last_resolved {
            Some(resolved) => Some(&resolved.1),
            x => {
                let data = (self.resolver)(context, ty, hash)?;
                Some(&x.insert((hash, data)).1)
            }
        }
    }

    pub fn get_const(&self, context: u64, ty: PreimageType, hash: Bytes32) -> Option<CBytes> {
        if let Some(resolved) = &self.last_resolved {
            if resolved.0 == hash {
                return Some(resolved.1.clone());
            }
        }
        (self.resolver)(context, ty, hash)
    }
}

#[derive(Clone, Debug)]
pub struct Machine {
    steps: u64, // Not part of machine hash
    status: MachineStatus,
    value_stack: Vec<Value>,
    internal_stack: Vec<Value>,
    frame_stack: Vec<StackFrame>,
    modules: Vec<Module>,
    modules_merkle: Option<Merkle>,
    global_state: GlobalState,
    pc: ProgramCounter,
    stdio_output: Vec<u8>,
    inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
    hotshot_commitments: HashMap<u64, [u8; 32]>,
    first_too_far: u64, // Not part of machine hash
    preimage_resolver: PreimageResolverWrapper,
    initial_hash: Bytes32,
    context: u64,
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
        h.update(hash);
        hash = h.finalize().into();
    }
    hash
}

fn hash_value_stack(stack: &[Value]) -> Bytes32 {
    hash_stack(stack.iter().map(|v| v.hash()), "Value stack:")
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
fn exec_ibin_op<T>(a: T, b: T, op: IBinOpType) -> Option<T>
where
    Wrapping<T>: ReinterpretAsSigned,
    T: Zero,
{
    let a = Wrapping(a);
    let b = Wrapping(b);
    if matches!(
        op,
        IBinOpType::DivS | IBinOpType::DivU | IBinOpType::RemS | IBinOpType::RemU,
    ) && b.is_zero()
    {
        return None;
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
    Some(res.0)
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

pub fn get_empty_preimage_resolver() -> PreimageResolver {
    Arc::new(|_, _, _| None) as _
}

impl Machine {
    pub const MAX_STEPS: u64 = 1 << 43;

    pub fn from_paths(
        library_paths: &[PathBuf],
        binary_path: &Path,
        language_support: bool,
        always_merkleize: bool,
        allow_hostapi_from_main: bool,
        global_state: GlobalState,
        inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
        preimage_resolver: PreimageResolver,
    ) -> Result<Machine> {
        let bin_source = file_bytes(binary_path)?;
        let bin = parse(&bin_source)
            .wrap_err_with(|| format!("failed to validate WASM binary at {:?}", binary_path))?;
        let mut libraries = vec![];
        let mut lib_sources = vec![];
        for path in library_paths {
            let error_message = format!("failed to validate WASM binary at {:?}", path);
            lib_sources.push((file_bytes(path)?, error_message));
        }
        for (source, error_message) in &lib_sources {
            let library = parse(source).wrap_err_with(|| error_message.clone())?;
            libraries.push(library);
        }
        Self::from_binaries(
            &libraries,
            bin,
            language_support,
            always_merkleize,
            allow_hostapi_from_main,
            global_state,
            inbox_contents,
            preimage_resolver,
        )
    }

    pub fn from_binaries(
        libraries: &[WasmBinary<'_>],
        bin: WasmBinary<'_>,
        runtime_support: bool,
        always_merkleize: bool,
        allow_hostapi_from_main: bool,
        global_state: GlobalState,
        inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
        preimage_resolver: PreimageResolver,
    ) -> Result<Machine> {
        use ArbValueType::*;

        // `modules` starts out with the entrypoint module, which will be initialized later
        let mut modules = vec![Module::default()];
        let mut available_imports = HashMap::default();
        let mut floating_point_impls = HashMap::default();

        for export in &bin.exports {
            if let ExternalKind::Func = export.kind {
                if let Some(ty_idx) = usize::try_from(export.index)
                    .unwrap()
                    .checked_sub(bin.imports.len())
                {
                    let ty = bin.functions[ty_idx];
                    let ty = &bin.types[usize::try_from(ty).unwrap()];
                    let module = u32::try_from(modules.len() + libraries.len()).unwrap();
                    available_imports.insert(
                        format!("env__wavm_guest_call__{}", export.name),
                        AvailableImport {
                            ty: ty.clone(),
                            module,
                            func: export.index,
                        },
                    );
                }
            }
        }

        for lib in libraries {
            let module = Module::from_binary(lib, &available_imports, &floating_point_impls, true)?;
            for (name, &func) in &*module.exports {
                let ty = module.func_types[func as usize].clone();
                available_imports.insert(
                    name.clone(),
                    AvailableImport {
                        module: modules.len() as u32,
                        func,
                        ty: ty.clone(),
                    },
                );
                if let Ok(op) = name.parse::<FloatInstruction>() {
                    let mut sig = op.signature();
                    // wavm codegen takes care of effecting this type change at callsites
                    for ty in sig.inputs.iter_mut().chain(sig.outputs.iter_mut()) {
                        if *ty == F32 {
                            *ty = I32;
                        } else if *ty == F64 {
                            *ty = I64;
                        }
                    }
                    ensure!(
                        ty == sig,
                        "Wrong type for floating point impl {:?} expecting {:?} but got {:?}",
                        name,
                        sig,
                        ty
                    );
                    floating_point_impls.insert(op, (modules.len() as u32, func));
                }
            }
            modules.push(module);
        }

        // Shouldn't be necessary, but to safe, don't allow the main binary to import its own guest calls
        available_imports.retain(|_, i| i.module as usize != modules.len());
        modules.push(Module::from_binary(
            &bin,
            &available_imports,
            &floating_point_impls,
            allow_hostapi_from_main,
        )?);

        // Build the entrypoint module
        let mut entrypoint = Vec::new();
        macro_rules! entry {
            ($opcode:ident) => {
                entrypoint.push(Instruction::simple(Opcode::$opcode));
            };
            ($opcode:ident, $value:expr) => {
                entrypoint.push(Instruction::with_data(Opcode::$opcode, $value));
            };
            ($opcode:ident ($inside:expr)) => {
                entrypoint.push(Instruction::simple(Opcode::$opcode($inside)));
            };
            (@cross, $module:expr, $func:expr) => {
                entrypoint.push(Instruction::with_data(
                    Opcode::CrossModuleCall,
                    pack_cross_module_call($module, $func),
                ));
            };
        }
        for (i, module) in modules.iter().enumerate() {
            if let Some(s) = module.start_function {
                ensure!(
                    module.func_types[s as usize] == FunctionType::default(),
                    "Start function takes inputs or outputs",
                );
                entry!(@cross, u32::try_from(i).unwrap(), s);
            }
        }
        let main_module_idx = modules.len() - 1;
        let main_module = &modules[main_module_idx];

        // Rust support
        let rust_fn = "__main_void";
        if let Some(&f) = main_module.exports.get(rust_fn).filter(|_| runtime_support) {
            let expected_type = FunctionType::new(vec![], vec![I32]);
            ensure!(
                main_module.func_types[f as usize] == expected_type,
                "Main function doesn't match expected signature of [] -> [ret]",
            );
            entry!(@cross, u32::try_from(main_module_idx).unwrap(), f);
            entry!(Drop);
            entry!(HaltAndSetFinished);
        }

        // Go support
        if let Some(&f) = main_module.exports.get("run").filter(|_| runtime_support) {
            let mut expected_type = FunctionType::default();
            expected_type.inputs.push(I32); // argc
            expected_type.inputs.push(I32); // argv
            ensure!(
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
            ensure!(
                main_module.internals_offset != 0,
                "Main module doesn't have internals"
            );
            let main_module_idx = u32::try_from(main_module_idx).unwrap();
            let main_module_store32 = main_module.internals_offset + 3;

            // Write "js\0" to name_str_ptr, to match what the actual JS environment does
            entry!(I32Const, name_str_ptr);
            entry!(I32Const, 0x736a); // b"js\0"
            entry!(@cross, main_module_idx, main_module_store32);
            entry!(I32Const, name_str_ptr + 4);
            entry!(I32Const, 0);
            entry!(@cross, main_module_idx, main_module_store32);

            // Write name_str_ptr to argv_ptr
            entry!(I32Const, argv_ptr);
            entry!(I32Const, name_str_ptr);
            entry!(@cross, main_module_idx, main_module_store32);
            entry!(I32Const, argv_ptr + 4);
            entry!(I32Const, 0);
            entry!(@cross, main_module_idx, main_module_store32);

            // Launch main with an argument count of 1 and argv_ptr
            entry!(I32Const, 1);
            entry!(I32Const, argv_ptr);
            entry!(@cross, main_module_idx, f);
            if let Some(i) = available_imports.get("wavm__go_after_run") {
                ensure!(
                    i.ty == FunctionType::default(),
                    "Resume function has non-empty function signature",
                );
                entry!(@cross, i.module, i.func);
            }
        }

        let entrypoint_types = vec![FunctionType::default()];
        let mut entrypoint_names = NameCustomSection {
            module: "entry".into(),
            functions: HashMap::default(),
        };
        entrypoint_names
            .functions
            .insert(0, "wavm_entrypoint".into());
        let entrypoint_funcs = vec![Function::new(
            &[],
            |code| {
                code.extend(entrypoint);
                Ok(())
            },
            FunctionType::default(),
            &entrypoint_types,
        )?];
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
            host_call_hooks: Arc::new(Vec::new()),
            start_function: None,
            func_types: Arc::new(vec![FunctionType::default()]),
            exports: Arc::new(HashMap::default()),
        };
        modules[0] = entrypoint;

        ensure!(
            u32::try_from(modules.len()).is_ok(),
            "module count doesn't fit in a u32",
        );

        // Merkleize things if requested
        for module in &mut modules {
            for table in module.tables.iter_mut() {
                table.elems_merkle = Merkle::new(
                    MerkleType::TableElement,
                    table.elems.iter().map(TableElement::hash).collect(),
                );
            }

            let tables_hashes: Result<_, _> = module.tables.iter().map(Table::hash).collect();
            module.tables_merkle = Merkle::new(MerkleType::Table, tables_hashes?);

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

        // find the first inbox index that's out of bounds
        let first_too_far = inbox_contents
            .iter()
            .filter(|((kind, _), _)| kind == &InboxIdentifier::Sequencer)
            .map(|((_, index), _)| *index + 1)
            .max()
            .unwrap_or(0);

        let mut mach = Machine {
            status: MachineStatus::Running,
            steps: 0,
            value_stack: vec![Value::RefNull, Value::I32(0), Value::I32(0)],
            internal_stack: Vec::new(),
            frame_stack: Vec::new(),
            modules,
            modules_merkle,
            global_state,
            pc: ProgramCounter::default(),
            stdio_output: Vec::new(),
            inbox_contents,
            first_too_far,
            preimage_resolver: PreimageResolverWrapper::new(preimage_resolver),
            initial_hash: Bytes32::default(),
            context: 0,
            hotshot_commitments: Default::default(),
        };
        mach.initial_hash = mach.hash();
        Ok(mach)
    }

    pub fn new_from_wavm(wavm_binary: &Path) -> Result<Machine> {
        let f = BufReader::new(File::open(wavm_binary)?);
        let decompressor = brotli2::read::BrotliDecoder::new(f);
        let mut modules: Vec<Module> = bincode::deserialize_from(decompressor)?;
        for module in modules.iter_mut() {
            for table in module.tables.iter_mut() {
                table.elems_merkle = Merkle::new(
                    MerkleType::TableElement,
                    table.elems.iter().map(TableElement::hash).collect(),
                );
            }
            let tables: Result<_> = module.tables.iter().map(Table::hash).collect();
            module.tables_merkle = Merkle::new(MerkleType::Table, tables?);

            let funcs =
                Arc::get_mut(&mut module.funcs).expect("Multiple copies of module functions");
            for func in funcs.iter_mut() {
                func.code_merkle = Merkle::new(
                    MerkleType::Instruction,
                    func.code.par_iter().map(|i| i.hash()).collect(),
                );
            }
            module.funcs_merkle = Arc::new(Merkle::new(
                MerkleType::Function,
                module.funcs.iter().map(Function::hash).collect(),
            ));
        }
        let mut mach = Machine {
            status: MachineStatus::Running,
            steps: 0,
            value_stack: vec![Value::RefNull, Value::I32(0), Value::I32(0)],
            internal_stack: Vec::new(),
            frame_stack: Vec::new(),
            modules,
            modules_merkle: None,
            global_state: Default::default(),
            pc: ProgramCounter::default(),
            stdio_output: Vec::new(),
            inbox_contents: Default::default(),
            first_too_far: 0,
            preimage_resolver: PreimageResolverWrapper::new(get_empty_preimage_resolver()),
            initial_hash: Bytes32::default(),
            context: 0,
            hotshot_commitments: Default::default(),
        };
        mach.initial_hash = mach.hash();
        Ok(mach)
    }

    pub fn serialize_binary<P: AsRef<Path>>(&self, path: P) -> Result<()> {
        ensure!(
            self.hash() == self.initial_hash,
            "serialize_binary can only be called on initial machine",
        );
        let mut f = File::create(path)?;
        let mut compressor = brotli2::write::BrotliEncoder::new(BufWriter::new(&mut f), 9);
        bincode::serialize_into(&mut compressor, &self.modules)?;
        compressor.flush()?;
        drop(compressor);
        f.sync_data()?;
        Ok(())
    }

    pub fn serialize_state<P: AsRef<Path>>(&self, path: P) -> Result<()> {
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
    pub fn deserialize_and_replace_state<P: AsRef<Path>>(&mut self, path: P) -> Result<()> {
        let reader = BufReader::new(File::open(path)?);
        let new_state: MachineState = bincode::deserialize_from(reader)?;
        if self.initial_hash != new_state.initial_hash {
            bail!(
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
        self.frame_stack = new_state.frame_stack.into_owned();
        self.global_state = new_state.global_state;
        self.pc = new_state.pc;
        self.stdio_output = new_state.stdio_output.into_owned();
        Ok(())
    }

    pub fn start_merkle_caching(&mut self) {
        for module in &mut self.modules {
            module.memory.cache_merkle_tree();
        }
        self.modules_merkle = Some(Merkle::new(
            MerkleType::Module,
            self.modules.iter().map(Module::hash).collect(),
        ));
    }

    pub fn stop_merkle_caching(&mut self) {
        self.modules_merkle = None;
        for module in &mut self.modules {
            module.memory.merkle = None;
        }
    }

    pub fn jump_into_function(&mut self, func: &str, mut args: Vec<Value>) {
        let frame_args = [Value::RefNull, Value::I32(0), Value::I32(0)];
        args.extend(frame_args);
        self.value_stack = args;

        let module = self.modules.last().expect("no module");
        let export = module.exports.iter().find(|x| x.0 == func);
        let export = export
            .unwrap_or_else(|| panic!("func {} not found", func))
            .1;

        self.frame_stack.clear();
        self.internal_stack.clear();

        self.pc = ProgramCounter {
            module: (self.modules.len() - 1).try_into().unwrap(),
            func: *export,
            inst: 0,
        };
        self.status = MachineStatus::Running;
        self.steps = 0;
    }

    pub fn get_final_result(&self) -> Result<Vec<Value>> {
        if !self.frame_stack.is_empty() {
            bail!(
                "machine has not successfully computed a final result {:?}",
                self.status
            )
        }
        Ok(self.value_stack.clone())
    }

    pub fn get_next_instruction(&self) -> Option<Instruction> {
        if self.is_halted() {
            return None;
        }
        self.modules[self.pc.module()].funcs[self.pc.func()]
            .code
            .get(self.pc.inst())
            .cloned()
    }

    pub fn next_instruction_is_host_io(&self) -> bool {
        self.get_next_instruction()
            .map(|i| i.opcode.is_host_io())
            .unwrap_or(true)
    }

    pub fn next_instruction_is_read_hotshot(&self) -> bool {
        self.get_next_instruction()
            .map(|i| i.opcode == Opcode::ReadHotShotCommitment)
            .unwrap_or(true)
    }

    pub fn get_pc(&self) -> Option<ProgramCounter> {
        if self.is_halted() {
            return None;
        }
        Some(self.pc)
    }

    fn test_next_instruction(func: &Function, pc: &ProgramCounter) {
        debug_assert!(func.code.len() > pc.inst.try_into().unwrap());
    }

    pub fn get_steps(&self) -> u64 {
        self.steps
    }

    pub fn step_n(&mut self, n: u64) -> Result<()> {
        if self.is_halted() {
            return Ok(());
        }
        let mut module = &mut self.modules[self.pc.module()];
        let mut func = &module.funcs[self.pc.func()];

        macro_rules! flush_module {
            () => {
                if let Some(merkle) = self.modules_merkle.as_mut() {
                    merkle.set(self.pc.module(), module.hash());
                }
            };
        }
        macro_rules! error {
            () => {{
                self.status = MachineStatus::Errored;
                break;
            }};
        }

        for _ in 0..n {
            self.steps += 1;
            if self.steps == Self::MAX_STEPS {
                error!();
            }
            let inst = func.code[self.pc.inst()];
            self.pc.inst += 1;
            match inst.opcode {
                Opcode::Unreachable => error!(),
                Opcode::Nop => {}
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
                    if let Some(hook) = module
                        .host_call_hooks
                        .get(self.pc.func())
                        .and_then(|h| h.as_ref())
                    {
                        if let Err(err) = Self::host_call_hook(
                            &self.value_stack,
                            module,
                            &mut self.stdio_output,
                            &hook.0,
                            &hook.1,
                        ) {
                            eprintln!(
                                "Failed to process host call hook for host call {:?} {:?}: {}",
                                hook.0, hook.1, err,
                            );
                        }
                    }
                }
                Opcode::ArbitraryJump => {
                    self.pc.inst = inst.argument_data as u32;
                    Machine::test_next_instruction(func, &self.pc);
                }
                Opcode::ArbitraryJumpIf => {
                    let x = self.value_stack.pop().unwrap();
                    if !x.is_i32_zero() {
                        self.pc.inst = inst.argument_data as u32;
                        Machine::test_next_instruction(func, &self.pc);
                    }
                }
                Opcode::Return => {
                    let frame = self.frame_stack.pop().unwrap();
                    match frame.return_ref {
                        Value::RefNull => error!(),
                        Value::InternalRef(pc) => {
                            let changing_module = pc.module != self.pc.module;
                            if changing_module {
                                flush_module!();
                            }
                            self.pc = pc;
                            if changing_module {
                                module = &mut self.modules[self.pc.module()];
                            }
                            func = &module.funcs[self.pc.func()];
                        }
                        v => bail!("attempted to return into an invalid reference: {:?}", v),
                    }
                }
                Opcode::Call => {
                    let current_frame = self.frame_stack.last().unwrap();
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack
                        .push(Value::I32(current_frame.caller_module));
                    self.value_stack
                        .push(Value::I32(current_frame.caller_module_internals));
                    self.pc.func = inst.argument_data as u32;
                    self.pc.inst = 0;
                    func = &module.funcs[self.pc.func()];
                }
                Opcode::CrossModuleCall => {
                    flush_module!();
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack.push(Value::I32(self.pc.module));
                    self.value_stack.push(Value::I32(module.internals_offset));
                    let (call_module, call_func) = unpack_cross_module_call(inst.argument_data);
                    self.pc.module = call_module;
                    self.pc.func = call_func;
                    self.pc.inst = 0;
                    module = &mut self.modules[self.pc.module()];
                    func = &module.funcs[self.pc.func()];
                }
                Opcode::CallerModuleInternalCall => {
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack.push(Value::I32(self.pc.module));
                    self.value_stack.push(Value::I32(module.internals_offset));

                    let current_frame = self.frame_stack.last().unwrap();
                    if current_frame.caller_module_internals > 0 {
                        let func_idx = u32::try_from(inst.argument_data)
                            .ok()
                            .and_then(|o| current_frame.caller_module_internals.checked_add(o))
                            .expect("Internal call function index overflow");
                        flush_module!();
                        self.pc.module = current_frame.caller_module;
                        self.pc.func = func_idx;
                        self.pc.inst = 0;
                        module = &mut self.modules[self.pc.module()];
                        func = &module.funcs[self.pc.func()];
                    } else {
                        // The caller module has no internals
                        error!();
                    }
                }
                Opcode::CallIndirect => {
                    let (table, ty) = crate::wavm::unpack_call_indirect(inst.argument_data);
                    let idx = match self.value_stack.pop() {
                        Some(Value::I32(i)) => usize::try_from(i).unwrap(),
                        x => bail!(
                            "WASM validation failed: top of stack before call_indirect is {:?}",
                            x,
                        ),
                    };
                    let ty = &module.types[usize::try_from(ty).unwrap()];
                    let elems = &module.tables[usize::try_from(table).unwrap()].elems;
                    if let Some(elem) = elems.get(idx).filter(|e| &e.func_ty == ty) {
                        match elem.val {
                            Value::FuncRef(call_func) => {
                                let current_frame = self.frame_stack.last().unwrap();
                                self.value_stack.push(Value::InternalRef(self.pc));
                                self.value_stack
                                    .push(Value::I32(current_frame.caller_module));
                                self.value_stack
                                    .push(Value::I32(current_frame.caller_module_internals));
                                self.pc.func = call_func;
                                self.pc.inst = 0;
                                func = &module.funcs[self.pc.func()];
                            }
                            Value::RefNull => error!(),
                            v => bail!("invalid table element value {:?}", v),
                        }
                    } else {
                        error!();
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
                        x => bail!(
                            "WASM validation failed: top of stack before memory load is {:?}",
                            x,
                        ),
                    };
                    if let Some(idx) = inst.argument_data.checked_add(base.into()) {
                        let val = module.memory.get_value(idx, ty, bytes, signed);
                        if let Some(val) = val {
                            self.value_stack.push(val);
                        } else {
                            error!();
                        }
                    } else {
                        error!();
                    }
                }
                Opcode::MemoryStore { ty: _, bytes } => {
                    let val = match self.value_stack.pop() {
                        Some(Value::I32(x)) => x.into(),
                        Some(Value::I64(x)) => x,
                        Some(Value::F32(x)) => x.to_bits().into(),
                        Some(Value::F64(x)) => x.to_bits(),
                        x => bail!(
                            "WASM validation failed: attempted to memory store type {:?}",
                            x,
                        ),
                    };
                    let base = match self.value_stack.pop() {
                        Some(Value::I32(x)) => x,
                        x => bail!(
                            "WASM validation failed: attempted to memory store with index type {:?}",
                            x,
                        ),
                    };
                    if let Some(idx) = inst.argument_data.checked_add(base.into()) {
                        if !module.memory.store_value(idx, val, bytes) {
                            error!();
                        }
                    } else {
                        error!();
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
                                bail!("WASM validation failed: wrong types for i32relop");
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
                                bail!("WASM validation failed: wrong types for i64relop");
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
                    let pages = u32::try_from(module.memory.size() / Memory::PAGE_SIZE)
                        .expect("Memory pages grew past a u32");
                    self.value_stack.push(Value::I32(pages));
                }
                Opcode::MemoryGrow => {
                    let old_size = module.memory.size();
                    let adding_pages = match self.value_stack.pop() {
                        Some(Value::I32(x)) => x,
                        v => bail!("WASM validation failed: bad value for memory.grow {:?}", v),
                    };
                    let page_size = Memory::PAGE_SIZE;
                    let max_size = module.memory.max_size * page_size;

                    let new_size = (|| {
                        let adding_size = u64::from(adding_pages).checked_mul(page_size)?;
                        let new_size = old_size.checked_add(adding_size)?;
                        if new_size <= max_size {
                            Some(new_size)
                        } else {
                            None
                        }
                    })();
                    if let Some(new_size) = new_size {
                        module.memory.resize(usize::try_from(new_size).unwrap());
                        // Push the old number of pages
                        let old_pages = u32::try_from(old_size / page_size).unwrap();
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
                                bail!("WASM validation failed: wrong types for i32unop");
                            }
                        }
                        IntegerValType::I64 => {
                            if let Some(Value::I64(a)) = va {
                                self.value_stack.push(Value::I64(exec_iun_op(a, op) as u64));
                            } else {
                                bail!("WASM validation failed: wrong types for i64unop");
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
                                if op == IBinOpType::DivS
                                    && (a as i32) == i32::MIN
                                    && (b as i32) == -1
                                {
                                    error!();
                                }
                                let value = match exec_ibin_op(a, b, op) {
                                    Some(value) => value,
                                    None => error!(),
                                };
                                self.value_stack.push(Value::I32(value))
                            } else {
                                bail!("WASM validation failed: wrong types for i32binop");
                            }
                        }
                        IntegerValType::I64 => {
                            if let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) {
                                if op == IBinOpType::DivS
                                    && (a as i64) == i64::MIN
                                    && (b as i64) == -1
                                {
                                    error!();
                                }
                                let value = match exec_ibin_op(a, b, op) {
                                    Some(value) => value,
                                    None => error!(),
                                };
                                self.value_stack.push(Value::I64(value))
                            } else {
                                bail!("WASM validation failed: wrong types for i64binop");
                            }
                        }
                    }
                }
                Opcode::I32WrapI64 => {
                    let x = match self.value_stack.pop() {
                        Some(Value::I64(x)) => x,
                        v => bail!(
                            "WASM validation failed: wrong type for i32.wrapi64: {:?}",
                            v,
                        ),
                    };
                    self.value_stack.push(Value::I32(x as u32));
                }
                Opcode::I64ExtendI32(signed) => {
                    let x: u32 = self.value_stack.pop().unwrap().assume_u32();
                    let x64 = match signed {
                        true => x as i32 as i64 as u64,
                        false => x as u64,
                    };
                    self.value_stack.push(Value::I64(x64));
                }
                Opcode::Reinterpret(dest, source) => {
                    let val = match self.value_stack.pop() {
                        Some(Value::I32(x)) if source == ArbValueType::I32 => {
                            assert_eq!(dest, ArbValueType::F32, "Unsupported reinterpret");
                            Value::F32(f32::from_bits(x))
                        }
                        Some(Value::I64(x)) if source == ArbValueType::I64 => {
                            assert_eq!(dest, ArbValueType::F64, "Unsupported reinterpret");
                            Value::F64(f64::from_bits(x))
                        }
                        Some(Value::F32(x)) if source == ArbValueType::F32 => {
                            assert_eq!(dest, ArbValueType::I32, "Unsupported reinterpret");
                            Value::I32(x.to_bits())
                        }
                        Some(Value::F64(x)) if source == ArbValueType::F64 => {
                            assert_eq!(dest, ArbValueType::I64, "Unsupported reinterpret");
                            Value::I64(x.to_bits())
                        }
                        v => bail!("bad reinterpret: val {:?} source {:?}", v, source),
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
                Opcode::MoveFromStackToInternal => {
                    self.internal_stack.push(self.value_stack.pop().unwrap());
                }
                Opcode::MoveFromInternalToStack => {
                    self.value_stack.push(self.internal_stack.pop().unwrap());
                }
                Opcode::Dup => {
                    let val = self.value_stack.last().cloned().unwrap();
                    self.value_stack.push(val);
                }
                Opcode::GetGlobalStateBytes32 => {
                    let ptr = self.value_stack.pop().unwrap().assume_u32();
                    let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.bytes32_vals.len()
                        || !module
                            .memory
                            .store_slice_aligned(ptr.into(), &*self.global_state.bytes32_vals[idx])
                    {
                        error!();
                    }
                }
                Opcode::SetGlobalStateBytes32 => {
                    let ptr = self.value_stack.pop().unwrap().assume_u32();
                    let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.bytes32_vals.len() {
                        error!();
                    } else if let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) {
                        self.global_state.bytes32_vals[idx] = hash;
                    } else {
                        error!();
                    }
                }
                Opcode::GetGlobalStateU64 => {
                    let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.u64_vals.len() {
                        error!();
                    } else {
                        self.value_stack
                            .push(Value::I64(self.global_state.u64_vals[idx]));
                    }
                }
                Opcode::SetGlobalStateU64 => {
                    let val = self.value_stack.pop().unwrap().assume_u64();
                    let idx = self.value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.u64_vals.len() {
                        error!();
                    } else {
                        self.global_state.u64_vals[idx] = val
                    }
                }
                Opcode::ReadPreImage => {
                    let offset = self.value_stack.pop().unwrap().assume_u32();
                    let ptr = self.value_stack.pop().unwrap().assume_u32();
                    let preimage_ty = PreimageType::try_from(u8::try_from(inst.argument_data)?)?;
                    // Preimage reads must be word aligned
                    if offset % 32 != 0 {
                        error!();
                    }
                    if let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) {
                        if let Some(preimage) =
                            self.preimage_resolver.get(self.context, preimage_ty, hash)
                        {
                            if preimage_ty == PreimageType::EthVersionedHash
                                && preimage.len() != BYTES_PER_BLOB
                            {
                                bail!(
                                    "kzg hash {} preimage should be {} bytes long but is instead {}",
                                    hash,
                                    BYTES_PER_BLOB,
                                    preimage.len(),
                                );
                            }
                            let offset = usize::try_from(offset).unwrap();
                            let len = std::cmp::min(32, preimage.len().saturating_sub(offset));
                            let read = preimage.get(offset..(offset + len)).unwrap_or_default();
                            let success = module.memory.store_slice_aligned(ptr.into(), read);
                            assert!(success, "Failed to write to previously read memory");
                            self.value_stack.push(Value::I32(len as u32));
                        } else {
                            eprintln!(
                                "{} for hash {}",
                                "Missing requested preimage".red(),
                                hash.red(),
                            );
                            self.eprint_backtrace();
                            bail!("missing requested preimage for hash {}", hash);
                        }
                    } else {
                        error!();
                    }
                }
                Opcode::ReadHotShotCommitment => {
                    let height = self.value_stack.pop().unwrap().assume_u64();
                    let ptr = self.value_stack.pop().unwrap().assume_u32();
                    if let Some(commitment) = self.hotshot_commitments.get(&height) {
                        if ptr as u64 + 32 > module.memory.size() {
                            error!();
                        } else {
                            let success = module.memory.store_slice_aligned(ptr.into(), commitment);
                            assert!(success, "Failed to write to previously read memory");
                        }
                    } else {
                        error!()
                    }
                }
                Opcode::ReadInboxMessage => {
                    let offset = self.value_stack.pop().unwrap().assume_u32();
                    let ptr = self.value_stack.pop().unwrap().assume_u32();
                    let msg_num = self.value_stack.pop().unwrap().assume_u64();
                    let inbox_identifier =
                        argument_data_to_inbox(inst.argument_data).expect("Bad inbox indentifier");
                    if let Some(message) = self.inbox_contents.get(&(inbox_identifier, msg_num)) {
                        if ptr as u64 + 32 > module.memory.size() {
                            error!();
                        } else {
                            let offset = usize::try_from(offset).unwrap();
                            let len = std::cmp::min(32, message.len().saturating_sub(offset));
                            let read = message.get(offset..(offset + len)).unwrap_or_default();
                            if module.memory.store_slice_aligned(ptr.into(), read) {
                                self.value_stack.push(Value::I32(len as u32));
                            } else {
                                error!();
                            }
                        }
                    } else {
                        let delayed = inbox_identifier == InboxIdentifier::Delayed;
                        if msg_num < self.first_too_far || delayed {
                            eprintln!("{} {msg_num}", "Missing inbox message".red());
                            self.eprint_backtrace();
                            bail!(
                                "missing inbox message {msg_num} of {}",
                                self.first_too_far - 1
                            );
                        }
                        self.status = MachineStatus::TooFar;
                        break;
                    }
                }
                Opcode::HaltAndSetFinished => {
                    self.status = MachineStatus::Finished;
                    break;
                }
            }
        }
        flush_module!();
        if self.is_halted() && !self.stdio_output.is_empty() {
            // If we halted, print out any trailing output that didn't have a newline.
            println!(
                "{} {}",
                "WASM says:".yellow(),
                String::from_utf8_lossy(&self.stdio_output),
            );
            self.stdio_output.clear();
        }
        Ok(())
    }

    fn host_call_hook(
        value_stack: &[Value],
        module: &Module,
        stdio_output: &mut Vec<u8>,
        module_name: &str,
        name: &str,
    ) -> Result<()> {
        macro_rules! pull_arg {
            ($offset:expr, $t:ident) => {
                value_stack
                    .get(value_stack.len().wrapping_sub($offset + 1))
                    .and_then(|v| match v {
                        Value::$t(x) => Some(*x),
                        _ => None,
                    })
                    .ok_or_else(|| eyre!("exit code not on top of stack"))?
            };
        }
        macro_rules! read_u32_ptr {
            ($ptr:expr) => {
                module
                    .memory
                    .get_u32($ptr.into())
                    .ok_or_else(|| eyre!("pointer out of bounds"))?
            };
        }
        macro_rules! read_bytes_segment {
            ($ptr:expr, $size:expr) => {
                module
                    .memory
                    .get_range($ptr as usize, $size as usize)
                    .ok_or_else(|| eyre!("bytes segment out of bounds"))?
            };
        }
        match (module_name, name) {
            ("wasi_snapshot_preview1", "proc_exit") | ("env", "exit") => {
                let exit_code = pull_arg!(0, I32);
                if exit_code != 0 {
                    println!(
                        "\x1b[31mWASM exiting\x1b[0m with exit code \x1b[31m{}\x1b[0m",
                        exit_code,
                    );
                }
                Ok(())
            }
            ("wasi_snapshot_preview1", "fd_write") => {
                let fd = pull_arg!(3, I32);
                if fd != 1 && fd != 2 {
                    // Not stdout or stderr, ignore
                    return Ok(());
                }
                let iovecs_ptr = pull_arg!(2, I32);
                let iovecs_len = pull_arg!(1, I32);
                for offset in 0..iovecs_len {
                    let offset = offset.wrapping_mul(8);
                    let data_ptr_ptr = iovecs_ptr.wrapping_add(offset);
                    let data_size_ptr = data_ptr_ptr.wrapping_add(4);

                    let data_ptr = read_u32_ptr!(data_ptr_ptr);
                    let data_size = read_u32_ptr!(data_size_ptr);
                    stdio_output.extend_from_slice(read_bytes_segment!(data_ptr, data_size));
                }
                while let Some(mut idx) = stdio_output.iter().position(|&c| c == b'\n') {
                    println!(
                        "\x1b[33mWASM says:\x1b[0m {}",
                        String::from_utf8_lossy(&stdio_output[..idx]),
                    );
                    if stdio_output.get(idx + 1) == Some(&b'\r') {
                        idx += 1;
                    }
                    *stdio_output = stdio_output.split_off(idx + 1);
                }
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
                h.update(hash_value_stack(&self.value_stack));
                h.update(hash_value_stack(&self.internal_stack));
                h.update(hash_stack_frame_stack(&self.frame_stack));
                h.update(self.global_state.hash());
                h.update(self.pc.module.to_be_bytes());
                h.update(self.pc.func.to_be_bytes());
                h.update(self.pc.inst.to_be_bytes());
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

        let mut data = vec![self.status as u8];

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

        data.extend(prove_window(
            &self.frame_stack,
            hash_stack_frame_stack,
            StackFrame::serialize_for_proof,
        ));

        data.extend(self.global_state.hash());

        data.extend(self.pc.module.to_be_bytes());
        data.extend(self.pc.func.to_be_bytes());
        data.extend(self.pc.inst.to_be_bytes());
        let mod_merkle = self.get_modules_merkle();
        data.extend(mod_merkle.root());

        // End machine serialization, serialize module

        let module = &self.modules[self.pc.module()];
        let mem_merkle = module.memory.merkelize();
        data.extend(module.serialize_for_proof(&mem_merkle));

        // Prove module is in modules merkle tree

        data.extend(
            mod_merkle
                .prove(self.pc.module())
                .expect("Failed to prove module"),
        );

        if self.is_halted() {
            return data;
        }

        // Begin next instruction proof

        let func = &module.funcs[self.pc.func()];
        data.extend(func.code[self.pc.inst()].serialize_for_proof());
        data.extend(
            func.code_merkle
                .prove(self.pc.inst())
                .expect("Failed to prove against code merkle"),
        );
        data.extend(
            module
                .funcs_merkle
                .prove(self.pc.func())
                .expect("Failed to prove against function merkle"),
        );

        // End next instruction proof, begin instruction specific serialization

        if let Some(next_inst) = func.code.get(self.pc.inst()) {
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
                // this isn't really a bool -> int, it's determining an offset based on a bool
                #[allow(clippy::bool_to_int_with_if)]
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
                        copy.step_n(1)
                            .expect("Failed to step machine forward for proof");
                        copy.modules[self.pc.module()]
                            .memory
                            .merkelize()
                            .into_owned()
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
                data.extend((table as u64).to_be_bytes());
                data.extend(ty.hash());
                let table_usize = usize::try_from(table).unwrap();
                let table = &module.tables[table_usize];
                data.extend(
                    table
                        .serialize_for_proof()
                        .expect("failed to serialize table"),
                );
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
                let offset = self.value_stack.last().unwrap().assume_u32();
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
                        let preimage_ty = PreimageType::try_from(
                            u8::try_from(next_inst.argument_data)
                                .expect("ReadPreImage argument data is out of range for a u8"),
                        )
                        .expect("Invalid preimage type in ReadPreImage argument data");
                        let preimage =
                            match self
                                .preimage_resolver
                                .get_const(self.context, preimage_ty, hash)
                            {
                                Some(b) => b,
                                None => panic!("Missing requested preimage for hash {}", hash),
                            };
                        data.push(0); // preimage proof type
                        match preimage_ty {
                            PreimageType::Keccak256 | PreimageType::Sha2_256 => {
                                // The proofs for these preimage types are just the raw preimages.
                                data.extend(preimage);
                            }
                            PreimageType::EthVersionedHash => {
                                prove_kzg_preimage(hash, &preimage, offset, &mut data)
                                    .expect("Failed to generate KZG preimage proof");
                            }
                        }
                    } else if next_inst.opcode == Opcode::ReadInboxMessage {
                        let msg_idx = self
                            .value_stack
                            .get(self.value_stack.len() - 3)
                            .unwrap()
                            .assume_u64();
                        let inbox_identifier = argument_data_to_inbox(next_inst.argument_data)
                            .expect("Bad inbox indentifier");
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
            } else if matches!(next_inst.opcode, Opcode::ReadHotShotCommitment) {
                let ptr = self.value_stack.get(0).unwrap().assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    idx /= Memory::LEAF_SIZE;
                    let prev_data = module.memory.get_leaf_data(idx);
                    data.extend(prev_data);
                    data.extend(mem_merkle.prove(idx).unwrap_or_default());

                    let h = self.value_stack.get(1).unwrap().assume_u64();
                    if let Some(commitment) = self.hotshot_commitments.get(&h) {
                        data.extend(commitment);
                        println!("read hotshot commitment proof generated. height: {:?}, commitment: {:?}", h, commitment);
                    }
                } else {
                    panic!("Should never ever get here")
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

    pub fn set_preimage_resolver(&mut self, resolver: PreimageResolver) {
        self.preimage_resolver.resolver = resolver;
    }

    pub fn set_context(&mut self, context: u64) {
        self.context = context;
    }

    pub fn add_inbox_msg(&mut self, identifier: InboxIdentifier, index: u64, data: Vec<u8>) {
        self.inbox_contents.insert((identifier, index), data);
        if index >= self.first_too_far && identifier == InboxIdentifier::Sequencer {
            self.first_too_far = index + 1
        }
    }

    pub fn add_hotshot_commitment(&mut self, height: u64, commitment: [u8; 32]) {
        self.hotshot_commitments.insert(height, commitment);
    }

    pub fn get_module_names(&self, module: usize) -> Option<&NameCustomSection> {
        self.modules.get(module).map(|m| &*m.names)
    }

    pub fn get_backtrace(&self) -> Vec<(String, String, usize)> {
        let mut res = Vec::new();
        let mut push_pc = |pc: ProgramCounter| {
            let names = &self.modules[pc.module()].names;
            let func = names
                .functions
                .get(&pc.func)
                .cloned()
                .unwrap_or_else(|| format!("{}", pc.func));
            let mut module = names.module.clone();
            if module.is_empty() {
                module = format!("{}", pc.module);
            }
            res.push((module, func, pc.inst()));
        };
        push_pc(self.pc);
        for frame in self.frame_stack.iter().rev() {
            if let Value::InternalRef(pc) = frame.return_ref {
                push_pc(pc);
            }
        }
        res
    }

    pub fn eprint_backtrace(&self) {
        eprintln!("Backtrace:");
        for (module, func, pc) in self.get_backtrace() {
            let func = rustc_demangle::demangle(&func);
            eprintln!("  {} {} @ {}", module, func.mint(), pc.blue());
        }
    }
}
