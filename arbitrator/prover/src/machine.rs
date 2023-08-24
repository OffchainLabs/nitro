// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    binary::{
        self, parse, ExportKind, ExportMap, FloatInstruction, Local, NameCustomSection, WasmBinary,
    },
    host,
    memory::Memory,
    merkle::{Merkle, MerkleType},
    programs::{
        config::{CompileConfig, WasmPricingInfo},
        meter::MeteredMachine,
        ModuleMod, StylusData, STYLUS_ENTRY_POINT,
    },
    reinterpret::{ReinterpretAsSigned, ReinterpretAsUnsigned},
    utils::{file_bytes, CBytes, RemoteTableType},
    value::{ArbValueType, FunctionType, IntegerValType, ProgramCounter, Value},
    wavm::{
        pack_cross_module_call, unpack_cross_module_call, wasm_to_wavm, FloatingPointImpls,
        IBinOpType, IRelOpType, IUnOpType, Instruction, Opcode,
    },
};
use arbutil::{math, Bytes32, Color};
use digest::Digest;
use eyre::{bail, ensure, eyre, Result, WrapErr};
use fnv::FnvHashMap as HashMap;
use itertools::izip;
use num::{traits::PrimInt, Zero};
use serde::{Deserialize, Serialize};
use serde_with::serde_as;
use sha3::Keccak256;
use smallvec::SmallVec;
use std::{
    borrow::Cow,
    convert::{TryFrom, TryInto},
    fmt::{self, Display},
    fs::File,
    hash::Hash,
    io::{BufReader, BufWriter, Write},
    num::Wrapping,
    path::{Path, PathBuf},
    sync::Arc,
};
use wasmer_types::FunctionIndex;
use wasmparser::{DataKind, ElementItem, ElementKind, Operator, TableType};

#[cfg(feature = "rayon")]
use rayon::prelude::*;

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

        #[cfg(feature = "rayon")]
        let code_hashes = code.par_iter().map(|i| i.hash()).collect();

        #[cfg(not(feature = "rayon"))]
        let code_hashes = code.iter().map(|i| i.hash()).collect();

        Function {
            code,
            ty,
            code_merkle: Merkle::new(MerkleType::Instruction, code_hashes),
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

impl AvailableImport {
    pub fn new(ty: FunctionType, module: u32, func: u32) -> Self {
        Self { ty, module, func }
    }
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
    /// Old modules use this format.
    /// TODO: remove this after the jump to stylus.
    #[serde(alias = "exports")]
    func_exports: Arc<HashMap<String, u32>>,
    #[serde(default)]
    all_exports: Arc<ExportMap>,
}

impl Module {
    const FORWARDING_PREFIX: &str = "arbitrator_forward__";

    fn from_binary(
        bin: &WasmBinary,
        available_imports: &HashMap<String, AvailableImport>,
        floating_point_impls: &FloatingPointImpls,
        allow_hostapi: bool,
        debug_funcs: bool,
        stylus_data: Option<StylusData>,
    ) -> Result<Module> {
        let mut code = Vec::new();
        let mut func_type_idxs: Vec<u32> = Vec::new();
        let mut memory = Memory::default();
        let mut tables = Vec::new();
        let mut host_call_hooks = Vec::new();
        for import in &bin.imports {
            let module = import.module;
            let have_ty = &bin.types[import.offset as usize];
            let Some(import_name) = import.name else {
                bail!("Missing name for import in {}", module.red());
            };
            let (forward, import_name) = match import_name.strip_prefix(Module::FORWARDING_PREFIX) {
                Some(name) => (true, name),
                None => (false, import_name),
            };

            let mut qualified_name = format!("{module}__{import_name}");
            qualified_name = qualified_name.replace(&['/', '.'] as &[char], "_");

            let func = if let Some(import) = available_imports.get(&qualified_name) {
                let call = match forward {
                    true => Opcode::CrossModuleForward,
                    false => Opcode::CrossModuleCall,
                };
                let wavm = vec![
                    Instruction::simple(Opcode::InitFrame),
                    Instruction::with_data(
                        call,
                        pack_cross_module_call(import.module, import.func),
                    ),
                    Instruction::simple(Opcode::Return),
                ];
                Function::new_from_wavm(wavm, import.ty.clone(), vec![])
            } else if let Ok((hostio, debug)) = host::get_impl(import.module, import_name) {
                ensure!(
                    (debug && debug_funcs) || (!debug && allow_hostapi),
                    "Host func {} in {} not enabled debug_funcs={debug_funcs} hostapi={allow_hostapi} debug={debug}",
                    import_name.red(),
                    import.module.red(),
                );
                hostio
            } else {
                bail!(
                    "No such import {} in {}",
                    import_name.red(),
                    import.module.red()
                )
            };
            ensure!(
                &func.ty == have_ty,
                "Import {} has different function signature than host function. Expected {} but got {}",
                import_name.red(), func.ty.red(), have_ty.red(),
            );

            func_type_idxs.push(import.offset);
            code.push(func);
            host_call_hooks.push(Some((import.module.into(), import_name.into())));
        }
        func_type_idxs.extend(bin.functions.iter());

        let internals = host::new_internal_funcs(stylus_data);
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
        if let Some(limits) = bin.memories.get(0) {
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
                    limits.initial.red(),
                    max_size.red()
                );
            }
            let size = initial * page_size;

            memory = Memory::new(size as usize, max_size);
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
            memory.set_range(offset, data.data)?;
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
            let Some(table) = tables.get_mut(t as usize) else {
                bail!("Element segment for non-exsistent table {}", t)
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
                let ElementItem::Func(index) = item else {
                    bail!("Non-constant element initializers are not supported")
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
        ensure!(
            code.len() < (1usize << 31),
            "Module function count must be under 2^31",
        );
        ensure!(!code.is_empty(), "Module has no code");

        let tables_hashes: Result<_, _> = tables.iter().map(Table::hash).collect();
        let func_exports = bin
            .exports
            .iter()
            .filter_map(|(name, (offset, kind))| {
                (kind == &ExportKind::Func).then(|| (name.to_owned(), *offset))
            })
            .collect();

        Ok(Module {
            memory,
            globals: bin.globals.clone(),
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
            func_exports: Arc::new(func_exports),
            all_exports: Arc::new(bin.exports.clone()),
        })
    }

    fn name(&self) -> &str {
        &self.names.module
    }

    fn find_func(&self, name: &str) -> Result<u32> {
        let Some(func) = self.func_exports.iter().find(|x| x.0 == name) else {
            bail!("func {} not found in {}", name.red(), self.name().red())
        };
        Ok(*func.1)
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
pub const GLOBAL_STATE_BYTES32_NUM: usize = 2;
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

pub type PreimageResolver = Arc<dyn Fn(u64, Bytes32) -> Option<CBytes>>;

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

    pub fn get(&mut self, context: u64, hash: Bytes32) -> Option<&[u8]> {
        // TODO: this is unnecessarily complicated by the rust borrow checker.
        // This will probably be simplifiable when Polonius is shipped.
        if matches!(&self.last_resolved, Some(r) if r.0 != hash) {
            self.last_resolved = None;
        }
        match &mut self.last_resolved {
            Some(resolved) => Some(&resolved.1),
            x => {
                let data = (self.resolver)(context, hash)?;
                Some(&x.insert((hash, data)).1)
            }
        }
    }

    pub fn get_const(&self, context: u64, hash: Bytes32) -> Option<CBytes> {
        if let Some(resolved) = &self.last_resolved {
            if resolved.0 == hash {
                return Some(resolved.1.clone());
            }
        }
        (self.resolver)(context, hash)
    }
}

#[derive(Clone, Debug)]
pub struct ErrorGuard {
    frame_stack: usize,
    value_stack: usize,
    inter_stack: usize,
    on_error: ProgramCounter,
}

impl Display for ErrorGuard {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            f,
            "{}{} {} {} {} {}{}",
            "ErrorGuard(".grey(),
            self.frame_stack.mint(),
            self.value_stack.mint(),
            self.inter_stack.mint(),
            "→".grey(),
            self.on_error,
            ")".grey(),
        )
    }
}

#[derive(Clone, Debug)]
struct ErrorGuardProof {
    frame_stack: Bytes32,
    value_stack: Bytes32,
    inter_stack: Bytes32,
    on_error: ProgramCounter,
}

impl ErrorGuardProof {
    const STACK_PREFIX: &str = "Guard stack:";
    const GUARD_PREFIX: &str = "Error guard:";

    fn new(
        frame_stack: Bytes32,
        value_stack: Bytes32,
        inter_stack: Bytes32,
        on_error: ProgramCounter,
    ) -> Self {
        Self {
            frame_stack,
            value_stack,
            inter_stack,
            on_error,
        }
    }

    fn serialize_for_proof(&self) -> Vec<u8> {
        let mut data = self.frame_stack.to_vec();
        data.extend(self.value_stack.0);
        data.extend(self.inter_stack.0);
        data.extend(Value::from(self.on_error).serialize_for_proof());
        data
    }

    fn hash(&self) -> Bytes32 {
        Keccak256::new()
            .chain(Self::GUARD_PREFIX)
            .chain(self.frame_stack)
            .chain(self.value_stack)
            .chain(self.inter_stack)
            .chain(Value::InternalRef(self.on_error).hash())
            .finalize()
            .into()
    }

    fn hash_guards(guards: &[Self]) -> Bytes32 {
        hash_stack(guards.iter().map(|g| g.hash()), Self::STACK_PREFIX)
    }
}

#[derive(Clone, Debug)]
pub struct Machine {
    steps: u64, // Not part of machine hash
    status: MachineStatus,
    value_stack: Vec<Value>,
    internal_stack: Vec<Value>,
    frame_stack: Vec<StackFrame>,
    guards: Vec<ErrorGuard>,
    modules: Vec<Module>,
    modules_merkle: Option<Merkle>,
    global_state: GlobalState,
    pc: ProgramCounter,
    stdio_output: Vec<u8>,
    inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
    first_too_far: u64, // Not part of machine hash
    preimage_resolver: PreimageResolverWrapper,
    stylus_modules: HashMap<Bytes32, Module>, // Not part of machine hash
    initial_hash: Bytes32,
    context: u64,
    debug_info: bool, // Not part of machine hash
}

type FrameStackHash = Bytes32;
type ValueStackHash = Bytes32;
type InterStackHash = Bytes32;

fn hash_stack<I, D>(stack: I, prefix: &str) -> Bytes32
where
    I: IntoIterator<Item = D>,
    D: AsRef<[u8]>,
{
    hash_stack_with_heights(stack, &[], prefix).0
}

/// Hashes a stack of n elements, returning the values at various heights along the way in O(n).
fn hash_stack_with_heights<I, D>(
    stack: I,
    mut heights: &[usize],
    prefix: &str,
) -> (Bytes32, Vec<Bytes32>)
where
    I: IntoIterator<Item = D>,
    D: AsRef<[u8]>,
{
    let mut parts = vec![];
    let mut hash = Bytes32::default();
    let mut count = 0;
    for item in stack.into_iter() {
        while heights.first() == Some(&count) {
            parts.push(hash);
            heights = &heights[1..];
        }

        hash = Keccak256::new()
            .chain(prefix)
            .chain(item.as_ref())
            .chain(hash)
            .finalize()
            .into();

        count += 1;
    }
    while !heights.is_empty() {
        assert_eq!(heights[0], count);
        parts.push(hash);
        heights = &heights[1..];
    }
    (hash, parts)
}

fn hash_value_stack(stack: &[Value]) -> ValueStackHash {
    hash_stack(stack.iter().map(|v| v.hash()), "Value stack:")
}

fn hash_stack_frame_stack(frames: &[StackFrame]) -> FrameStackHash {
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
    Arc::new(|_, _| None) as _
}

impl Machine {
    pub const MAX_STEPS: u64 = 1 << 43;

    pub fn from_paths(
        library_paths: &[PathBuf],
        binary_path: &Path,
        language_support: bool,
        always_merkleize: bool,
        allow_hostapi_from_main: bool,
        debug_funcs: bool,
        debug_info: bool,
        global_state: GlobalState,
        inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
        preimage_resolver: PreimageResolver,
    ) -> Result<Machine> {
        let bin_source = file_bytes(binary_path)?;
        let bin = parse(&bin_source, binary_path)
            .wrap_err_with(|| format!("failed to validate WASM binary at {:?}", binary_path))?;
        let mut libraries = vec![];
        let mut lib_sources = vec![];
        for path in library_paths {
            let error_message = format!("failed to validate WASM binary at {:?}", path);
            lib_sources.push((file_bytes(path)?, path, error_message));
        }
        for (source, path, error_message) in &lib_sources {
            let library = parse(source, path).wrap_err_with(|| error_message.clone())?;
            libraries.push(library);
        }
        Self::from_binaries(
            &libraries,
            bin,
            language_support,
            always_merkleize,
            allow_hostapi_from_main,
            debug_funcs,
            debug_info,
            global_state,
            inbox_contents,
            preimage_resolver,
            None,
        )
    }

    /// Creates an instrumented user Machine from the wasm or wat at the given `path`.
    #[cfg(feature = "native")]
    pub fn from_user_path(path: &Path, compile: &CompileConfig) -> Result<Self> {
        let data = std::fs::read(path)?;
        let wasm = wasmer::wat2wasm(&data)?;
        let mut bin = binary::parse(&wasm, Path::new("user"))?;
        let stylus_data = bin.instrument(compile)?;

        let user_test = std::fs::read("../../target/machines/latest/user_test.wasm")?;
        let user_test = parse(&user_test, Path::new("user_test"))?;
        let wasi_stub = std::fs::read("../../target/machines/latest/wasi_stub.wasm")?;
        let wasi_stub = parse(&wasi_stub, Path::new("wasi_stub"))?;
        let soft_float = std::fs::read("../../target/machines/latest/soft-float.wasm")?;
        let soft_float = parse(&soft_float, Path::new("soft-float"))?;

        let mut machine = Self::from_binaries(
            &[soft_float, wasi_stub, user_test],
            bin,
            false,
            false,
            false,
            compile.debug.debug_funcs,
            true,
            GlobalState::default(),
            HashMap::default(),
            Arc::new(|_, _| panic!("tried to read preimage")),
            Some(stylus_data),
        )?;

        let footprint: u32 = stylus_data.footprint.into();
        machine.call_function("user_test", "set_pages", vec![footprint.into()])?;
        Ok(machine)
    }

    /// Produces a compile-only `Machine` from a user program.
    /// Note: the machine's imports are stubbed, so execution isn't possible.
    pub fn new_user_stub(
        wasm: &[u8],
        page_limit: u16,
        version: u16,
        debug: bool,
    ) -> Result<(Machine, WasmPricingInfo)> {
        let compile = CompileConfig::version(version, debug);
        let forward = include_bytes!("../../../target/machines/latest/forward_stub.wasm");
        let forward = binary::parse(forward, Path::new("forward")).unwrap();

        let binary = WasmBinary::parse_user(wasm, page_limit, &compile);
        let (bin, stylus_data, footprint) = match binary {
            Ok(data) => data,
            Err(err) => return Err(err.wrap_err("failed to parse program")),
        };
        let info = WasmPricingInfo {
            footprint,
            size: wasm.len() as u32,
        };
        let mach = Self::from_binaries(
            &[forward],
            bin,
            false,
            false,
            false,
            compile.debug.debug_funcs,
            debug,
            GlobalState::default(),
            HashMap::default(),
            Arc::new(|_, _| panic!("user program tried to read preimage")),
            Some(stylus_data),
        )?;
        Ok((mach, info))
    }

    /// Adds a user program to the machine's known set of wasms, compiling it into a link-able module.
    /// Note that the module produced will need to be configured before execution via hostio calls.
    pub fn add_program(
        &mut self,
        wasm: &[u8],
        version: u16,
        debug_funcs: bool,
        hash: Option<Bytes32>,
    ) -> Result<Bytes32> {
        let mut bin = binary::parse(wasm, Path::new("user"))?;
        let config = CompileConfig::version(version, debug_funcs);
        let stylus_data = bin.instrument(&config)?;

        let forward = include_bytes!("../../../target/machines/latest/forward_stub.wasm");
        let forward = binary::parse(forward, Path::new("forward")).unwrap();

        let mut machine = Self::from_binaries(
            &[forward],
            bin,
            false,
            false,
            false,
            debug_funcs,
            self.debug_info,
            GlobalState::default(),
            HashMap::default(),
            Arc::new(|_, _| panic!("tried to read preimage")),
            Some(stylus_data),
        )?;

        let module = machine.modules.pop().unwrap();
        let hash = hash.unwrap_or_else(|| module.hash());
        self.stylus_modules.insert(hash, module);
        Ok(hash)
    }

    pub fn from_binaries(
        libraries: &[WasmBinary<'_>],
        bin: WasmBinary<'_>,
        runtime_support: bool,
        always_merkleize: bool,
        allow_hostapi_from_main: bool,
        debug_funcs: bool,
        debug_info: bool,
        global_state: GlobalState,
        inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
        preimage_resolver: PreimageResolver,
        stylus_data: Option<StylusData>,
    ) -> Result<Machine> {
        use ArbValueType::*;

        // `modules` starts out with the entrypoint module, which will be initialized later
        let mut modules = vec![Module::default()];
        let mut available_imports = HashMap::default();
        let mut floating_point_impls = HashMap::default();
        let main_module_index = u32::try_from(modules.len() + libraries.len())?;

        // make the main module's exports available to libraries
        for (name, &(export, kind)) in &bin.exports {
            if kind == ExportKind::Func {
                let index: usize = export.try_into()?;
                if let Some(index) = index.checked_sub(bin.imports.len()) {
                    let ty: usize = bin.functions[index].try_into()?;
                    let ty = bin.types[ty].clone();
                    available_imports.insert(
                        format!("env__wavm_guest_call__{name}"),
                        AvailableImport::new(ty, main_module_index, export),
                    );
                }
            }
        }

        // collect all the library exports in advance so they can use each other's
        for (index, lib) in libraries.iter().enumerate() {
            let module = 1 + index as u32; // off by one due to the entry point
            for (name, &(export, kind)) in &lib.exports {
                if kind == ExportKind::Func {
                    let ty = match lib.get_function(FunctionIndex::from_u32(export)) {
                        Ok(ty) => ty,
                        Err(error) => bail!("failed to read export {}: {}", name, error),
                    };
                    let import = AvailableImport::new(ty, module, export);
                    available_imports.insert(name.to_owned(), import);
                }
            }
        }

        for lib in libraries {
            let module = Module::from_binary(
                lib,
                &available_imports,
                &floating_point_impls,
                true,
                debug_funcs,
                None,
            )?;
            for (name, &func) in &*module.func_exports {
                let ty = module.func_types[func as usize].clone();
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
                        "Wrong type for floating point impl {} expecting {} but got {}",
                        name.red(),
                        sig.red(),
                        ty.red()
                    );
                    floating_point_impls.insert(op, (modules.len() as u32, func));
                }
            }
            modules.push(module);
        }

        // Shouldn't be necessary, but to be safe, don't allow the main binary to import its own guest calls
        available_imports.retain(|_, i| i.module as usize != modules.len());
        modules.push(Module::from_binary(
            &bin,
            &available_imports,
            &floating_point_impls,
            allow_hostapi_from_main,
            debug_funcs,
            stylus_data,
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
        let main_exports = &main_module.func_exports;

        // Rust support
        let rust_fn = "__main_void";
        if let Some(&f) = main_exports.get(rust_fn).filter(|_| runtime_support) {
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
        if let Some(&f) = main_exports.get("run").filter(|_| runtime_support) {
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
            func_exports: Arc::new(HashMap::default()),
            all_exports: Arc::new(HashMap::default()),
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
            stylus_modules: HashMap::default(),
            guards: vec![],
            initial_hash: Bytes32::default(),
            context: 0,
            debug_info,
        };
        mach.initial_hash = mach.hash();
        Ok(mach)
    }

    #[cfg(feature = "native")]
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
                #[cfg(feature = "rayon")]
                let code_hashes = func.code.par_iter().map(|i| i.hash()).collect();

                #[cfg(not(feature = "rayon"))]
                let code_hashes = func.code.iter().map(|i| i.hash()).collect();

                func.code_merkle = Merkle::new(MerkleType::Instruction, code_hashes);
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
            stylus_modules: HashMap::default(),
            guards: vec![],
            initial_hash: Bytes32::default(),
            context: 0,
            debug_info: false,
        };
        mach.initial_hash = mach.hash();
        Ok(mach)
    }

    #[cfg(feature = "native")]
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

    pub fn program_info(&self) -> (u32, u32) {
        let module = self.modules.last().unwrap();
        let main = module.find_func(STYLUS_ENTRY_POINT).unwrap();
        (main, module.internals_offset)
    }

    pub fn main_module_name(&self) -> String {
        self.modules.last().expect("no module").name().to_owned()
    }

    pub fn main_module_memory(&self) -> &Memory {
        &self.modules.last().expect("no module").memory
    }

    pub fn main_module_hash(&self) -> Bytes32 {
        self.modules.last().expect("no module").hash()
    }

    /// finds the first module with the given name
    pub fn find_module(&self, name: &str) -> Result<u32> {
        let Some(module) = self.modules.iter().position(|m| m.name() == name) else {
            let names: Vec<_> = self.modules.iter().map(|m| m.name()).collect();
            let names = names.join(", ");
            bail!("module {} not found among: {}", name.red(), names)
        };
        Ok(module as u32)
    }

    pub fn find_module_func(&self, module: &str, func: &str) -> Result<(u32, u32)> {
        let qualified = format!("{module}__{func}");
        let offset = self.find_module(module)?;
        let module = &self.modules[offset as usize];
        let func = module
            .find_func(func)
            .or_else(|_| module.find_func(&qualified))?;
        Ok((offset, func))
    }

    pub fn jump_into_func(&mut self, module: u32, func: u32, mut args: Vec<Value>) -> Result<()> {
        let Some(source_module) = self.modules.get(module as usize) else {
            bail!("no module at offest {}", module.red())
        };
        let Some(source_func) = source_module.funcs.get(func as usize) else {
            bail!("no func at offset {} in module {}", func.red(), source_module.name().red())
        };
        let ty = &source_func.ty;
        if ty.inputs.len() != args.len() {
            let name = source_module.names.functions.get(&func).unwrap();
            bail!(
                "func {} has type {} but received args {:?}",
                name.red(),
                ty.red(),
                args
            )
        }

        let frame_args = [Value::RefNull, Value::I32(0), Value::I32(0)];
        args.extend(frame_args);
        self.value_stack = args;

        self.frame_stack.clear();
        self.internal_stack.clear();

        self.pc = ProgramCounter {
            module,
            func,
            inst: 0,
        };
        self.status = MachineStatus::Running;
        self.steps = 0;
        Ok(())
    }

    pub fn get_final_result(&self) -> Result<Vec<Value>> {
        if !self.frame_stack.is_empty() {
            bail!(
                "machine has not successfully computed a final result {}",
                self.status.red()
            )
        }
        Ok(self.value_stack.clone())
    }

    pub fn call_function(
        &mut self,
        module: &str,
        func: &str,
        args: Vec<Value>,
    ) -> Result<Vec<Value>> {
        let (module, func) = self.find_module_func(module, func)?;
        self.jump_into_func(module, func, args)?;
        self.step_n(Machine::MAX_STEPS)?;
        self.get_final_result()
    }

    pub fn call_user_func(&mut self, func: &str, args: Vec<Value>, ink: u64) -> Result<Vec<Value>> {
        self.set_ink(ink);
        self.call_function("user", func, args)
    }

    /// Gets the *last* global with the given name, if one exists
    /// Note: two globals may have the same name, so use carefully!
    pub fn get_global(&self, name: &str) -> Result<Value> {
        for module in self.modules.iter().rev() {
            if let Some((global, ExportKind::Global)) = module.all_exports.get(name) {
                return Ok(module.globals[*global as usize]);
            }
        }
        bail!("global {} not found", name.red())
    }

    /// Sets the *last* global with the given name, if one exists
    /// Note: two globals may have the same name, so use carefully!
    pub fn set_global(&mut self, name: &str, value: Value) -> Result<()> {
        for module in self.modules.iter_mut().rev() {
            if let Some((global, ExportKind::Global)) = module.all_exports.get(name) {
                module.globals[*global as usize] = value;
                return Ok(());
            }
        }
        bail!("global {} not found", name.red())
    }

    pub fn read_memory(&self, module: u32, ptr: u32, len: u32) -> Result<&[u8]> {
        let Some(module) = &self.modules.get(module as usize) else {
            bail!("no module at offset {}", module.red())
        };
        let memory = module.memory.get_range(ptr as usize, len as usize);
        let error = || format!("failed memory read of {} bytes @ {}", len.red(), ptr.red());
        memory.ok_or_else(|| eyre!(error()))
    }

    pub fn write_memory(&mut self, module: u32, ptr: u32, data: &[u8]) -> Result<()> {
        let Some(module) = &mut self.modules.get_mut(module as usize) else {
            bail!("no module at offset {}", module.red())
        };
        if let Err(err) = module.memory.set_range(ptr as usize, data) {
            let msg = eyre!(
                "failed to write {} bytes to memory @ {}",
                data.len().red(),
                ptr.red()
            );
            bail!(err.wrap_err(msg));
        }
        Ok(())
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

    pub fn get_pc(&self) -> Option<ProgramCounter> {
        if self.is_halted() {
            return None;
        }
        Some(self.pc)
    }

    fn test_next_instruction(func: &Function, pc: &ProgramCounter) {
        let inst: usize = pc.inst.try_into().unwrap();
        debug_assert!(func.code.len() > inst);
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

        macro_rules! reset_refs {
            () => {
                module = &mut self.modules[self.pc.module()];
                func = &module.funcs[self.pc.func()];
            };
        }
        macro_rules! flush_module {
            () => {
                if let Some(merkle) = self.modules_merkle.as_mut() {
                    merkle.set(self.pc.module(), module.hash());
                }
            };
        }
        macro_rules! error {
            () => {
                error!("")
            };
            ($format:expr $(, $message:expr)*) => {{
                flush_module!();
                let print_debug_info = |machine: &Self| {
                    println!("\n{} {}", "error on line".grey(), line!().pink());
                    println!($format, $($message.pink()),*);
                    println!("{}", "Backtrace:".grey());
                    machine.print_backtrace(true);
                };

                if let Some(guard) = self.guards.pop() {
                    if self.debug_info {
                        print_debug_info(self);
                    }
                    println!("{}", "Recovering...".pink());

                    // recover at the previous stack heights
                    assert!(self.frame_stack.len() >= guard.frame_stack);
                    assert!(self.value_stack.len() >= guard.value_stack);
                    assert!(self.internal_stack.len() >= guard.inter_stack);
                    self.frame_stack.truncate(guard.frame_stack);
                    self.value_stack.truncate(guard.value_stack);
                    self.internal_stack.truncate(guard.inter_stack);
                    self.pc = guard.on_error;

                    // indicate an error has occured
                    self.value_stack.push(0_u32.into());
                    reset_refs!();
                    continue;
                } else {
                    print_debug_info(self);
                }
                self.status = MachineStatus::Errored;
                module = &mut self.modules[self.pc.module()];
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
                                "Failed to process host call hook for host call {:?} {:?}: {err}",
                                hook.0, hook.1,
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
                    let frame = self.frame_stack.last().unwrap();
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack.push(frame.caller_module.into());
                    self.value_stack.push(frame.caller_module_internals.into());
                    self.pc.func = inst.argument_data as u32;
                    self.pc.inst = 0;
                    func = &module.funcs[self.pc.func()];
                }
                Opcode::CrossModuleCall => {
                    flush_module!();
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack.push(self.pc.module.into());
                    self.value_stack.push(module.internals_offset.into());
                    let (call_module, call_func) = unpack_cross_module_call(inst.argument_data);
                    self.pc.module = call_module;
                    self.pc.func = call_func;
                    self.pc.inst = 0;
                    reset_refs!();
                }
                Opcode::CrossModuleForward => {
                    flush_module!();
                    let frame = self.frame_stack.last().unwrap();
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack.push(frame.caller_module.into());
                    self.value_stack.push(frame.caller_module_internals.into());
                    let (call_module, call_func) = unpack_cross_module_call(inst.argument_data);
                    self.pc.module = call_module;
                    self.pc.func = call_func;
                    self.pc.inst = 0;
                    reset_refs!();
                }
                Opcode::CrossModuleDynamicCall => {
                    flush_module!();
                    let call_func = self.value_stack.pop().unwrap().assume_u32();
                    let call_module = self.value_stack.pop().unwrap().assume_u32();
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack.push(self.pc.module.into());
                    self.value_stack.push(module.internals_offset.into());
                    self.pc.module = call_module;
                    self.pc.func = call_func;
                    self.pc.inst = 0;
                    reset_refs!();
                }
                Opcode::CallerModuleInternalCall => {
                    self.value_stack.push(Value::InternalRef(self.pc));
                    self.value_stack.push(self.pc.module.into());
                    self.value_stack.push(module.internals_offset.into());

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
                        reset_refs!();
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
                    let Some(elem) = elems.get(idx).filter(|e| &e.func_ty == ty) else {
                        error!()
                    };
                    match elem.val {
                        Value::FuncRef(call_func) => {
                            let frame = self.frame_stack.last().unwrap();
                            self.value_stack.push(Value::InternalRef(self.pc));
                            self.value_stack.push(frame.caller_module.into());
                            self.value_stack.push(frame.caller_module_internals.into());
                            self.pc.func = call_func;
                            self.pc.inst = 0;
                            func = &module.funcs[self.pc.func()];
                        }
                        Value::RefNull => error!(),
                        v => bail!("invalid table element value {:?}", v),
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
                    let Some(index) = inst.argument_data.checked_add(base.into()) else {
                        error!()
                    };
                    let Some(value) = module.memory.get_value(index, ty, bytes, signed) else {
                        error!("failed to read offset {}", index)
                    };
                    self.value_stack.push(value);
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
                    let Some(idx) = inst.argument_data.checked_add(base.into()) else {
                        error!()
                    };
                    if !module.memory.store_value(idx, val, bytes) {
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
                        .push(f32::from_bits(inst.argument_data as u32).into());
                }
                Opcode::F64Const => {
                    self.value_stack
                        .push(f64::from_bits(inst.argument_data).into());
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
                    self.value_stack.push(pages.into());
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
                        self.value_stack.push(old_pages.into());
                    } else {
                        // Push -1
                        self.value_stack.push(u32::MAX.into());
                    }
                }
                Opcode::IUnOp(w, op) => {
                    let va = self.value_stack.pop();
                    match w {
                        IntegerValType::I32 => {
                            let Some(Value::I32(value)) = va else {
                                bail!("WASM validation failed: wrong types for i32unop");
                            };
                            self.value_stack.push(exec_iun_op(value, op).into());
                        }
                        IntegerValType::I64 => {
                            let Some(Value::I64(value)) = va else {
                                bail!("WASM validation failed: wrong types for i64unop");
                            };
                            self.value_stack
                                .push(Value::I64(exec_iun_op(value, op) as u64));
                        }
                    }
                }
                Opcode::IBinOp(w, op) => {
                    let vb = self.value_stack.pop();
                    let va = self.value_stack.pop();
                    match w {
                        IntegerValType::I32 => {
                            let (Some(Value::I32(a)), Some(Value::I32(b))) = (va, vb) else {
                                bail!("WASM validation failed: wrong types for i32binop")
                            };
                            if op == IBinOpType::DivS && (a as i32) == i32::MIN && (b as i32) == -1
                            {
                                error!()
                            }
                            let Some(value) = exec_ibin_op(a, b, op) else {
                                error!()
                            };
                            self.value_stack.push(value.into());
                        }
                        IntegerValType::I64 => {
                            let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) else {
                                bail!("WASM validation failed: wrong types for i64binop")
                            };
                            if op == IBinOpType::DivS && (a as i64) == i64::MIN && (b as i64) == -1
                            {
                                error!();
                            }
                            let Some(value) = exec_ibin_op(a, b, op) else {
                                error!()
                            };
                            self.value_stack.push(value.into());
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
                    self.value_stack.push(x64.into());
                }
                Opcode::Reinterpret(dest, source) => {
                    let val = match self.value_stack.pop() {
                        Some(Value::I32(x)) if source == ArbValueType::I32 => {
                            assert_eq!(dest, ArbValueType::F32, "Unsupported reinterpret");
                            f32::from_bits(x).into()
                        }
                        Some(Value::I64(x)) if source == ArbValueType::I64 => {
                            assert_eq!(dest, ArbValueType::F64, "Unsupported reinterpret");
                            f64::from_bits(x).into()
                        }
                        Some(Value::F32(x)) if source == ArbValueType::F32 => {
                            assert_eq!(dest, ArbValueType::I32, "Unsupported reinterpret");
                            x.to_bits().into()
                        }
                        Some(Value::F64(x)) if source == ArbValueType::F64 => {
                            assert_eq!(dest, ArbValueType::I64, "Unsupported reinterpret");
                            x.to_bits().into()
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
                    self.value_stack.push(x.into());
                }
                Opcode::I64ExtendS(b) => {
                    let mut x = self.value_stack.pop().unwrap().assume_u64();
                    let mask = (1u64 << b) - 1;
                    x &= mask;
                    if x & (1 << (b - 1)) != 0 {
                        x |= !mask;
                    }
                    self.value_stack.push(x.into());
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
                            .push(self.global_state.u64_vals[idx].into());
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

                    let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) else {
                        error!()
                    };
                    let Some(preimage) = self.preimage_resolver.get(self.context, hash) else {
                        eprintln!(
                            "{} for hash {}",
                            "Missing requested preimage".red(),
                            hash.red(),
                        );
                        self.print_backtrace(true);
                        bail!("missing requested preimage for hash {}", hash)
                    };

                    let offset = usize::try_from(offset).unwrap();
                    let len = std::cmp::min(32, preimage.len().saturating_sub(offset));
                    let read = preimage.get(offset..(offset + len)).unwrap_or_default();
                    let success = module.memory.store_slice_aligned(ptr.into(), read);
                    assert!(success, "Failed to write to previously read memory");
                    self.value_stack.push(Value::I32(len as u32));
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
                            self.print_backtrace(true);
                            bail!(
                                "missing inbox message {msg_num} of {}",
                                self.first_too_far - 1
                            );
                        }
                        self.status = MachineStatus::TooFar;
                        break;
                    }
                }
                Opcode::LinkModule => {
                    let ptr = self.value_stack.pop().unwrap().assume_u32();
                    let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) else {
                        error!("no hash for {}", ptr)
                    };
                    let Some(module) = self.stylus_modules.get(&hash) else {
                        let keys: Vec<_> = self.stylus_modules.keys().map(hex::encode).collect();
                        bail!("no program for {} in {{{}}}", hex::encode(hash), keys.join(", "))
                    };
                    flush_module!();
                    let index = self.modules.len() as u32;
                    self.value_stack.push(index.into());
                    self.modules.push(module.clone());
                    if let Some(cached) = &mut self.modules_merkle {
                        cached.push_leaf(hash);
                    }
                    reset_refs!();
                }
                Opcode::UnlinkModule => {
                    flush_module!();
                    self.modules.pop();
                    if let Some(cached) = &mut self.modules_merkle {
                        cached.pop_leaf();
                    }
                    reset_refs!();
                }
                Opcode::PushErrorGuard => {
                    self.guards.push(ErrorGuard {
                        frame_stack: self.frame_stack.len(),
                        value_stack: self.value_stack.len(),
                        inter_stack: self.internal_stack.len(),
                        on_error: self.pc,
                    });
                    self.value_stack.push(1_u32.into());
                    reset_refs!();
                }
                Opcode::PopErrorGuard => {
                    self.guards.pop();
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
            Self::say(String::from_utf8_lossy(&self.stdio_output));
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
                    Self::say(String::from_utf8_lossy(&stdio_output[..idx]));
                    if stdio_output.get(idx + 1) == Some(&b'\r') {
                        idx += 1;
                    }
                    *stdio_output = stdio_output.split_off(idx + 1);
                }
                Ok(())
            }
            ("console", "log_i32" | "log_i64" | "log_f32" | "log_f64")
            | ("console", "tee_i32" | "tee_i64" | "tee_f32" | "tee_f64") => {
                let value = value_stack.last().ok_or_else(|| eyre!("missing value"))?;
                Self::say(value);
                Ok(())
            }
            ("console", "log_txt") => {
                let ptr = pull_arg!(1, I32);
                let len = pull_arg!(0, I32);
                let text = read_bytes_segment!(ptr, len);
                match std::str::from_utf8(text) {
                    Ok(text) => Self::say(text),
                    Err(_) => Self::say(hex::encode(text)),
                }
                Ok(())
            }
            _ => Ok(()),
        }
    }

    pub fn say<D: Display>(text: D) {
        let text = format!("{text}");
        let text = match text.len() {
            0..=250 => text,
            _ => format!("{} ...", &text[0..250]),
        };
        println!("{} {text}", "WASM says:".yellow());
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

    fn stack_hashes(
        &self,
    ) -> (
        FrameStackHash,
        ValueStackHash,
        InterStackHash,
        Vec<ErrorGuardProof>,
    ) {
        macro_rules! compute {
            ($field:expr, $stack:expr, $prefix:expr) => {{
                let heights: Vec<_> = self.guards.iter().map($field).collect();
                let frames = $stack.iter().map(|v| v.hash());
                hash_stack_with_heights(frames, &heights, concat!($prefix, " stack:"))
            }};
        }
        let (frame_stack, frames) = compute!(|x| x.frame_stack, self.frame_stack, "Stack frame");
        let (value_stack, values) = compute!(|x| x.value_stack, self.value_stack, "Value");
        let (inter_stack, inters) = compute!(|x| x.inter_stack, self.internal_stack, "Value");

        let pcs = self.guards.iter().map(|x| x.on_error);
        let mut guards: Vec<ErrorGuardProof> = vec![];
        assert_eq!(values.len(), frames.len());
        assert_eq!(values.len(), inters.len());
        assert_eq!(values.len(), pcs.len());

        for (frames, values, inters, pc) in izip!(frames, values, inters, pcs) {
            guards.push(ErrorGuardProof::new(frames, values, inters, pc));
        }
        (frame_stack, value_stack, inter_stack, guards)
    }

    pub fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        match self.status {
            MachineStatus::Running => {
                let (frame_stack, value_stack, inter_stack, guards) = self.stack_hashes();

                h.update(b"Machine running:");
                h.update(value_stack);
                h.update(inter_stack);
                h.update(frame_stack);
                h.update(self.global_state.hash());
                h.update(self.pc.module.to_be_bytes());
                h.update(self.pc.func.to_be_bytes());
                h.update(self.pc.inst.to_be_bytes());
                h.update(self.get_modules_root());

                if !guards.is_empty() {
                    h.update(b"With guards:");
                    h.update(ErrorGuardProof::hash_guards(&guards));
                }
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

        macro_rules! out {
            ($bytes:expr) => {
                data.extend($bytes);
            };
        }
        macro_rules! fail {
            ($format:expr $(,$message:expr)*) => {{
                let text = format!($format, $($message.red()),*);
                panic!("WASM validation failed: {text}");
            }};
        }

        out!(prove_stack(
            &self.value_stack,
            STACK_PROVING_DEPTH,
            hash_value_stack,
            |v| v.serialize_for_proof(),
        ));

        out!(prove_stack(
            &self.internal_stack,
            1,
            hash_value_stack,
            |v| v.serialize_for_proof(),
        ));

        out!(prove_window(
            &self.frame_stack,
            hash_stack_frame_stack,
            StackFrame::serialize_for_proof,
        ));

        out!(self.prove_guards());

        out!(self.global_state.hash());

        out!(self.pc.module.to_be_bytes());
        out!(self.pc.func.to_be_bytes());
        out!(self.pc.inst.to_be_bytes());

        let mod_merkle = self.get_modules_merkle();
        out!(mod_merkle.root());

        // End machine serialization, serialize module

        let module = &self.modules[self.pc.module()];
        let mem_merkle = module.memory.merkelize();
        out!(module.serialize_for_proof(&mem_merkle));

        // Prove module is in modules merkle tree

        out!(mod_merkle
            .prove(self.pc.module())
            .expect("Failed to prove module"));

        if self.is_halted() {
            return data;
        }

        // Begin next instruction proof

        let func = &module.funcs[self.pc.func()];
        out!(func.code[self.pc.inst()].serialize_for_proof());
        out!(func
            .code_merkle
            .prove(self.pc.inst())
            .expect("Failed to prove against code merkle"));
        out!(module
            .funcs_merkle
            .prove(self.pc.func())
            .expect("Failed to prove against function merkle"));

        // End next instruction proof, begin instruction specific serialization

        let Some(next_inst) = func.code.get(self.pc.inst()) else {
            return data;
        };

        let op = next_inst.opcode;
        let arg = next_inst.argument_data;

        use Opcode::*;
        match op {
            GetGlobalStateU64 | SetGlobalStateU64 => {
                out!(self.global_state.serialize());
            }
            LocalGet | LocalSet => {
                let locals = &self.frame_stack.last().unwrap().locals;
                let idx = arg as usize;
                out!(locals[idx].serialize_for_proof());
                let merkle =
                    Merkle::new(MerkleType::Value, locals.iter().map(|v| v.hash()).collect());
                out!(merkle.prove(idx).expect("Out of bounds local access"));
            }
            GlobalGet | GlobalSet => {
                let idx = arg as usize;
                out!(module.globals[idx].serialize_for_proof());
                let globals_merkle = module.globals.iter().map(|v| v.hash()).collect();
                let merkle = Merkle::new(MerkleType::Value, globals_merkle);
                out!(merkle.prove(idx).expect("Out of bounds global access"));
            }
            MemoryLoad { .. } | MemoryStore { .. } => {
                let is_store = matches!(op, MemoryStore { .. });
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
                    x => fail!("memory index type is {x:?}"),
                };
                if let Some(mut idx) = u64::from(base)
                    .checked_add(arg)
                    .and_then(|x| usize::try_from(x).ok())
                {
                    // Prove the leaf this index is in, and the next one, if they are within the memory's size.
                    idx /= Memory::LEAF_SIZE;
                    out!(module.memory.get_leaf_data(idx));
                    out!(mem_merkle.prove(idx).unwrap_or_default());
                    // Now prove the next leaf too, in case it's accessed.
                    let next_leaf_idx = idx.saturating_add(1);
                    out!(module.memory.get_leaf_data(next_leaf_idx));
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
                    out!(second_mem_merkle.prove(next_leaf_idx).unwrap_or_default());
                }
            }
            CallIndirect => {
                let (table, ty) = crate::wavm::unpack_call_indirect(arg);
                let idx = match self.value_stack.last() {
                    Some(Value::I32(i)) => *i,
                    x => fail!("top of stack before call_indirect is {x:?}"),
                };
                let ty = &module.types[usize::try_from(ty).unwrap()];
                out!((table as u64).to_be_bytes());
                out!(ty.hash());
                let table_usize = usize::try_from(table).unwrap();
                let table = &module.tables[table_usize];
                out!(table
                    .serialize_for_proof()
                    .expect("failed to serialize table"));
                out!(module
                    .tables_merkle
                    .prove(table_usize)
                    .expect("Failed to prove tables merkle"));
                let idx_usize = usize::try_from(idx).unwrap();
                if let Some(elem) = table.elems.get(idx_usize) {
                    out!(elem.func_ty.hash());
                    out!(elem.val.serialize_for_proof());
                    out!(table
                        .elems_merkle
                        .prove(idx_usize)
                        .expect("Failed to prove elements merkle"));
                }
            }
            GetGlobalStateBytes32 | SetGlobalStateBytes32 => {
                out!(self.global_state.serialize());
                let ptr = self.value_stack.last().unwrap().assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    // Prove the leaf this index is in
                    idx /= Memory::LEAF_SIZE;
                    out!(module.memory.get_leaf_data(idx));
                    out!(mem_merkle.prove(idx).unwrap_or_default());
                }
            }
            ReadPreImage | ReadInboxMessage => {
                let ptr = self
                    .value_stack
                    .get(self.value_stack.len() - 2)
                    .unwrap()
                    .assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    // Prove the leaf this index is in
                    idx /= Memory::LEAF_SIZE;
                    let prev_data = module.memory.get_leaf_data(idx);
                    out!(prev_data);
                    out!(mem_merkle.prove(idx).unwrap_or_default());
                    if op == Opcode::ReadPreImage {
                        let hash = Bytes32(prev_data);
                        let Some(preimage) = self.preimage_resolver.get_const(self.context, hash) else {
                            fail!("Missing requested preimage for hash {}", hash)
                        };
                        data.push(0); // preimage proof type
                        out!(preimage);
                    } else if op == Opcode::ReadInboxMessage {
                        let msg_idx = self
                            .value_stack
                            .get(self.value_stack.len() - 3)
                            .unwrap()
                            .assume_u64();
                        let inbox_id = argument_data_to_inbox(arg).expect("Bad inbox indentifier");
                        if let Some(msg_data) = self.inbox_contents.get(&(inbox_id, msg_idx)) {
                            data.push(0); // inbox proof type
                            out!(msg_data);
                        }
                    } else {
                        unreachable!()
                    }
                }
            }
            LinkModule | UnlinkModule => {
                if op == LinkModule {
                    let leaf_index = match self.value_stack.last() {
                        Some(Value::I32(x)) => *x as usize / Memory::LEAF_SIZE,
                        x => fail!("module pointer has invalid type {x:?}"),
                    };
                    out!(module.memory.get_leaf_data(leaf_index));
                    out!(mem_merkle.prove(leaf_index).unwrap_or_default());
                }

                // prove that our proposed leaf x has a leaf-like hash
                let module = self.modules.last().unwrap();
                out!(module.serialize_for_proof(&module.memory.merkelize()));

                // prove that leaf x is under the root at position p
                let leaf = self.modules.len() - 1;
                out!((leaf as u32).to_be_bytes());
                out!(mod_merkle.prove(leaf).unwrap());

                // if needed, prove that x is the last module by proving that leaf p + 1 is 0
                let balanced = math::is_power_of_2(leaf + 1);
                if !balanced {
                    out!(mod_merkle.prove_any(leaf + 1));
                }
            }
            _ => {}
        }
        data
    }

    fn prove_guards(&self) -> Vec<u8> {
        let mut data = Vec::with_capacity(33); // size in the empty case
        let guards = self.stack_hashes().3;
        if guards.is_empty() {
            data.extend(Bytes32::default());
            data.push(0);
        } else {
            let last_idx = guards.len() - 1;
            data.extend(ErrorGuardProof::hash_guards(&guards[..last_idx]));
            data.push(1);
            data.extend(guards[last_idx].serialize_for_proof());
        }
        data
    }

    pub fn get_data_stack(&self) -> &[Value] {
        &self.value_stack
    }

    pub fn get_internals_stack(&self) -> &[Value] {
        &self.internal_stack
    }

    pub fn get_guards(&self) -> &[ErrorGuard] {
        &self.guards
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

    pub fn get_module_names(&self, module: usize) -> Option<&NameCustomSection> {
        self.modules.get(module).map(|m| &*m.names)
    }

    pub fn print_backtrace(&self, stderr: bool) {
        let print = |line: String| match stderr {
            true => println!("{}", line),
            false => eprintln!("{}", line),
        };

        let print_pc = |pc: ProgramCounter| {
            let names = &self.modules[pc.module()].names;
            let func = names
                .functions
                .get(&pc.func)
                .cloned()
                .unwrap_or_else(|| pc.func.to_string());
            let func = rustc_demangle::demangle(&func);
            let module = match names.module.is_empty() {
                true => pc.module.to_string(),
                false => names.module.clone(),
            };
            let inst = format!("#{}", pc.inst);
            print(format!(
                "  {} {} {} {}",
                module.grey(),
                func.mint(),
                "inst".grey(),
                inst.blue(),
            ));
        };

        print_pc(self.pc);
        for frame in self.frame_stack.iter().rev().take(25) {
            if let Value::InternalRef(pc) = frame.return_ref {
                print_pc(pc);
            }
        }
        if self.frame_stack.len() > 25 {
            print(format!("  ... and {} more", self.frame_stack.len() - 25).grey());
        }
    }
}
