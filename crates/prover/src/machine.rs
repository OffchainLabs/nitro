// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#[cfg(feature = "counters")]
use std::sync::atomic::{AtomicUsize, Ordering};
use std::{
    borrow::Cow,
    convert::{TryFrom, TryInto},
    fmt::{self, Display},
    fs::File,
    hash::Hash,
    io::{BufReader, BufWriter, Write},
    num::Wrapping,
    ops::Add,
    path::{Path, PathBuf},
    sync::Arc,
};

use arbutil::{Bytes32, Color, DebugColor, PreimageType, crypto, math};
use brotli::Dictionary;
#[cfg(feature = "native")]
use c_kzg::BYTES_PER_BLOB;
use digest::Digest;
use eyre::{Result, WrapErr, bail, ensure, eyre};
use fnv::FnvHashMap as HashMap;
use lazy_static::lazy_static;
use num::{Zero, traits::PrimInt};
#[cfg(feature = "rayon")]
use rayon::prelude::*;
use serde::{Deserialize, Serialize};
use serde_with::serde_as;
use sha3::Keccak256;
use smallvec::SmallVec;
use wasmer_types::FunctionIndex;
use wasmparser::{DataKind, ElementItems, ElementKind, Operator, RefType, TableType};

#[cfg(feature = "native")]
use crate::kzg::prove_kzg_preimage;
use crate::{
    binary::{
        self, ExportKind, ExportMap, FloatInstruction, Local, NameCustomSection, WasmBinary, parse,
    },
    host,
    memory::Memory,
    merkle::{Merkle, MerkleType},
    programs::{ModuleMod, StylusData, config::CompileConfig, meter::MeteredMachine},
    reinterpret::{ReinterpretAsSigned, ReinterpretAsUnsigned},
    utils::{CBytes, RemoteTableType, file_bytes},
    value::{ArbValueType, FunctionType, IntegerValType, ProgramCounter, Value},
    wavm::{
        self, FloatingPointImpls, IBinOpType, IRelOpType, IUnOpType, Instruction, Opcode,
        pack_cross_module_call, unpack_cross_module_call, wasm_to_wavm,
    },
    wavm_serialize::{
        Cursor, FunctionParts, TableElementParts, TableParts, WAVM_COMPRESSION_BROTLI,
        WAVM_COMPRESSION_NONE, WAVM_MAGIC, WAVM_SERIALIZE_VERSION, read_count, read_export_map,
        read_func_exports, read_function_parts, read_function_type, read_host_call_hooks,
        read_names, read_table_parts, read_value, write_bytes, write_bytes32, write_count,
        write_export_map, write_func_exports, write_function_parts, write_function_type,
        write_host_call_hooks, write_names, write_optional_u32, write_table_parts, write_u32,
        write_u64, write_value,
    },
};

#[cfg(feature = "counters")]
static GET_MODULES_MERKLE_COUNTER: AtomicUsize = AtomicUsize::new(0);

#[cfg(feature = "counters")]
pub fn print_counters() {
    println!(
        "GET_MODULES_MERKLE_COUNTER: {}",
        GET_MODULES_MERKLE_COUNTER.load(Ordering::Relaxed)
    );
}

#[cfg(feature = "counters")]
pub fn reset_counters() {
    GET_MODULES_MERKLE_COUNTER.store(0, Ordering::Relaxed);
}

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
    pub code: Vec<Instruction>,
    pub ty: FunctionType,
    #[serde(skip)]
    code_merkle: Merkle,
    pub local_types: Vec<ArbValueType>,
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
                let (table, ty) = wavm::unpack_call_indirect(inst.argument_data);
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
        let mut func = Function {
            code,
            ty,
            code_merkle: Merkle::default(), // TODO: make an option
            local_types,
        };
        func.set_code_merkle();
        func
    }

    const CHUNK_SIZE: usize = 64;

    fn set_code_merkle(&mut self) {
        let code = &self.code;
        let chunks = math::div_ceil::<64>(code.len());
        let crunch = |x: usize| Instruction::hash(&code[64 * x..(64 * (x + 1)).min(code.len())]);

        #[cfg(feature = "rayon")]
        let code_hashes = (0..chunks).into_par_iter().map(crunch).collect();

        #[cfg(not(feature = "rayon"))]
        let code_hashes = (0..chunks).map(crunch).collect();

        self.code_merkle = Merkle::new(MerkleType::Instruction, code_hashes);
    }

    fn serialize_body_for_proof(&self, pc: ProgramCounter) -> Vec<u8> {
        let start = pc.inst() / 64 * 64;
        let end = (start + 64).min(self.code.len());
        Instruction::serialize_for_proof(&self.code[start..end])
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
pub(crate) struct TableElement {
    func_ty: FunctionType,
    pub val: Value,
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
pub(crate) struct Table {
    #[serde(with = "RemoteTableType")]
    pub ty: TableType,
    pub elems: Vec<TableElement>,
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
pub struct Module {
    pub(crate) globals: Vec<Value>,
    pub(crate) memory: Memory,
    pub(crate) tables: Vec<Table>,
    #[serde(skip)]
    pub(crate) tables_merkle: Merkle,
    pub(crate) funcs: Arc<Vec<Function>>,
    #[serde(skip)]
    pub(crate) funcs_merkle: Arc<Merkle>,
    pub(crate) types: Arc<Vec<FunctionType>>,
    pub(crate) internals_offset: u32,
    pub(crate) names: Arc<NameCustomSection>,
    pub(crate) host_call_hooks: Arc<Vec<Option<(String, String)>>>,
    pub(crate) start_function: Option<u32>,
    pub(crate) func_types: Arc<Vec<FunctionType>>,
    /// Old modules use this format.
    /// TODO: remove this after the jump to stylus.
    #[serde(alias = "exports")]
    pub(crate) func_exports: Arc<HashMap<String, u32>>,
    #[serde(default)]
    pub(crate) all_exports: Arc<ExportMap>,
    /// Used to make modules unique.
    pub(crate) extra_hash: Arc<Bytes32>,
}

lazy_static! {
    static ref USER_IMPORTS: HashMap<String, AvailableImport> = {
        let mut imports = HashMap::default();

        let forward = include_bytes!("forward_stub.wat");
        let forward = wat::parse_bytes(forward).unwrap();
        let forward = binary::parse(&forward, Path::new("forward")).unwrap();

        for (name, &(export, kind)) in &forward.exports {
            if kind == ExportKind::Func {
                let ty = match forward.get_function(FunctionIndex::from_u32(export)) {
                    Ok(ty) => ty,
                    Err(error) => panic!("failed to read export {name}: {error:?}"),
                };
                let import = AvailableImport::new(ty, 1, export);
                imports.insert(name.to_owned(), import);
            }
        }
        imports
    };
}

impl Module {
    const FORWARDING_PREFIX: &'static str = "arbitrator_forward__";

    fn from_binary(
        bin: &WasmBinary,
        available_imports: &HashMap<String, AvailableImport>,
        floating_point_impls: &FloatingPointImpls,
        allow_hostapi: bool,
        debug_funcs: bool,
        stylus_data: Option<StylusData>,
        version: u16,
    ) -> Result<Module> {
        let mut code = Vec::new();
        let mut func_type_idxs: Vec<u32> = Vec::new();
        let mut memory = Memory::default();
        let mut tables = Vec::new();
        let mut host_call_hooks = Vec::new();
        let bin_name = &bin.names.module;
        for import in &bin.imports {
            let module = import.module;
            let have_ty = &bin.types[import.offset as usize];
            // allow_hostapi is only set for system modules like the
            // forwarder. We restrict stripping the prefix for user modules.
            let (forward, import_name) =
                if allow_hostapi && import.name.starts_with(Self::FORWARDING_PREFIX) {
                    (true, &import.name[Self::FORWARDING_PREFIX.len()..])
                } else {
                    (false, import.name)
                };

            let qualified_name = format!("{module}__{import_name}");

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
            } else {
                match host::get_impl(import.module, import_name) {
                    Ok((hostio, debug)) => {
                        ensure!(
                            (debug && debug_funcs) || (!debug && allow_hostapi),
                            "Host func {} in {} not enabled debug_funcs={debug_funcs} hostapi={allow_hostapi} debug={debug}",
                            import_name.red(),
                            import.module.red(),
                        );
                        hostio
                    }
                    _ => {
                        bail!(
                            "No such import {} in {} for {}",
                            import_name.red(),
                            import.module.red(),
                            bin_name.red()
                        )
                    }
                }
            };
            ensure!(
                &func.ty == have_ty,
                "Import {} for {} has different function signature than export.\nexpected {} in {}\nbut have {}",
                import_name.red(),
                bin_name.red(),
                func.ty.red(),
                module.red(),
                have_ty.red(),
            );

            func_type_idxs.push(import.offset);
            code.push(func);
            host_call_hooks.push(Some((import.module.into(), import_name.into())));
        }
        func_type_idxs.extend(bin.functions.iter());

        let func_exports: HashMap<String, u32> = bin
            .exports
            .iter()
            .filter(|(_, (_, kind))| kind == &ExportKind::Func)
            .map(|(name, (offset, _))| (name.to_owned(), *offset))
            .collect();

        let internals = host::new_internal_funcs(stylus_data, version);
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
                        bin_name,
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
            let allowed = Memory::MAX_WASM_PAGES;

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
            let (memory_index, mut init) = match &data.kind {
                DataKind::Active {
                    memory_index,
                    offset_expr,
                } => (memory_index, offset_expr.get_operators_reader()),
                _ => continue,
            };
            ensure!(
                *memory_index == 0,
                "Attempted to write to nonexistant memory"
            );

            let offset = match (init.read()?, init.read()?, init.eof()) {
                (Operator::I32Const { value }, Operator::End, true) => value as usize,
                x => bail!("Non-constant element segment offset expression {x:?}"),
            };
            if !matches!(
                offset.checked_add(data.data.len()),
                Some(x) if (x as u64) <= memory.size(),
            ) {
                bail!(
                    "Out-of-bounds data memory init with offset {} and size {}",
                    offset.red(),
                    data.data.len().red(),
                );
            }
            memory.set_range(offset, data.data)?;
        }

        for table in &bin.tables {
            let element_type = table.element_type;
            ensure!(
                element_type == RefType::FUNCREF,
                "unsupported table type {element_type}"
            );
            tables.push(Table {
                elems: vec![TableElement::default(); usize::try_from(table.initial).unwrap()],
                ty: *table,
                elems_merkle: Merkle::default(),
            });
        }

        for elem in &bin.elements {
            let (t, mut init) = match &elem.kind {
                ElementKind::Active {
                    table_index,
                    offset_expr,
                } => (
                    table_index.unwrap_or_default() as usize,
                    offset_expr.get_operators_reader(),
                ),
                _ => continue, // we don't support the ops that use these
            };
            let offset = match (init.read()?, init.read()?, init.eof()) {
                (Operator::I32Const { value }, Operator::End, true) => value as usize,
                x => bail!("Non-constant element segment offset expression {x:?}"),
            };
            let Some(table) = tables.get_mut(t) else {
                bail!("Element segment for non-exsistent table {}", t.red())
            };

            let mut contents = vec![];
            let ElementItems::Functions(item_reader) = elem.items.clone() else {
                bail!("Non-constant element initializers are not supported");
            };
            for func in item_reader.into_iter() {
                let index = func?;
                let func_ty = func_types[index as usize].clone();
                contents.push(TableElement {
                    val: Value::FuncRef(index),
                    func_ty,
                })
            }

            let len = contents.len();
            ensure!(
                offset.saturating_add(len) <= table.elems.len(),
                "Out of bounds element segment at offset {offset} and length {len} for table of length {}",
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
            extra_hash: Arc::new(crypto::keccak(&bin.extra_data).into()),
        })
    }

    pub fn from_user_binary(
        bin: &WasmBinary,
        debug_funcs: bool,
        stylus_data: Option<StylusData>,
        version: u16,
    ) -> Result<Module> {
        Self::from_binary(
            bin,
            &USER_IMPORTS,
            &HashMap::default(),
            false,
            debug_funcs,
            stylus_data,
            version,
        )
    }

    pub fn name(&self) -> &str {
        &self.names.module
    }

    fn find_func(&self, name: &str) -> Result<u32> {
        let Some(func) = self.func_exports.iter().find(|x| x.0 == name) else {
            bail!("func {} not found in {}", name.red(), self.name().red())
        };
        Ok(*func.1)
    }

    pub fn hash(&self) -> Bytes32 {
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
        h.update(*self.extra_hash);
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
        data.extend(*self.extra_hash);
        data.extend(self.internals_offset.to_be_bytes());
        data
    }

    /// Serializes the `Module` into the WAVM wire format
    /// (`MAGIC | VERSION | COMPRESSION_TAG | u32 LEN | brotli(body)`). Fields
    /// are emitted in `Module` declaration order; schema changes require bumping
    /// `WAVM_SERIALIZE_VERSION`, which `validateOrUpgradeWavmSerializeVersion`
    /// translates into an on-disk purge. Returns `Err` instead of panicking so
    /// failures cross the FFI boundary as status codes.
    pub fn to_wavm_bytes(&self) -> Result<Vec<u8>> {
        // Body follows `Module`'s declaration order; `tables_merkle` and
        // `funcs_merkle` are re-derived on decode.
        //
        // Pre-size to skip the ~10 doubling reallocations a multi-MB body would
        // otherwise cost. Instruction stream + memory buffer dominate.
        let body_capacity_hint = self.memory.size() as usize
            + self
                .funcs
                .iter()
                .map(|f| 64 + f.code.len() * 16)
                .sum::<usize>()
            + 4096;
        let mut body = Vec::with_capacity(body_capacity_hint);

        write_count(&mut body, self.globals.len())?;
        for v in &self.globals {
            write_value(&mut body, *v);
        }

        // `get_range(0, size)` is total by construction (`size == buffer.len()`);
        // the `bail!` makes a future Memory-invariant break loud instead of
        // silently diverging the hash from the activator's.
        let size = self.memory.size() as usize;
        let buffer = self.memory.get_range(0, size).ok_or_else(|| {
            eyre!(
                "wavm encode: memory.get_range(0, {size}) returned None — Memory invariant broken"
            )
        })?;
        write_bytes(&mut body, buffer)?;
        write_u64(&mut body, self.memory.max_size);

        write_count(&mut body, self.tables.len())?;
        for t in &self.tables {
            let parts = TableParts {
                ty: t.ty,
                elems: t
                    .elems
                    .iter()
                    .map(|e| TableElementParts {
                        func_ty: e.func_ty.clone(),
                        val: e.val,
                    })
                    .collect(),
            };
            write_table_parts(&mut body, &parts)?;
        }

        write_count(&mut body, self.funcs.len())?;
        for f in self.funcs.iter() {
            let parts = FunctionParts {
                local_types: f.local_types.clone(),
                ty: f.ty.clone(),
                code: f.code.clone(),
            };
            write_function_parts(&mut body, &parts)?;
        }

        write_count(&mut body, self.types.len())?;
        for ty in self.types.iter() {
            write_function_type(&mut body, ty)?;
        }

        write_u32(&mut body, self.internals_offset);
        write_names(&mut body, &self.names)?;
        write_host_call_hooks(&mut body, &self.host_call_hooks)?;
        write_optional_u32(&mut body, self.start_function);

        write_count(&mut body, self.func_types.len())?;
        for ty in self.func_types.iter() {
            write_function_type(&mut body, ty)?;
        }

        write_func_exports(&mut body, &self.func_exports)?;
        write_export_map(&mut body, &self.all_exports)?;
        write_bytes32(&mut body, &self.extra_hash);

        // q=0 shrinks realistic modules ~20–80x at sub-ms cost; higher q saves
        // little on an already-tiny payload.
        let compressed = brotli::compress(&body, 0, 22, Dictionary::Empty)
            .map_err(|s| eyre!("wavm encode: brotli compression failed: {s:?}"))?;

        // Length-prefix the body so the decoder catches trailing-bytes
        // corruption regardless of brotli's behavior on extra input.
        let mut out = Vec::with_capacity(WAVM_MAGIC.len() + 1 + 1 + 4 + compressed.len());
        out.extend_from_slice(WAVM_MAGIC);
        out.push(WAVM_SERIALIZE_VERSION);
        out.push(WAVM_COMPRESSION_BROTLI);
        write_bytes(&mut out, &compressed)?;

        Ok(out)
    }

    /// Deserializes a `Module` from the stable WAVM wire format. Returns `Err` on
    /// header mismatch, malformed payload, or trailing bytes; never panics. The
    /// rebuilt module's `Module::hash()` matches the activator's by construction:
    /// `Table::elems_merkle` is left as `Merkle::default()` to mirror
    /// `Module::from_binary`.
    pub fn from_wavm_bytes(data: &[u8]) -> Result<Module> {
        ensure!(
            data.len() > WAVM_MAGIC.len(),
            "wavm decode: data too short for header"
        );
        ensure!(
            &data[..WAVM_MAGIC.len()] == WAVM_MAGIC,
            "wavm decode: magic mismatch"
        );
        let version = data[WAVM_MAGIC.len()];
        ensure!(
            version == WAVM_SERIALIZE_VERSION,
            "wavm decode: unsupported WavmSerializeVersion {version}, expected {WAVM_SERIALIZE_VERSION}",
        );

        // Read the envelope: COMPRESSION_TAG | u32 BODY_LEN | BODY. Length
        // prefix is verified against `env.is_empty()` below so trailing
        // bytes after the body are rejected regardless of brotli's
        // trailing-byte behavior.
        let mut env = Cursor::new(&data[WAVM_MAGIC.len() + 1..]);
        let compression_tag = env.read_u8()?;
        let raw_body = env.read_bytes()?;
        ensure!(
            env.is_empty(),
            "wavm decode: {} trailing byte(s) after envelope",
            env.remaining(),
        );

        let body: Vec<u8> = match compression_tag {
            WAVM_COMPRESSION_NONE => raw_body,
            WAVM_COMPRESSION_BROTLI => brotli::decompress(&raw_body, Dictionary::Empty)
                .map_err(|s| eyre!("wavm decode: brotli decompression failed: {s:?}"))?,
            other => bail!("wavm decode: unknown compression tag {other}"),
        };

        let mut c = Cursor::new(&body);

        // globals (each `Value` is at least 1 tag byte)
        let n_globals = read_count(&mut c, 1)?;
        let mut globals = Vec::with_capacity(n_globals);
        for _ in 0..n_globals {
            globals.push(read_value(&mut c)?);
        }

        // memory: buffer + max_size
        let memory_buffer = c.read_bytes()?;
        let memory_max_size = c.read_u64()?;

        // tables (each table is at least kind(1) + initial(8) + maximum-flag(1)
        // + table64(1) + shared(1) + elem-count(4) = 16 bytes)
        let n_tables = read_count(&mut c, 16)?;
        let mut tables = Vec::with_capacity(n_tables);
        for _ in 0..n_tables {
            let parts = read_table_parts(&mut c)?;
            let elems = parts
                .elems
                .into_iter()
                .map(|e| TableElement {
                    func_ty: e.func_ty,
                    val: e.val,
                })
                .collect();
            tables.push(Table {
                ty: parts.ty,
                elems,
                elems_merkle: Merkle::default(),
            });
        }

        // funcs (each function is at least local-count(4) + input-count(4)
        // + output-count(4) + inst-count(4) = 16 bytes)
        let n_funcs = read_count(&mut c, 16)?;
        let mut funcs: Vec<Function> = Vec::with_capacity(n_funcs);
        for _ in 0..n_funcs {
            let parts = read_function_parts(&mut c)?;
            funcs.push(Function::new_from_wavm(
                parts.code,
                parts.ty,
                parts.local_types,
            ));
        }

        // types (each FunctionType has two u32 counts = 8 bytes minimum)
        let n_types = read_count(&mut c, 8)?;
        let mut types = Vec::with_capacity(n_types);
        for _ in 0..n_types {
            types.push(read_function_type(&mut c)?);
        }

        // internals_offset
        let internals_offset = c.read_u32()?;

        // names (module + functions map)
        let names = read_names(&mut c)?;

        // host_call_hooks
        let host_call_hooks = read_host_call_hooks(&mut c)?;

        // start_function
        let start_function = c.read_optional_u32()?;

        // func_types
        let n_func_types = read_count(&mut c, 8)?;
        let mut func_types = Vec::with_capacity(n_func_types);
        for _ in 0..n_func_types {
            func_types.push(read_function_type(&mut c)?);
        }

        // func_exports
        let func_exports = read_func_exports(&mut c)?;

        // all_exports
        let all_exports = read_export_map(&mut c)?;

        // extra_hash
        let extra_hash = c.read_bytes32()?;

        ensure!(
            c.is_empty(),
            "wavm decode: {} trailing byte(s) after module",
            c.remaining(),
        );

        // Reconstruct memory.
        let mut memory = Memory::new(memory_buffer.len(), memory_max_size);
        if !memory_buffer.is_empty() {
            memory.set_range(0, &memory_buffer)?;
        }
        memory.cache_merkle_tree();

        // Leave each `table.elems_merkle` as `Merkle::default()` to mirror the activator
        // path (`Module::from_binary`): `Table::hash` reads `elems_merkle.root()`, so
        // hashing here with an empty merkle keeps `Module::hash()` identical to the
        // hash reported by `stylus_activate`. Repopulating would diverge from the
        // activator's commitment and from the on-chain `module_hash`.
        let tables_hashes: Result<_> = tables.iter().map(Table::hash).collect();
        let tables_merkle = Merkle::new(MerkleType::Table, tables_hashes?);

        // funcs_merkle is over Function::hash() values (which themselves merkleize the
        // instruction stream). new_from_wavm above already populated each Function's
        // code_merkle, so this is straightforward.
        let funcs_merkle = Arc::new(Merkle::new(
            MerkleType::Function,
            funcs.iter().map(Function::hash).collect(),
        ));

        Ok(Module {
            globals,
            memory,
            tables,
            tables_merkle,
            funcs: Arc::new(funcs),
            funcs_merkle,
            types: Arc::new(types),
            internals_offset,
            names: Arc::new(names),
            host_call_hooks: Arc::new(host_call_hooks),
            start_function,
            func_types: Arc::new(func_types),
            func_exports: Arc::new(func_exports),
            all_exports: Arc::new(all_exports),
            extra_hash: Arc::new(extra_hash),
        })
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

impl From<GlobalState> for validation::GoGlobalState {
    fn from(gs: GlobalState) -> Self {
        Self {
            block_hash: gs.bytes32_vals[0],
            send_root: gs.bytes32_vals[1],
            batch: gs.u64_vals[0],
            pos_in_batch: gs.u64_vals[1],
        }
    }
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
    globals: Cow<'a, [Value]>,
    memory: Cow<'a, Memory>,
}

/// Represents if the machine can recover and where to jump back if so.
#[derive(Clone, Copy, Debug, Serialize, Deserialize)]
pub enum ThreadState {
    /// Execution is in the main thread. Errors are fatal.
    Main,
    /// Execution is in a cothread. Errors recover to the associated pc with the main thread.
    CoThread(ProgramCounter),
}

impl ThreadState {
    fn is_cothread(&self) -> bool {
        match self {
            ThreadState::Main => false,
            ThreadState::CoThread(_) => true,
        }
    }

    fn serialize(&self) -> Bytes32 {
        match self {
            ThreadState::Main => Bytes32([0xff; 32]),
            ThreadState::CoThread(pc) => (*pc).serialize(),
        }
    }
}

#[derive(Serialize, Deserialize)]
pub struct MachineState<'a> {
    steps: u64, // Not part of machine hash
    thread_state: ThreadState,
    status: MachineStatus,
    value_stacks: Cow<'a, [Vec<Value>]>,
    internal_stack: Cow<'a, [Value]>,
    frame_stacks: Cow<'a, [Vec<StackFrame>]>,
    modules: Vec<ModuleState<'a>>,
    global_state: GlobalState,
    pc: ProgramCounter,
    stdio_output: Cow<'a, [u8]>,
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

    #[cfg(feature = "native")]
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

    #[cfg(feature = "native")]
    pub fn get_const(&self, context: u64, ty: PreimageType, hash: Bytes32) -> Option<CBytes> {
        if let Some(resolved) = &self.last_resolved
            && resolved.0 == hash
        {
            return Some(resolved.1.clone());
        }
        (self.resolver)(context, ty, hash)
    }
}

#[derive(Clone, Debug)]
pub struct Machine {
    steps: u64, // Not part of machine hash
    thread_state: ThreadState,
    status: MachineStatus,
    value_stacks: Vec<Vec<Value>>,
    internal_stack: Vec<Value>,
    frame_stacks: Vec<Vec<StackFrame>>,
    modules: Vec<Module>,
    modules_merkle: Option<Merkle>,
    global_state: GlobalState,
    pc: ProgramCounter,
    stdio_output: Vec<u8>,
    inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
    first_too_far: u64, // Not part of machine hash
    preimage_resolver: PreimageResolverWrapper,
    /// Linkable Stylus modules in compressed form. Not part of the machine hash.
    stylus_modules: HashMap<Bytes32, Vec<u8>>,
    initial_hash: Bytes32,
    context: u64,
    debug_info: bool, // Not part of machine hash
}

type FrameStackHash = Bytes32;
type ValueStackHash = Bytes32;
type MultiStackHash = Bytes32;
type InterStackHash = Bytes32;

pub(crate) fn hash_stack<I, D>(stack: I, prefix: &str) -> Bytes32
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

        use digest::Update;

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

fn hash_multistack<T, F>(multistack: &[&[T]], stack_hasher: F) -> MultiStackHash
where
    F: Fn(&[T]) -> Bytes32,
{
    hash_stack(multistack.iter().map(|v| stack_hasher(v)), "cothread:")
}

#[must_use]
#[cfg(feature = "native")]
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
#[cfg(feature = "native")]
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

// prove_multistacks encodes proof for multistacks:
// - Proof of first(main) if not cothread otherwise last
// - Hash of first if cothread, otherwise last
// - Recursive hash of the rest
// If length is < 1, hash of last element is assumed 0xff..f, same for hash
// of in-between stacks ([2nd..last)).
// Accepts prover function so that it can work both for proving stack and window.
#[must_use]
#[cfg(feature = "native")]
fn prove_multistack<T, F, MF>(
    cothread: bool,
    items: Vec<&[T]>,
    stack_hasher: F,
    multistack_hasher: MF,
    prover: fn(&[T]) -> Vec<u8>,
) -> Vec<u8>
where
    F: Fn(&[T]) -> Bytes32,
    MF: Fn(&[&[T]], F) -> Bytes32,
{
    let mut data = Vec::with_capacity(33);

    if cothread {
        data.extend(prover(items.last().unwrap()));
        data.extend(stack_hasher(items.first().unwrap()))
    } else {
        data.extend(prover(items.first().unwrap()));

        let last_hash = if items.len() > 1 {
            stack_hasher(items.last().unwrap())
        } else {
            Machine::NO_STACK_HASH
        };
        data.extend(last_hash);
    }
    let hash: Bytes32 = if items.len() > 2 {
        multistack_hasher(&items[1..items.len() - 1], stack_hasher)
    } else {
        Bytes32::default()
    };
    data.extend(hash);
    data
}

#[must_use]
#[cfg(feature = "native")]
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
#[cfg(feature = "native")]
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

#[cfg(feature = "native")]
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
    pub const NO_STACK_HASH: Bytes32 = Bytes32([255_u8; 32]);

    pub fn from_paths(
        library_paths: &[PathBuf],
        binary_path: &Path,
        language_support: bool,
        allow_hostapi_from_main: bool,
        debug_funcs: bool,
        debug_info: bool,
        global_state: GlobalState,
        inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
        preimage_resolver: PreimageResolver,
    ) -> Result<Machine> {
        let bin_source = file_bytes(binary_path)?;
        let bin = parse(&bin_source, binary_path)
            .wrap_err_with(|| format!("failed to validate WASM binary at {binary_path:?}"))?;
        let mut libraries = vec![];
        let mut lib_sources = vec![];
        for path in library_paths {
            let error_message = format!("failed to validate WASM binary at {path:?}");
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
            allow_hostapi_from_main,
            debug_funcs,
            debug_info,
            global_state,
            inbox_contents,
            preimage_resolver,
            None,
            0, // version only applies to user (Stylus) modules, not system libraries
        )
    }

    /// Creates an instrumented user Machine from the wasm or wat at the given `path`.
    #[cfg(feature = "native")]
    pub fn from_user_path(path: &Path, compile: &CompileConfig) -> Result<Self> {
        let data = std::fs::read(path)?;
        let wasm = wasmer::wat2wasm(&data)?;
        let mut bin = binary::parse(&wasm, Path::new("user"))?;
        let stylus_data = bin.instrument(compile, &Bytes32::default())?;

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
            true,
            compile.debug.debug_funcs,
            true,
            GlobalState::default(),
            HashMap::default(),
            Arc::new(|_, _, _| panic!("tried to read preimage")),
            Some(stylus_data),
            compile.version,
        )?;

        let footprint: u32 = stylus_data.footprint.into();
        machine.call_function("user_test", "set_pages", vec![footprint.into()])?;
        Ok(machine)
    }

    /// Adds a user program to the machine's known set of wasms, compiling it into a link-able
    /// module. Note that the module produced will need to be configured before execution via
    /// hostio calls.
    pub fn add_program(
        &mut self,
        wasm: &[u8],
        codehash: &Bytes32,
        version: u16,
        debug_funcs: bool,
    ) -> Result<Bytes32> {
        let mut bin = binary::parse(wasm, Path::new("user"))?;
        let config = CompileConfig::version(version, debug_funcs);
        let stylus_data = bin.instrument(&config, codehash)?;

        // enable debug mode if debug funcs are available
        if debug_funcs {
            self.debug_info = true;
        }

        let module = Module::from_user_binary(&bin, debug_funcs, Some(stylus_data), version)?;
        let hash = module.hash();
        self.add_stylus_module(hash, module.to_wavm_bytes()?);
        Ok(hash)
    }

    /// Adds a pre-built program to the machine's known set of wasms.
    pub fn add_stylus_module(&mut self, hash: Bytes32, module: Vec<u8>) {
        self.stylus_modules.insert(hash, module);
    }

    pub fn from_binaries(
        libraries: &[WasmBinary<'_>],
        bin: WasmBinary<'_>,
        runtime_support: bool,
        allow_hostapi_from_main: bool,
        debug_funcs: bool,
        debug_info: bool,
        global_state: GlobalState,
        inbox_contents: HashMap<(InboxIdentifier, u64), Vec<u8>>,
        preimage_resolver: PreimageResolver,
        stylus_data: Option<StylusData>,
        version: u16,
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
                        Err(error) => bail!("failed to read export {name}: {error}"),
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
                0, // version only applies to user (Stylus) modules, not system libraries
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

        // Shouldn't be necessary, but to be safe, don't allow the main binary to import its own
        // guest calls
        available_imports.retain(|_, i| i.module as usize != modules.len());
        modules.push(Module::from_binary(
            &bin,
            &available_imports,
            &floating_point_impls,
            allow_hostapi_from_main,
            debug_funcs,
            stylus_data,
            version,
        )?);

        // Build the entrypoint module
        let mut entrypoint = Vec::new();
        macro_rules! entry {
            ($opcode:ident) => {
                entrypoint.push(Instruction::simple(Opcode::$opcode));
            };
            ($opcode:ident, $value:expr_2021) => {
                entrypoint.push(Instruction::with_data(Opcode::$opcode, $value));
            };
            ($opcode:ident ($inside:expr_2021)) => {
                entrypoint.push(Instruction::simple(Opcode::$opcode($inside)));
            };
            (@cross, $module:expr_2021, $func:expr_2021) => {
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
            let expected_type = FunctionType::new([], [I32]);
            ensure!(
                main_module.func_types[f as usize] == expected_type,
                "Main function doesn't match expected signature of [] -> [ret]",
            );
            entry!(@cross, u32::try_from(main_module_idx).unwrap(), f);
            entry!(Drop);
            entry!(HaltAndSetFinished);
        }

        // Go/wasi support
        if let Some(&f) = main_exports.get("_start").filter(|_| runtime_support) {
            let expected_type = FunctionType::new([], []);
            ensure!(
                main_module.func_types[f as usize] == expected_type,
                "Main function doesn't match expected signature of [] -> []",
            );
            entry!(@cross, u32::try_from(main_module_idx).unwrap(), f);
            entry!(HaltAndSetFinished);
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
            host_call_hooks: Default::default(),
            start_function: None,
            func_types: Arc::new(vec![FunctionType::default()]),
            func_exports: Default::default(),
            all_exports: Default::default(),
            extra_hash: Default::default(),
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
            module.memory.cache_merkle_tree();
        }
        let modules_merkle = Some(Merkle::new(
            MerkleType::Module,
            modules.iter().map(Module::hash).collect(),
        ));

        // find the first inbox index that's out of bounds
        let first_too_far = inbox_contents
            .iter()
            .filter(|((kind, _), _)| kind == &InboxIdentifier::Sequencer)
            .map(|((_, index), _)| *index + 1)
            .max()
            .unwrap_or(0);

        let mut mach = Machine {
            status: MachineStatus::Running,
            thread_state: ThreadState::Main,
            steps: 0,
            value_stacks: vec![vec![Value::RefNull, Value::I32(0), Value::I32(0)]],
            internal_stack: Vec::new(),
            frame_stacks: vec![Vec::new()],
            modules,
            modules_merkle,
            global_state,
            pc: ProgramCounter::default(),
            stdio_output: Vec::new(),
            inbox_contents,
            first_too_far,
            preimage_resolver: PreimageResolverWrapper::new(preimage_resolver),
            stylus_modules: HashMap::default(),
            initial_hash: Bytes32::default(),
            context: 0,
            debug_info,
        };
        mach.initial_hash = mach.hash();
        Ok(mach)
    }

    // new_finished returns a Machine in the Finished state at step 0.
    //
    // This allows the Mahine to be set up to model the final state of the
    // machine at the end of the execution of a block.
    pub fn new_finished(gs: GlobalState) -> Machine {
        Machine {
            steps: 0,
            status: MachineStatus::Finished,
            global_state: gs,
            // The machine is in the Finished state, so nothing else really matters.
            // values_stacks and frame_stacks cannot be empty for proof serialization,
            // but everything else can just be entirely blank.
            thread_state: ThreadState::Main,
            value_stacks: vec![Vec::new()],
            frame_stacks: vec![Vec::new()],
            internal_stack: Default::default(),
            modules: Default::default(),
            modules_merkle: Default::default(),
            pc: Default::default(),
            stdio_output: Default::default(),
            inbox_contents: Default::default(),
            first_too_far: Default::default(),
            preimage_resolver: PreimageResolverWrapper::new(Arc::new(|_, _, _| None)),
            stylus_modules: Default::default(),
            initial_hash: Default::default(),
            context: Default::default(),
            debug_info: Default::default(),
        }
    }

    pub fn new_from_wavm(wavm_binary: &Path) -> Result<Machine> {
        let mut modules: Vec<Module> = {
            let compressed = std::fs::read(wavm_binary)?;
            let Ok(modules) = brotli::decompress(&compressed, Dictionary::Empty) else {
                bail!("failed to decompress wavm binary");
            };
            bincode::deserialize(&modules)?
        };

        for module in modules.iter_mut() {
            for table in module.tables.iter_mut() {
                table.elems_merkle = Merkle::new(
                    MerkleType::TableElement,
                    table.elems.iter().map(TableElement::hash).collect(),
                );
            }
            let tables: Result<_> = module.tables.iter().map(Table::hash).collect();
            module.tables_merkle = Merkle::new(MerkleType::Table, tables?);

            let funcs = Arc::get_mut(&mut module.funcs).expect("Multiple copies of module funcs");
            funcs.iter_mut().for_each(Function::set_code_merkle);

            module.funcs_merkle = Arc::new(Merkle::new(
                MerkleType::Function,
                module.funcs.iter().map(Function::hash).collect(),
            ));
            module.memory.cache_merkle_tree();
        }
        let modules_merkle = Some(Merkle::new(
            MerkleType::Module,
            modules.iter().map(Module::hash).collect(),
        ));
        let mut mach = Machine {
            status: MachineStatus::Running,
            thread_state: ThreadState::Main,
            steps: 0,
            value_stacks: vec![vec![Value::RefNull, Value::I32(0), Value::I32(0)]],
            internal_stack: Vec::new(),
            frame_stacks: vec![Vec::new()],
            modules,
            modules_merkle,
            global_state: Default::default(),
            pc: ProgramCounter::default(),
            stdio_output: Vec::new(),
            inbox_contents: Default::default(),
            first_too_far: 0,
            preimage_resolver: PreimageResolverWrapper::new(get_empty_preimage_resolver()),
            stylus_modules: HashMap::default(),
            initial_hash: Bytes32::default(),
            context: 0,
            debug_info: false,
        };
        mach.initial_hash = mach.hash();
        Ok(mach)
    }

    pub fn serialize_binary<P: AsRef<Path>>(&self, path: P) -> Result<()> {
        ensure!(
            self.hash() == self.initial_hash,
            "serialize_binary can only be called on initial machine",
        );
        let modules = bincode::serialize(&self.modules)?;
        let window = brotli::DEFAULT_WINDOW_SIZE;
        let Ok(output) = brotli::compress(&modules, 9, window, Dictionary::Empty) else {
            bail!("failed to compress binary");
        };

        let mut file = File::create(path)?;
        file.write_all(&output)?;
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
            thread_state: self.thread_state,
            status: self.status,
            value_stacks: Cow::Borrowed(&self.value_stacks),
            internal_stack: Cow::Borrowed(&self.internal_stack),
            frame_stacks: Cow::Borrowed(&self.frame_stacks),
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

    // Requires that this is the same base machine. If this returns an error, it has not mutated
    // `self`.
    pub fn deserialize_and_replace_state<P: AsRef<Path>>(&mut self, path: P) -> Result<()> {
        let reader = BufReader::new(File::open(path)?);
        let new_state: MachineState = bincode::deserialize_from(reader)?;
        if self.initial_hash != new_state.initial_hash {
            bail!(
                "attempted to load deserialize machine with initial hash {} into machine with initial hash {}",
                new_state.initial_hash,
                self.initial_hash,
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
        self.value_stacks = new_state.value_stacks.into_owned();
        self.internal_stack = new_state.internal_stack.into_owned();
        self.frame_stacks = new_state.frame_stacks.into_owned();
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
        let Some(module) = self
            .modules
            .iter()
            .position(|m| m.name().trim_end_matches(".wasm") == name)
        else {
            let names: Vec<_> = self.modules.iter().map(|m| m.name()).collect();
            let names = names.join(", ");
            bail!("module {} not found among: {names}", name.red())
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
            bail!("no module at offset {}", module.red())
        };
        let Some(source_func) = source_module.funcs.get(func as usize) else {
            bail!(
                "no func at offset {} in module {}",
                func.red(),
                source_module.name().red()
            )
        };
        let ty = &source_func.ty;
        if ty.inputs.len() != args.len() {
            // `names.functions` is sparse (Wasm name section is optional); fall
            // back to the numeric index for functions that have no symbol.
            let name = source_module
                .names
                .functions
                .get(&func)
                .cloned()
                .unwrap_or_else(|| format!("#{func}"));
            bail!(
                "func {} has type {} but received args {:?}",
                name.red(),
                ty.red(),
                args.debug_red(),
            )
        }

        let frame_args = [Value::RefNull, Value::I32(0), Value::I32(0)];
        args.extend(frame_args);
        self.value_stacks[0] = args;

        self.frame_stacks[0].clear();
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
        if self.thread_state.is_cothread() {
            bail!("machine in cothread when expecting final result")
        }
        if !self.frame_stacks[0].is_empty() {
            bail!(
                "machine has not successfully computed a final result {}",
                self.status.red()
            )
        }
        Ok(self.value_stacks[0].clone())
    }

    #[cfg(feature = "native")]
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

    #[cfg(feature = "native")]
    pub fn call_user_func(
        &mut self,
        func: &str,
        args: Vec<Value>,
        ink: arbutil::evm::api::Ink,
    ) -> Result<Vec<Value>> {
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

    #[cfg(feature = "native")]
    fn test_next_instruction(func: &Function, pc: &ProgramCounter) {
        let inst: usize = pc.inst.try_into().unwrap();
        debug_assert!(func.code.len() > inst);
    }

    pub fn get_steps(&self) -> u64 {
        self.steps
    }

    #[cfg(feature = "native")]
    pub fn step_n(&mut self, n: u64) -> Result<()> {
        if self.is_halted() {
            return Ok(());
        }
        let (mut value_stack, mut frame_stack) = match self.thread_state {
            ThreadState::Main => (&mut self.value_stacks[0], &mut self.frame_stacks[0]),
            ThreadState::CoThread(_) => (
                self.value_stacks.last_mut().unwrap(),
                self.frame_stacks.last_mut().unwrap(),
            ),
        };
        let mut module = &mut self.modules[self.pc.module()];
        let mut func = &module.funcs[self.pc.func()];

        macro_rules! reset_refs {
            () => {
                (value_stack, frame_stack) = match self.thread_state {
                    ThreadState::Main => (&mut self.value_stacks[0], &mut self.frame_stacks[0]),
                    ThreadState::CoThread(_) => (
                        self.value_stacks.last_mut().unwrap(),
                        self.frame_stacks.last_mut().unwrap(),
                    ),
                };
                module = &mut self.modules[self.pc.module()];
                func = &module.funcs[self.pc.func()];
            };
        }
        macro_rules! error {
            () => {
                error!("")
            };
            ($format:expr_2021 $(, $message:expr_2021)*) => {{
                if self.debug_info {
                    println!("\n{} {}", "error on line".grey(), line!().pink());
                    println!($format, $($message.pink()),*);
                    println!("{}", "backtrace:".grey());
                    self.print_backtrace(true);
                }

                if let ThreadState::CoThread(recovery_pc) = self.thread_state {
                    self.thread_state = ThreadState::Main;
                    self.pc = recovery_pc;
                    reset_refs!();
                    if self.debug_info {
                        println!("\n{}", "switching to main thread".grey());
                        println!("\n{} {:?}", "next opcode: ".grey(), func.code[self.pc.inst()]);
                    }
                    continue;
                }
                self.status = MachineStatus::Errored;
                break;
            }};
        }

        for _ in 0..n {
            self.steps += 1;
            if self.steps == Self::MAX_STEPS {
                println!("\n{}", "Machine out of steps".red());
                self.status = MachineStatus::Errored;
                self.print_backtrace(true);
                break;
            }

            let inst = func.code[self.pc.inst()];
            self.pc.inst += 1;
            match inst.opcode {
                Opcode::Unreachable => error!("unreachable"),
                Opcode::Nop => {}
                Opcode::InitFrame => {
                    let caller_module_internals = value_stack.pop().unwrap().assume_u32();
                    let caller_module = value_stack.pop().unwrap().assume_u32();
                    let return_ref = value_stack.pop().unwrap();
                    frame_stack.push(StackFrame {
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
                        && let Err(err) = Self::host_call_hook(
                            value_stack,
                            module,
                            &mut self.stdio_output,
                            &hook.0,
                            &hook.1,
                        )
                    {
                        eprintln!(
                            "Failed to process host call hook for host call {:?} {:?}: {err}",
                            hook.0, hook.1,
                        );
                    }
                }
                Opcode::ArbitraryJump => {
                    self.pc.inst = inst.argument_data as u32;
                    Machine::test_next_instruction(func, &self.pc);
                }
                Opcode::ArbitraryJumpIf => {
                    let x = value_stack.pop().unwrap();
                    if !x.is_i32_zero() {
                        self.pc.inst = inst.argument_data as u32;
                        Machine::test_next_instruction(func, &self.pc);
                    }
                }
                Opcode::Return => {
                    let frame = frame_stack.pop().unwrap();
                    match frame.return_ref {
                        Value::RefNull => error!(),
                        Value::InternalRef(pc) => {
                            let changing_module = pc.module != self.pc.module;
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
                    let frame = frame_stack.last().unwrap();
                    value_stack.push(Value::InternalRef(self.pc));
                    value_stack.push(frame.caller_module.into());
                    value_stack.push(frame.caller_module_internals.into());
                    self.pc.func = inst.argument_data as u32;
                    self.pc.inst = 0;
                    func = &module.funcs[self.pc.func()];
                }
                Opcode::CrossModuleCall => {
                    value_stack.push(Value::InternalRef(self.pc));
                    value_stack.push(self.pc.module.into());
                    value_stack.push(module.internals_offset.into());
                    let (call_module, call_func) = unpack_cross_module_call(inst.argument_data);
                    self.pc.module = call_module;
                    self.pc.func = call_func;
                    self.pc.inst = 0;
                    reset_refs!();
                }
                Opcode::CrossModuleForward => {
                    let frame = frame_stack.last().unwrap();
                    value_stack.push(Value::InternalRef(self.pc));
                    value_stack.push(frame.caller_module.into());
                    value_stack.push(frame.caller_module_internals.into());
                    let (call_module, call_func) = unpack_cross_module_call(inst.argument_data);
                    self.pc.module = call_module;
                    self.pc.func = call_func;
                    self.pc.inst = 0;
                    reset_refs!();
                }
                Opcode::CrossModuleInternalCall => {
                    let call_internal = inst.argument_data as u32;
                    let call_module = value_stack.pop().unwrap().assume_u32();
                    value_stack.push(Value::InternalRef(self.pc));
                    value_stack.push(self.pc.module.into());
                    value_stack.push(module.internals_offset.into());
                    module = &mut self.modules[call_module as usize];
                    self.pc.module = call_module;
                    self.pc.func = module.internals_offset + call_internal;
                    self.pc.inst = 0;
                    reset_refs!();
                }
                Opcode::CallerModuleInternalCall => {
                    value_stack.push(Value::InternalRef(self.pc));
                    value_stack.push(self.pc.module.into());
                    value_stack.push(module.internals_offset.into());

                    let current_frame = frame_stack.last().unwrap();
                    if current_frame.caller_module_internals > 0 {
                        let func_idx = u32::try_from(inst.argument_data)
                            .ok()
                            .and_then(|o| current_frame.caller_module_internals.checked_add(o))
                            .expect("Internal call function index overflow");
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
                    let idx = match value_stack.pop() {
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
                            let frame = frame_stack.last().unwrap();
                            value_stack.push(Value::InternalRef(self.pc));
                            value_stack.push(frame.caller_module.into());
                            value_stack.push(frame.caller_module_internals.into());
                            self.pc.func = call_func;
                            self.pc.inst = 0;
                            func = &module.funcs[self.pc.func()];
                        }
                        Value::RefNull => error!(),
                        v => bail!("invalid table element value {:?}", v),
                    }
                }
                Opcode::LocalGet => {
                    let val = frame_stack.last().unwrap().locals[inst.argument_data as usize];
                    value_stack.push(val);
                }
                Opcode::LocalSet => {
                    let val = value_stack.pop().unwrap();
                    let locals = &mut frame_stack.last_mut().unwrap().locals;
                    if locals.len() <= inst.argument_data as usize {
                        error!("not enough locals")
                    }
                    locals[inst.argument_data as usize] = val;
                }
                Opcode::GlobalGet => {
                    value_stack.push(module.globals[inst.argument_data as usize]);
                }
                Opcode::GlobalSet => {
                    let val = value_stack.pop().unwrap();
                    module.globals[inst.argument_data as usize] = val;
                }
                Opcode::MemoryLoad { ty, bytes, signed } => {
                    let base = match value_stack.pop() {
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
                    value_stack.push(value);
                }
                Opcode::MemoryStore { ty: _, bytes } => {
                    let val = match value_stack.pop() {
                        Some(Value::I32(x)) => x.into(),
                        Some(Value::I64(x)) => x,
                        Some(Value::F32(x)) => x.to_bits().into(),
                        Some(Value::F64(x)) => x.to_bits(),
                        x => bail!(
                            "WASM validation failed: attempted to memory store type {:?}",
                            x,
                        ),
                    };
                    let base = match value_stack.pop() {
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
                    value_stack.push(Value::I32(inst.argument_data as u32));
                }
                Opcode::I64Const => {
                    value_stack.push(Value::I64(inst.argument_data));
                }
                Opcode::F32Const => {
                    value_stack.push(f32::from_bits(inst.argument_data as u32).into());
                }
                Opcode::F64Const => {
                    value_stack.push(f64::from_bits(inst.argument_data).into());
                }
                Opcode::I32Eqz => {
                    let val = value_stack.pop().unwrap();
                    value_stack.push(Value::I32(val.is_i32_zero() as u32));
                }
                Opcode::I64Eqz => {
                    let val = value_stack.pop().unwrap();
                    value_stack.push(Value::I32(val.is_i64_zero() as u32));
                }
                Opcode::IRelOp(t, op, signed) => {
                    let vb = value_stack.pop();
                    let va = value_stack.pop();
                    match t {
                        IntegerValType::I32 => {
                            if let (Some(Value::I32(a)), Some(Value::I32(b))) = (va, vb) {
                                if signed {
                                    value_stack.push(exec_irel_op(a as i32, b as i32, op));
                                } else {
                                    value_stack.push(exec_irel_op(a, b, op));
                                }
                            } else {
                                bail!("WASM validation failed: wrong types for i32relop");
                            }
                        }
                        IntegerValType::I64 => {
                            if let (Some(Value::I64(a)), Some(Value::I64(b))) = (va, vb) {
                                if signed {
                                    value_stack.push(exec_irel_op(a as i64, b as i64, op));
                                } else {
                                    value_stack.push(exec_irel_op(a, b, op));
                                }
                            } else {
                                bail!("WASM validation failed: wrong types for i64relop");
                            }
                        }
                    }
                }
                Opcode::Drop => {
                    value_stack.pop().unwrap();
                }
                Opcode::Select => {
                    let selector_zero = value_stack.pop().unwrap().is_i32_zero();
                    let val2 = value_stack.pop().unwrap();
                    let val1 = value_stack.pop().unwrap();
                    if selector_zero {
                        value_stack.push(val2);
                    } else {
                        value_stack.push(val1);
                    }
                }
                Opcode::MemorySize => {
                    let pages = u32::try_from(module.memory.size() / Memory::PAGE_SIZE)
                        .expect("Memory pages grew past a u32");
                    value_stack.push(pages.into());
                }
                Opcode::MemoryGrow => {
                    let old_size = module.memory.size();
                    let adding_pages = match value_stack.pop() {
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
                        value_stack.push(old_pages.into());
                    } else {
                        // Push -1
                        value_stack.push(u32::MAX.into());
                    }
                }
                Opcode::IUnOp(w, op) => {
                    let va = value_stack.pop();
                    match w {
                        IntegerValType::I32 => {
                            let Some(Value::I32(value)) = va else {
                                bail!("WASM validation failed: wrong types for i32unop");
                            };
                            value_stack.push(exec_iun_op(value, op).into());
                        }
                        IntegerValType::I64 => {
                            let Some(Value::I64(value)) = va else {
                                bail!("WASM validation failed: wrong types for i64unop");
                            };
                            value_stack.push(Value::I64(exec_iun_op(value, op) as u64));
                        }
                    }
                }
                Opcode::IBinOp(w, op) => {
                    let vb = value_stack.pop();
                    let va = value_stack.pop();
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
                            value_stack.push(value.into());
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
                            value_stack.push(value.into());
                        }
                    }
                }
                Opcode::I32WrapI64 => {
                    let x = match value_stack.pop() {
                        Some(Value::I64(x)) => x,
                        v => bail!(
                            "WASM validation failed: wrong type for i32.wrapi64: {:?}",
                            v,
                        ),
                    };
                    value_stack.push(Value::I32(x as u32));
                }
                Opcode::I64ExtendI32(signed) => {
                    let x: u32 = value_stack.pop().unwrap().assume_u32();
                    let x64 = match signed {
                        true => x as i32 as i64 as u64,
                        false => x as u64,
                    };
                    value_stack.push(x64.into());
                }
                Opcode::Reinterpret(dest, source) => {
                    let val = match value_stack.pop() {
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
                    value_stack.push(val);
                }
                Opcode::I32ExtendS(b) => {
                    let mut x = value_stack.pop().unwrap().assume_u32();
                    let mask = (1u32 << b) - 1;
                    x &= mask;
                    if x & (1 << (b - 1)) != 0 {
                        x |= !mask;
                    }
                    value_stack.push(x.into());
                }
                Opcode::I64ExtendS(b) => {
                    let mut x = value_stack.pop().unwrap().assume_u64();
                    let mask = (1u64 << b) - 1;
                    x &= mask;
                    if x & (1 << (b - 1)) != 0 {
                        x |= !mask;
                    }
                    value_stack.push(x.into());
                }
                Opcode::MoveFromStackToInternal => {
                    self.internal_stack.push(value_stack.pop().unwrap());
                }
                Opcode::MoveFromInternalToStack => {
                    value_stack.push(self.internal_stack.pop().unwrap());
                }
                Opcode::Dup => {
                    let val = value_stack.last().cloned().unwrap();
                    value_stack.push(val);
                }
                Opcode::GetGlobalStateBytes32 => {
                    let ptr = value_stack.pop().unwrap().assume_u32();
                    let idx = value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.bytes32_vals.len()
                        || !module
                            .memory
                            .store_slice_aligned(ptr.into(), &*self.global_state.bytes32_vals[idx])
                    {
                        error!();
                    }
                }
                Opcode::SetGlobalStateBytes32 => {
                    let ptr = value_stack.pop().unwrap().assume_u32();
                    let idx = value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.bytes32_vals.len() {
                        error!();
                    } else if let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) {
                        self.global_state.bytes32_vals[idx] = hash;
                    } else {
                        error!();
                    }
                }
                Opcode::GetGlobalStateU64 => {
                    let idx = value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.u64_vals.len() {
                        error!();
                    } else {
                        value_stack.push(self.global_state.u64_vals[idx].into());
                    }
                }
                Opcode::SetGlobalStateU64 => {
                    let val = value_stack.pop().unwrap().assume_u64();
                    let idx = value_stack.pop().unwrap().assume_u32() as usize;
                    if idx >= self.global_state.u64_vals.len() {
                        error!();
                    } else {
                        self.global_state.u64_vals[idx] = val
                    }
                }
                Opcode::ValidateCertificate => {
                    let preimage_type = value_stack.pop().unwrap().assume_u32();
                    let hash_ptr = value_stack.pop().unwrap().assume_u32();

                    // Try to convert preimage_type to PreimageType
                    let Ok(preimage_ty) = PreimageType::try_from(u8::try_from(preimage_type)?)
                    else {
                        // For invalid preimage types, return 0 (invalid)
                        value_stack.push(Value::from(0u32));
                        continue;
                    };

                    // Load the hash from memory
                    let Some(hash) = module.memory.load_32_byte_aligned(hash_ptr.into()) else {
                        error!();
                    };

                    // For types other than DACertificate, always return valid (1)
                    if preimage_ty != PreimageType::DACertificate {
                        value_stack.push(Value::from(1u32));
                        continue;
                    }

                    // For DACertificate, check if the preimage exists in the resolver
                    // (which means it was pre-validated during batch processing)
                    let is_valid = self
                        .preimage_resolver
                        .get(self.context, preimage_ty, hash)
                        .is_some();

                    value_stack.push(Value::from(if is_valid { 1u32 } else { 0u32 }));
                }
                Opcode::ReadPreImage => {
                    let offset = value_stack.pop().unwrap().assume_u32();
                    let ptr = value_stack.pop().unwrap().assume_u32();
                    let preimage_ty = PreimageType::try_from(u8::try_from(inst.argument_data)?)?;
                    // Preimage reads must be word aligned
                    if offset % 32 != 0 {
                        error!();
                    }

                    let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) else {
                        error!();
                    };

                    // For DACertificate type, ValidateCertificate should have been called first
                    // ReadPreImage assumes certificates are valid and preimages are available.

                    let Some(preimage) =
                        self.preimage_resolver.get(self.context, preimage_ty, hash)
                    else {
                        eprintln!(
                            "{} for hash {}",
                            "Missing requested preimage".red(),
                            hash.red(),
                        );
                        self.print_backtrace(true);
                        bail!("missing requested preimage for hash {}", hash);
                    };
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
                    value_stack.push(Value::I32(len as u32));
                }
                Opcode::ReadInboxMessage => {
                    let offset = value_stack.pop().unwrap().assume_u32();
                    let ptr = value_stack.pop().unwrap().assume_u32();
                    let msg_num = value_stack.pop().unwrap().assume_u64();
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
                                value_stack.push(Value::I32(len as u32));
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
                    let ptr = value_stack.pop().unwrap().assume_u32();
                    let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) else {
                        error!("no hash for {}", ptr)
                    };
                    let Some(bytes) = self.stylus_modules.get(&hash) else {
                        let modules = &self.stylus_modules;
                        let keys: Vec<_> = modules.keys().take(16).map(hex::encode).collect();
                        let dots = if modules.len() > 16 {
                            "..."
                        } else {
                            Default::default()
                        };
                        bail!("no program for {hash} in {{{}{dots}}}", keys.join(", "))
                    };

                    // `Module::from_wavm_bytes` mirrors the activator's empty
                    // `Table::elems_merkle` semantics, so `new_module.hash() == hash`
                    // is guaranteed by construction. Enforce that invariant in
                    // release as well: a silent mismatch would install a module
                    // whose `hash()` no longer equals the WAVM-level lookup key,
                    // breaking the BOLD fraud-proof commitment.
                    //
                    // Decode and hash-check BEFORE mutating `value_stack` /
                    // `self.modules` so the machine's pre-step state is
                    // preserved on the bail path; the previous order pushed
                    // the module index before validating and left
                    // `value_stack` with a phantom index when bailing.
                    let new_module = match Module::from_wavm_bytes(bytes) {
                        Ok(m) => m,
                        Err(e) => bail!("failed to decode stylus module {hash}: {e}"),
                    };
                    let new_hash = new_module.hash();
                    if new_hash != hash {
                        bail!(
                            "decoded stylus module hash {new_hash} diverged from lookup key {hash} — wavm round-trip invariant broken",
                        );
                    }
                    // Now commit: push the offset on the stack and install
                    // the module.
                    let index = self.modules.len() as u32;
                    value_stack.push(index.into());
                    self.modules.push(new_module);
                    if let Some(cached) = &mut self.modules_merkle {
                        cached.push_leaf(hash);
                    }
                    reset_refs!();
                }
                Opcode::UnlinkModule => {
                    self.modules.pop();
                    if let Some(cached) = &mut self.modules_merkle {
                        cached.pop_leaf();
                    }
                    reset_refs!();
                }
                Opcode::HaltAndSetFinished => {
                    self.status = MachineStatus::Finished;
                    break;
                }
                Opcode::NewCoThread => {
                    if self.thread_state.is_cothread() {
                        error!("called NewCoThread from cothread")
                    }
                    self.value_stacks.push(Vec::new());
                    self.frame_stacks.push(Vec::new());
                    reset_refs!();
                }
                Opcode::PopCoThread => {
                    if self.thread_state.is_cothread() {
                        error!("called PopCoThread from cothread")
                    }
                    self.value_stacks.pop();
                    self.frame_stacks.pop();
                    reset_refs!();
                }
                Opcode::SwitchThread => {
                    let next_recovery = match inst.argument_data {
                        0 => ThreadState::Main,
                        x => ThreadState::CoThread(self.pc.add((x - 1).try_into().unwrap())),
                    };
                    if next_recovery.is_cothread() == self.thread_state.is_cothread() {
                        error!("SwitchThread doesn't switch")
                    }
                    self.thread_state = next_recovery;
                    reset_refs!();
                }
            }
        }
        if self.is_halted() && !self.stdio_output.is_empty() {
            // If we halted, print out any trailing output that didn't have a newline.
            Self::say(String::from_utf8_lossy(&self.stdio_output));
            self.stdio_output.clear();
        }
        Ok(())
    }

    #[cfg(feature = "native")]
    fn host_call_hook(
        value_stack: &[Value],
        module: &Module,
        stdio_output: &mut Vec<u8>,
        module_name: &str,
        name: &str,
    ) -> Result<()> {
        macro_rules! pull_arg {
            ($offset:expr_2021, $t:ident) => {
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
            ($ptr:expr_2021) => {
                module
                    .memory
                    .get_u32($ptr.into())
                    .ok_or_else(|| eyre!("pointer out of bounds"))?
            };
        }
        macro_rules! read_bytes_segment {
            ($ptr:expr_2021, $size:expr_2021) => {
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
                        "\x1b[31mWASM exiting\x1b[0m with exit code \x1b[31m{exit_code}\x1b[0m",
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
        println!("{} {text}", "WASM says:".yellow());
    }

    pub fn print_modules(&self) {
        for module in &self.modules {
            println!("{module}\n");
        }
        for module in self.stylus_modules.values() {
            match Module::from_wavm_bytes(module) {
                Ok(m) => println!("{m}\n"),
                Err(e) => println!("<failed to decode stylus module: {e}>\n"),
            }
        }
    }

    pub fn is_halted(&self) -> bool {
        self.status != MachineStatus::Running
    }

    pub fn get_status(&self) -> MachineStatus {
        self.status
    }

    fn get_modules_merkle(&self) -> Cow<'_, Merkle> {
        #[cfg(feature = "counters")]
        GET_MODULES_MERKLE_COUNTER.fetch_add(1, Ordering::Relaxed);

        if let Some(merkle) = &self.modules_merkle {
            for (i, module) in self.modules.iter().enumerate() {
                merkle.set(i, module.hash());
            }
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

    fn stack_hashes(&self) -> (FrameStackHash, ValueStackHash, InterStackHash) {
        macro_rules! compute {
            ($stack:expr_2021, $prefix:expr_2021) => {{
                let frames = $stack.iter().map(|v| v.hash());
                hash_stack(frames, concat!($prefix, " stack:"))
            }};
        }
        // compute_multistack returns the hash of multistacks as follows:
        // Keccak(
        //      "multistack:"
        //      + hash_stack(first_stack)
        //      + hash_stack(last_stack)
        //      + Keccak("cothread:" + 2nd_stack+Keccak("cothread:" + 3drd_stack + ...)
        // )
        macro_rules! compute_multistack {
            ($field:expr_2021, $stacks:expr_2021, $prefix:expr_2021, $hasher: expr_2021) => {{
                let first_elem = *$stacks.first().unwrap();
                let first_hash = hash_stack(
                    first_elem.iter().map(|v| v.hash()),
                    concat!($prefix, " stack:"),
                );

                let last_hash = if $stacks.len() <= 1 {
                    Machine::NO_STACK_HASH
                } else {
                    let last_elem = *$stacks.last().unwrap();
                    hash_stack(
                        last_elem.iter().map(|v| v.hash()),
                        concat!($prefix, " stack:"),
                    )
                };

                // Hash of stacks [2nd..last) or 0xfff...f if len <= 2.
                let mut hash = if $stacks.len() <= 2 {
                    Bytes32::default()
                } else {
                    hash_multistack(&$stacks[1..$stacks.len() - 1], $hasher)
                };

                hash = Keccak256::new()
                    .chain("multistack:")
                    .chain(first_hash)
                    .chain(last_hash)
                    .chain(hash)
                    .finalize()
                    .into();
                hash
            }};
        }

        use digest::Update;
        let frame_stacks = compute_multistack!(
            |x| x.frame_stack,
            self.get_frame_stacks(),
            "Stack frame",
            hash_stack_frame_stack
        );
        let value_stacks = compute_multistack!(
            |x| x.value_stack,
            self.get_data_stacks(),
            "Value",
            hash_value_stack
        );
        let inter_stack = compute!(self.internal_stack, "Value");

        (frame_stacks, value_stacks, inter_stack)
    }

    pub fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        match self.status {
            MachineStatus::Running => {
                let (frame_stacks, value_stacks, inter_stack) = self.stack_hashes();

                h.update(b"Machine running:");
                h.update(value_stacks);
                h.update(inter_stack);
                h.update(frame_stacks);
                h.update(self.global_state.hash());
                h.update(self.pc.module.to_be_bytes());
                h.update(self.pc.func.to_be_bytes());
                h.update(self.pc.inst.to_be_bytes());
                h.update(self.thread_state.serialize());
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

    #[cfg(feature = "native")]
    pub fn serialize_proof(&self) -> Vec<u8> {
        // Could be variable, but not worth it yet
        const STACK_PROVING_DEPTH: usize = 3;

        let mut data = vec![self.status as u8];

        macro_rules! out {
            ($bytes:expr_2021) => {
                data.extend($bytes);
            };
        }
        macro_rules! fail {
            ($format:expr_2021 $(,$message:expr_2021)*) => {{
                let text = format!($format, $($message.red()),*);
                panic!("WASM validation failed: {text}");
            }};
        }
        out!(prove_multistack(
            self.thread_state.is_cothread(),
            self.get_data_stacks(),
            hash_value_stack,
            hash_multistack,
            |stack| prove_stack(stack, STACK_PROVING_DEPTH, hash_value_stack, |v| v
                .serialize_for_proof()),
        ));

        out!(prove_stack(
            &self.internal_stack,
            1,
            hash_value_stack,
            |v| v.serialize_for_proof(),
        ));

        out!(prove_multistack(
            self.thread_state.is_cothread(),
            self.get_frame_stacks(),
            hash_stack_frame_stack,
            hash_multistack,
            |stack| prove_window(
                stack,
                hash_stack_frame_stack,
                StackFrame::serialize_for_proof
            ),
        ));

        out!(self.global_state.hash());

        out!(self.pc.module.to_be_bytes());
        out!(self.pc.func.to_be_bytes());
        out!(self.pc.inst.to_be_bytes());

        out!(self.thread_state.serialize());

        let mod_merkle = self.get_modules_merkle();
        out!(mod_merkle.root());

        if self.is_halted() {
            // If the machine is halted, instead of serializing the module,
            // serialize the global state and return.
            // This is for the "kickstart" BoLD proof, but it's backwards compatible
            // with the old OSP behavior which reads no further.
            out!(self.global_state.serialize());
            return data;
        }

        // End machine serialization, serialize module

        let module = &self.modules[self.pc.module()];
        let mem_merkle = module.memory.merkelize();
        out!(module.serialize_for_proof(&mem_merkle));

        // Prove module is in modules merkle tree

        out!(
            mod_merkle
                .prove(self.pc.module())
                .expect("Failed to prove module")
        );

        if self.is_halted() {
            return data;
        }

        // Begin next instruction proof

        let func = &module.funcs[self.pc.func()];
        out!(func.serialize_body_for_proof(self.pc));
        out!(
            func.code_merkle
                .prove(self.pc.inst() / Function::CHUNK_SIZE)
                .expect("Failed to prove against code merkle")
        );
        out!(
            module
                .funcs_merkle
                .prove(self.pc.func())
                .expect("Failed to prove against function merkle")
        );

        // End next instruction proof, begin instruction specific serialization

        let Some(next_inst) = func.code.get(self.pc.inst()) else {
            return data;
        };

        let op = next_inst.opcode;
        let arg = next_inst.argument_data;
        let value_stack = self.get_data_stack();
        let frame_stack = self.get_frame_stack();

        use Opcode::*;
        match op {
            GetGlobalStateU64 | SetGlobalStateU64 => {
                out!(self.global_state.serialize());
            }
            LocalGet | LocalSet => {
                let locals = &frame_stack.last().unwrap().locals;
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
                let base = match value_stack.get(value_stack.len() - 1 - stack_idx_offset) {
                    Some(Value::I32(x)) => *x,
                    x => fail!("memory index type is {x:?}"),
                };
                if let Some(mut idx) = u64::from(base)
                    .checked_add(arg)
                    .and_then(|x| usize::try_from(x).ok())
                {
                    // Prove the leaf this index is in, and the next one, if they are within the
                    // memory's size.
                    idx /= Memory::LEAF_SIZE;
                    out!(module.memory.get_leaf_data(idx));
                    out!(mem_merkle.prove(idx).unwrap_or_default());
                    // Now prove the next leaf too, in case it's accessed.
                    let next_leaf_idx = idx.saturating_add(1);
                    out!(module.memory.get_leaf_data(next_leaf_idx));
                    let second_mem_merkle = if is_store {
                        // For stores, prove the second merkle against a state after the first leaf
                        // is set. This state also happens to have the
                        // second leaf set, but that's irrelevant.
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
                let idx = match value_stack.last() {
                    Some(Value::I32(i)) => *i,
                    x => fail!("top of stack before call_indirect is {x:?}"),
                };
                let ty = &module.types[usize::try_from(ty).unwrap()];
                out!((table as u64).to_be_bytes());
                out!(ty.hash());
                let table_usize = usize::try_from(table).unwrap();
                let table = &module.tables[table_usize];
                out!(
                    table
                        .serialize_for_proof()
                        .expect("failed to serialize table")
                );
                out!(
                    module
                        .tables_merkle
                        .prove(table_usize)
                        .expect("Failed to prove tables merkle")
                );
                let idx_usize = usize::try_from(idx).unwrap();
                if let Some(elem) = table.elems.get(idx_usize) {
                    out!(elem.func_ty.hash());
                    out!(elem.val.serialize_for_proof());
                    out!(
                        table
                            .elems_merkle
                            .prove(idx_usize)
                            .expect("Failed to prove elements merkle")
                    );
                }
            }
            CrossModuleInternalCall => {
                let module_idx = value_stack.last().unwrap().assume_u32() as usize;
                let called_module = &self.modules[module_idx];
                out!(called_module.serialize_for_proof(&called_module.memory.merkelize()));
                out!(
                    mod_merkle
                        .prove(module_idx)
                        .expect("Failed to prove module for CrossModuleInternalCall")
                );
            }
            GetGlobalStateBytes32 | SetGlobalStateBytes32 => {
                out!(self.global_state.serialize());
                let ptr = value_stack.last().unwrap().assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    // Prove the leaf this index is in
                    idx /= Memory::LEAF_SIZE;
                    out!(module.memory.get_leaf_data(idx));
                    out!(mem_merkle.prove(idx).unwrap_or_default());
                }
            }
            ReadPreImage | ReadInboxMessage => {
                let offset = value_stack.last().unwrap().assume_u32();
                let ptr = value_stack.get(value_stack.len() - 2).unwrap().assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    // Prove the leaf this index is in
                    idx /= Memory::LEAF_SIZE;
                    let prev_data = module.memory.get_leaf_data(idx);
                    out!(prev_data);
                    out!(mem_merkle.prove(idx).unwrap_or_default());
                    if op == Opcode::ReadPreImage {
                        let hash = Bytes32(prev_data);
                        let preimage_ty = PreimageType::try_from(
                            u8::try_from(next_inst.argument_data)
                                .expect("ReadPreImage argument data is out of range for a u8"),
                        )
                        .expect("Invalid preimage type in ReadPreImage argument data");
                        let Some(preimage) =
                            self.preimage_resolver
                                .get_const(self.context, preimage_ty, hash)
                        else {
                            panic!("Missing requested preimage for hash {hash}")
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
                            PreimageType::DACertificate => {
                                // We do something special here; we don't create the final proof.
                                // For DACertificate preimages, signal that this proof needs
                                // enhancement Set the enhancement
                                // flag (0x80) on the machine status byte.
                                data[0] |= 0x80;

                                // Append hash and offset for the enhancer to use
                                data.extend(hash.0);
                                data.extend((offset as u64).to_be_bytes());

                                // Append marker to identify this as DACertificate ReadPreimage
                                data.push(0xDA);
                                // The enhancement flag and marker data will be stripped out of
                                // the proof by the enhancer.
                            }
                        }
                    } else if next_inst.opcode == Opcode::ReadInboxMessage {
                        let msg_idx = value_stack.get(value_stack.len() - 3).unwrap().assume_u64();
                        let inbox_identifier =
                            argument_data_to_inbox(arg).expect("Bad inbox indentifier");
                        if let Some(msg_data) =
                            self.inbox_contents.get(&(inbox_identifier, msg_idx))
                        {
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
                    let leaf_index = match value_stack.last() {
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
            PopCoThread => {
                macro_rules! prove_pop {
                    ($multistack:expr_2021, $hasher:expr_2021) => {
                        let len = $multistack.len();
                        if (len > 2) {
                            out!($hasher($multistack[len - 2]));
                        } else {
                            out!(Machine::NO_STACK_HASH);
                        }
                        if (len > 3) {
                            out!(hash_multistack(&$multistack[1..len - 2], $hasher));
                        } else {
                            out!(Bytes32::default());
                        }
                    };
                }
                prove_pop!(self.get_data_stacks(), hash_value_stack);
                prove_pop!(self.get_frame_stacks(), hash_stack_frame_stack);
            }
            ValidateCertificate => {
                // ValidateCertificate reads a hash from memory, so we need to prove that memory
                // access
                let ptr = value_stack.get(value_stack.len() - 2).unwrap().assume_u32();
                if let Some(mut idx) = usize::try_from(ptr).ok().filter(|x| x % 32 == 0) {
                    // Prove the leaf this index is in
                    idx /= Memory::LEAF_SIZE;
                    out!(module.memory.get_leaf_data(idx));
                    out!(mem_merkle.prove(idx).unwrap_or_default());
                }

                // Check if this is a DACertificate ValidateCertificate that needs enhancement
                let preimage_type = value_stack.last().unwrap().assume_u32();
                if let Ok(preimage_ty) = PreimageType::try_from(
                    u8::try_from(preimage_type)
                        .expect("ValidateCertificate preimage_type is out of range for u8"),
                ) && preimage_ty == PreimageType::DACertificate
                {
                    // We do something special here; we don't create the final proof.
                    // For DACertificate preimages, signal that this proof needs enhancement
                    // Set the enhancement flag (0x80) on the machine status byte.
                    data[0] |= 0x80;

                    // Load the hash from memory
                    if let Some(hash) = module.memory.load_32_byte_aligned(ptr.into()) {
                        // Append hash for the enhancer to use
                        data.extend(hash.0);

                        // Append marker to identify this as DACertificate ValidateCertificate
                        data.push(0xDB);
                        // The enhancement flag and marker data will be stripped out of
                        // the proof by the enhancer.
                    }
                }
            }
            _ => {}
        }
        data
    }

    pub fn get_data_stack(&self) -> &[Value] {
        match self.thread_state {
            ThreadState::Main => &self.value_stacks[0],
            ThreadState::CoThread(_) => self.value_stacks.last().unwrap(),
        }
    }

    pub fn get_data_stacks(&self) -> Vec<&[Value]> {
        self.value_stacks.iter().map(|v| v.as_slice()).collect()
    }

    fn get_frame_stack(&self) -> &[StackFrame] {
        match self.thread_state {
            ThreadState::Main => &self.frame_stacks[0],
            ThreadState::CoThread(_) => self.frame_stacks.last().unwrap(),
        }
    }

    fn get_frame_stacks(&self) -> Vec<&[StackFrame]> {
        self.frame_stacks
            .iter()
            .map(|v: &Vec<_>| v.as_slice())
            .collect()
    }

    pub fn get_internals_stack(&self) -> &[Value] {
        &self.internal_stack
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
            true => println!("{line}"),
            false => eprintln!("{line}"),
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
        let frame_stack = self.get_frame_stack();
        for frame in frame_stack.iter().rev().take(25) {
            if let Value::InternalRef(pc) = frame.return_ref {
                print_pc(pc);
            }
        }
        if frame_stack.len() > 25 {
            print(format!("  ... and {} more", frame_stack.len() - 25).grey());
        }
    }
}

#[cfg(test)]
mod wavm_format_tests {
    //! End-to-end tests for `Module::to_wavm_bytes` / `from_wavm_bytes`.
    //!
    //! These tests construct `Module`s directly (taking advantage of `pub(crate)`
    //! field access from inside `machine.rs`) so they exercise the full round-trip
    //! and merkle reconstruction without requiring a full Stylus toolchain.

    use std::sync::Arc;

    use arbutil::Bytes32;
    use brotli::Dictionary;
    use wasmparser::{RefType, TableType};

    use super::*;
    use crate::{
        memory::Memory,
        value::{ArbValueType, FunctionType, Value},
        wavm::{Instruction, Opcode},
        wavm_serialize::{WAVM_COMPRESSION_BROTLI, WAVM_MAGIC, WAVM_SERIALIZE_VERSION},
    };

    /// Build a small but non-trivial `Module` covering the consensus-relevant
    /// fields. The exact contents don't have to be executable — what matters is
    /// that `Module::hash()` inputs (globals, memory, `tables_merkle.root()`,
    /// `funcs_merkle.root()`, `extra_hash`, `internals_offset`) and the
    /// non-hashed but round-tripped fields are populated.
    fn build_test_module() -> Module {
        let mut memory = Memory::new(64, 1);
        memory.set_range(0, &[1, 2, 3, 4, 5, 6, 7, 8]).unwrap();
        memory.cache_merkle_tree();

        let function_type = FunctionType {
            inputs: vec![ArbValueType::I32, ArbValueType::I64],
            outputs: vec![ArbValueType::I32],
        };

        let func = Function::new_from_wavm(
            vec![
                Instruction {
                    opcode: Opcode::InitFrame,
                    argument_data: 0,
                    proving_argument_data: Some(Bytes32([0u8; 32])),
                },
                Instruction::with_data(Opcode::I32Const, 42),
                Instruction::simple(Opcode::Return),
            ],
            function_type.clone(),
            vec![ArbValueType::I32, ArbValueType::I64],
        );

        // Match the activator path (`Module::from_binary`): `elems_merkle` stays
        // empty so `Table::hash` sees `Merkle::default().root()` (all zeros), which
        // is what gets committed in `tables_merkle` on activation.
        let table = Table {
            ty: TableType {
                element_type: RefType::FUNCREF,
                initial: 2,
                maximum: Some(8),
                table64: false,
                shared: false,
            },
            elems: vec![
                TableElement {
                    func_ty: function_type.clone(),
                    val: Value::FuncRef(0),
                },
                TableElement::default(),
            ],
            elems_merkle: Merkle::default(),
        };
        let tables = vec![table];
        let tables_hashes: Result<_> = tables.iter().map(Table::hash).collect();
        let tables_merkle = Merkle::new(MerkleType::Table, tables_hashes.unwrap());

        let funcs = Arc::new(vec![func]);
        let funcs_merkle = Arc::new(Merkle::new(
            MerkleType::Function,
            funcs.iter().map(Function::hash).collect(),
        ));

        let mut func_exports: HashMap<String, u32> = HashMap::default();
        func_exports.insert("main".to_owned(), 0);

        let mut all_exports = ExportMap::default();
        all_exports.insert("main".to_owned(), (0, ExportKind::Func));

        Module {
            globals: vec![Value::I32(7), Value::I64(99), Value::RefNull],
            memory,
            tables,
            tables_merkle,
            funcs,
            funcs_merkle,
            types: Arc::new(vec![function_type.clone()]),
            internals_offset: 0,
            names: Arc::new(NameCustomSection {
                module: "test_module".to_owned(),
                functions: Default::default(),
            }),
            host_call_hooks: Arc::new(vec![None, Some(("env".to_owned(), "alloc".to_owned()))]),
            start_function: Some(0),
            func_types: Arc::new(vec![function_type]),
            func_exports: Arc::new(func_exports),
            all_exports: Arc::new(all_exports),
            extra_hash: Arc::new(Bytes32([0xAA; 32])),
        }
    }

    #[test]
    fn module_round_trip_preserves_hash() {
        // The core invariant: `Module::hash()` (consensus-locked) must be
        // identical after a serialize → deserialize round trip.
        let original = build_test_module();
        let original_hash = original.hash();

        let bytes = original.to_wavm_bytes().expect("encode should succeed");
        let rebuilt = Module::from_wavm_bytes(&bytes).expect("round-trip should succeed");

        assert_eq!(
            rebuilt.hash(),
            original_hash,
            "rebuilt module hash differs from original",
        );
    }

    fn assert_activation_round_trip(label: &str, wat: &[u8]) {
        let wasm = wasmer::wat2wasm(wat).expect("wat2wasm");

        let codehash = Bytes32::default();
        let stylus_version = 3u16;
        let mut gas = u64::MAX;
        let (module, _stylus_data) =
            Module::activate(&wasm, &codehash, stylus_version, 0, 65535, false, &mut gas)
                .expect("activation");

        let activation_hash = module.hash();
        let bytes = module
            .to_wavm_bytes()
            .expect("to_wavm_bytes after activation");
        let rebuilt = Module::from_wavm_bytes(&bytes).expect("from_wavm_bytes after activation");

        assert_eq!(
            rebuilt.hash(),
            activation_hash,
            "[{label}] deserialized hash must match the activator's hash so LinkModule's lookup-by-hash matches the rebuilt module's identity",
        );
    }

    #[test]
    fn activation_to_round_trip_hash_matches() {
        // Take a real Stylus user WASM through `Module::activate` (the
        // production activator path), serialize, deserialize, and assert hash
        // equality. The WAT declares a table with `initial > 0` and a
        // non-empty `elem` segment, so `Table::hash` is evaluated against a
        // non-trivially-shaped table; any future code that re-derives
        // `elems_merkle` from `elems` on one side but not the other would
        // change `tables_merkle.root()` and fail this assertion. (Today both
        // activator and deserializer hold `Merkle::default()` for this field;
        // the test guards against either side starting to populate it.)
        let minimal = br#"(module
            (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
            (memory (export "memory") 1 1)
            (table 2 funcref)
            (elem (i32.const 0) $user_entrypoint)
            (func $user_entrypoint (export "user_entrypoint") (param i32) (result i32)
                i32.const 0
            )
        )"#;
        assert_activation_round_trip("minimal", minimal);

        // Richer WAT: exercises IBinOp (add/sub/mul), IRelOp (eq),
        // MemoryLoad/MemoryStore, branching (if/else), a multi-entry function
        // table, and a non-entrypoint helper. This is the configuration that
        // catches "an encoding bug in a non-hashed sub-arg surfaces only on
        // contracts with control flow"; the minimal WAT above does not reach
        // any of these opcodes through its 1-instruction body.
        let rich = br#"(module
            (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
            (memory (export "memory") 1 1)
            (table 2 funcref)
            (elem (i32.const 0) $user_entrypoint $helper)
            (func $helper (param i32) (result i32)
                local.get 0
                i32.const 7
                i32.add
            )
            (func $user_entrypoint (export "user_entrypoint") (param i32) (result i32)
                (local $acc i32)
                ;; memory store + load
                i32.const 0
                i32.const 42
                i32.store
                i32.const 0
                i32.load
                local.set $acc
                ;; branch on argument
                local.get 0
                i32.const 0
                i32.eq
                if (result i32)
                    local.get $acc
                    i32.const 1
                    i32.add
                else
                    local.get $acc
                    i32.const 1
                    i32.sub
                end
                ;; call a helper to exercise an extra Function entry
                call $helper
                ;; final mul to exercise another IBinOp variant
                i32.const 3
                i32.mul
            )
        )"#;
        assert_activation_round_trip("rich", rich);
    }

    #[test]
    fn non_hashed_fields_round_trip() {
        // `Module::hash()` (consensus) only covers globals/memory/tables_merkle/
        // funcs_merkle/extra_hash/internals_offset. Several fields round-trip
        // through the wire format but never contribute to the hash, so a
        // serialize/deserialize bug in any of them would slip past the hash
        // assertion in `module_round_trip_preserves_hash`.
        //
        // The most important of these is `TableElement::{val, func_ty}`; for
        // activator-shaped modules `Table::hash` only commits `elems.len()` +
        // empty `elems_merkle.root()`, so per-element data is invisible to the
        // hash until a `Machine` rebuilds `elems_merkle`. A bug here would
        // surface only mid-execution, far from the encode/decode site.
        let original = build_test_module();
        let bytes = original.to_wavm_bytes().unwrap();
        let r = Module::from_wavm_bytes(&bytes).expect("decode");

        assert_eq!(r.types, original.types, "types");
        assert_eq!(r.func_types, original.func_types, "func_types");
        assert_eq!(
            r.internals_offset, original.internals_offset,
            "internals_offset"
        );
        assert_eq!(r.names.module, original.names.module, "names.module");
        assert_eq!(
            r.names.functions, original.names.functions,
            "names.functions"
        );
        assert_eq!(
            r.host_call_hooks, original.host_call_hooks,
            "host_call_hooks"
        );
        assert_eq!(r.start_function, original.start_function, "start_function");
        assert_eq!(r.func_exports, original.func_exports, "func_exports");
        // `all_exports` is a HashMap — compare via sorted keys to be order-insensitive.
        let mut got: Vec<_> = r.all_exports.iter().collect();
        let mut want: Vec<_> = original.all_exports.iter().collect();
        got.sort_by_key(|(k, _)| (*k).clone());
        want.sort_by_key(|(k, _)| (*k).clone());
        assert_eq!(got, want, "all_exports");

        // Per-function: `ty` and `local_types` are not in Function::hash().
        assert_eq!(r.funcs.len(), original.funcs.len(), "funcs.len()");
        for (rf, of) in r.funcs.iter().zip(original.funcs.iter()) {
            assert_eq!(rf.ty, of.ty, "Function::ty");
            assert_eq!(rf.local_types, of.local_types, "Function::local_types");
            // (Function::code itself IS covered by code_merkle in Function::hash,
            // which is in turn covered by funcs_merkle in module_round_trip_preserves_hash.)
        }

        // Per-table: TableType invariants beyond element_type + elems.len() are
        // not in Table::hash() (Table::hash uses elems.len() + elems_merkle.root()).
        assert_eq!(r.tables.len(), original.tables.len(), "tables.len()");
        for (rt, ot) in r.tables.iter().zip(original.tables.iter()) {
            assert_eq!(
                rt.ty.element_type, ot.ty.element_type,
                "TableType.element_type"
            );
            assert_eq!(rt.ty.initial, ot.ty.initial, "TableType.initial");
            assert_eq!(rt.ty.maximum, ot.ty.maximum, "TableType.maximum");
            assert_eq!(rt.ty.table64, ot.ty.table64, "TableType.table64");
            assert_eq!(rt.ty.shared, ot.ty.shared, "TableType.shared");

            // The highest-leverage assertion: per-element `val` and `func_ty`.
            // `Table::hash` only commits `elems_merkle.root()` (empty for
            // activator modules), so without this check a corrupt encoder/
            // decoder for `TableElement` would silently round-trip.
            assert_eq!(rt.elems.len(), ot.elems.len(), "table.elems.len()");
            for (re, oe) in rt.elems.iter().zip(ot.elems.iter()) {
                assert_eq!(re.val, oe.val, "TableElement.val");
                assert_eq!(re.func_ty, oe.func_ty, "TableElement.func_ty");
            }
        }

        // memory.max_size is covered by Memory::hash, but explicit here for clarity.
        assert_eq!(
            r.memory.max_size, original.memory.max_size,
            "memory.max_size"
        );
        assert_eq!(r.memory.size(), original.memory.size(), "memory.size()");
    }

    #[test]
    fn distinct_modules_differing_in_non_hashed_fields_hash_equally() {
        // Property: two modules that differ ONLY in fields outside `Module::hash`'s
        // commit set (globals + memory + tables_merkle + funcs_merkle + extra_hash
        // + internals_offset) must produce identical hashes. This is what catches
        // "a future PR adds a non-consensus field anywhere in the Module type
        // graph and on-chain `module_hash` silently shifts for every contract";
        // round-trip tests pass trivially in that scenario because encode→decode
        // is symmetric on the new field.
        let base_hash = build_test_module().hash();

        let check = |label: &str, m: &Module| {
            assert_eq!(
                m.hash(),
                base_hash,
                "[{label}] mutating a non-hashed field changed Module::hash() — \
                 either the mutation accidentally touched a hashed input, or this \
                 field actually IS in the hash",
            );
        };

        // 1. Module.types
        {
            let mut v = build_test_module();
            v.types = Arc::new(vec![]);
            check("Module.types", &v);
        }

        // 2. Module.names.module
        {
            let mut v = build_test_module();
            v.names = Arc::new(NameCustomSection {
                module: "a_different_module_name".to_owned(),
                functions: Default::default(),
            });
            check("Module.names.module", &v);
        }

        // 3. Module.names.functions
        {
            let mut v = build_test_module();
            let mut fnames: HashMap<u32, String> = HashMap::default();
            fnames.insert(0, "named_function".to_owned());
            v.names = Arc::new(NameCustomSection {
                module: "test_module".to_owned(),
                functions: fnames,
            });
            check("Module.names.functions", &v);
        }

        // 4. Module.host_call_hooks
        {
            let mut v = build_test_module();
            v.host_call_hooks =
                Arc::new(vec![Some(("env".to_owned(), "different_hook".to_owned()))]);
            check("Module.host_call_hooks", &v);
        }

        // 5. Module.start_function
        {
            let mut v = build_test_module();
            v.start_function = None;
            check("Module.start_function", &v);
        }

        // 6. Module.func_types
        {
            let mut v = build_test_module();
            v.func_types = Arc::new(vec![]);
            check("Module.func_types", &v);
        }

        // 7. Module.func_exports
        {
            let mut v = build_test_module();
            let mut e: HashMap<String, u32> = HashMap::default();
            e.insert("renamed_main".to_owned(), 0);
            v.func_exports = Arc::new(e);
            check("Module.func_exports", &v);
        }

        // 8. Module.all_exports
        {
            let mut v = build_test_module();
            let mut e = ExportMap::default();
            e.insert("renamed_main".to_owned(), (0, ExportKind::Func));
            v.all_exports = Arc::new(e);
            check("Module.all_exports", &v);
        }

        // 9. Function.ty and Function.local_types (keep code identical so the
        // function's code_merkle — and therefore Function::hash — is unchanged).
        // funcs_merkle is rebuilt from the mutated funcs to prove the property
        // rather than rely on a stale merkle root.
        {
            let mut v = build_test_module();
            let same_code = v.funcs[0].code.clone();
            let different_ty = FunctionType {
                inputs: vec![ArbValueType::F32],
                outputs: vec![ArbValueType::F64],
            };
            let different_locals = vec![ArbValueType::F32, ArbValueType::F64, ArbValueType::I32];
            let new_func = Function::new_from_wavm(same_code, different_ty, different_locals);
            v.funcs = Arc::new(vec![new_func]);
            v.funcs_merkle = Arc::new(Merkle::new(
                MerkleType::Function,
                v.funcs.iter().map(Function::hash).collect(),
            ));
            check("Function.ty + Function.local_types", &v);
        }

        // 10. TableElement.val and TableElement.func_ty. `Table::hash` reads
        // `elems_merkle.root()` + `elems.len()` + `ty`; for activator-shaped
        // tables `elems_merkle = Merkle::default()` so per-element bytes never
        // feed it. `elems.len()` and `ty` are unchanged here. Rebuild
        // tables_merkle from the mutated tables to prove the property.
        {
            let mut v = build_test_module();
            let different_func_ty = FunctionType {
                inputs: vec![ArbValueType::F32],
                outputs: vec![ArbValueType::F64],
            };
            v.tables[0].elems[0] = TableElement {
                func_ty: different_func_ty,
                val: Value::FuncRef(99),
            };
            let tables_hashes: Result<_> = v.tables.iter().map(Table::hash).collect();
            v.tables_merkle = Merkle::new(MerkleType::Table, tables_hashes.unwrap());
            check("TableElement.val + TableElement.func_ty", &v);
        }
    }

    #[test]
    fn module_serialization_is_canonical() {
        // Same logical module → same bytes. Catches HashMap-iteration-order leaks
        // and any other non-determinism creeping into the wire format.
        let module = build_test_module();
        let a = module.to_wavm_bytes().unwrap();
        let b = module.to_wavm_bytes().unwrap();
        assert_eq!(a, b, "two serializations of the same module differ");

        // And byte-for-byte identical to a fresh round-trip's re-serialization.
        let rebuilt = Module::from_wavm_bytes(&a).unwrap();
        let c = rebuilt.to_wavm_bytes().unwrap();
        assert_eq!(
            a, c,
            "round-tripped module re-serializes to different bytes"
        );
    }

    #[test]
    fn header_magic_and_version_are_emitted() {
        let bytes = build_test_module().to_wavm_bytes().unwrap();
        assert!(bytes.len() > 5);
        assert_eq!(&bytes[..4], WAVM_MAGIC);
        assert_eq!(bytes[4], WAVM_SERIALIZE_VERSION);
    }

    #[test]
    fn bad_magic_is_rejected() {
        let mut bytes = build_test_module().to_wavm_bytes().unwrap();
        bytes[0] = b'X';
        let err = Module::from_wavm_bytes(&bytes).unwrap_err();
        assert!(err.to_string().contains("magic"), "unexpected error: {err}");
    }

    #[test]
    fn wrong_version_is_rejected() {
        let mut bytes = build_test_module().to_wavm_bytes().unwrap();
        bytes[4] = WAVM_SERIALIZE_VERSION.wrapping_add(1);
        let err = Module::from_wavm_bytes(&bytes).unwrap_err();
        assert!(
            err.to_string().contains("WavmSerializeVersion"),
            "unexpected error: {err}"
        );
    }

    #[test]
    fn truncated_header_is_rejected() {
        // Just the magic, no version byte and no body.
        let bytes = WAVM_MAGIC.to_vec();
        let err = Module::from_wavm_bytes(&bytes).unwrap_err();
        assert!(
            err.to_string().contains("too short"),
            "unexpected error: {err}"
        );
    }

    #[test]
    fn empty_body_is_rejected() {
        // Valid header but no body — the decoder must error on the first missing
        // field rather than producing a half-built Module or panicking.
        let mut bytes = Vec::new();
        bytes.extend_from_slice(WAVM_MAGIC);
        bytes.push(WAVM_SERIALIZE_VERSION);

        let result = Module::from_wavm_bytes(&bytes);
        assert!(result.is_err(), "empty body should error: {result:?}");
    }

    #[test]
    fn truncation_anywhere_in_body_is_rejected() {
        // Build a valid module, then truncate the body at every byte boundary.
        // The positional decoder must report an error rather than panic for any
        // cut — coarser sweeps would miss truncations that land inside
        // u8/u16/u32/Bytes32 reads, which is where slice-indexing-without-bounds
        // regressions would surface.
        let bytes = build_test_module().to_wavm_bytes().unwrap();
        let header = WAVM_MAGIC.len() + 1;
        for cut in header + 1..bytes.len() {
            let result = Module::from_wavm_bytes(&bytes[..cut]);
            assert!(
                result.is_err(),
                "truncation at byte {cut} should error, got {result:?}",
            );
        }
    }

    #[test]
    fn trailing_garbage_is_rejected() {
        // Positional decoder consumes exactly the bytes it needs; anything left
        // over indicates either corruption or an encoder bug, and must error
        // rather than silently succeed.
        let mut bytes = build_test_module().to_wavm_bytes().unwrap();
        bytes.extend_from_slice(b"unexpected trailing content");
        let err = Module::from_wavm_bytes(&bytes).unwrap_err();
        assert!(
            err.to_string().contains("trailing"),
            "expected trailing-bytes error, got: {err}",
        );
    }

    #[test]
    fn unknown_compression_tag_is_rejected() {
        // Flip the compression tag byte (position MAGIC.len() + 1 = 5) to a
        // value the decoder doesn't recognise. The decoder must bail with the
        // tag value in the error rather than treating it as raw or brotli.
        let mut bytes = build_test_module().to_wavm_bytes().unwrap();
        bytes[WAVM_MAGIC.len() + 1] = 0xFF;
        let err = Module::from_wavm_bytes(&bytes).unwrap_err();
        let msg = err.to_string();
        assert!(
            msg.contains("compression tag") || msg.contains("compression"),
            "expected compression-tag error, got: {err}",
        );
    }

    #[test]
    fn raw_compression_tag_is_accepted() {
        // Operationally we keep the raw tag as a forward-compat hook: if brotli
        // ever needs to be disabled for a payload, the decoder must accept
        // tag=0 without bumping `WAVM_SERIALIZE_VERSION`. Manually construct a
        // raw-tagged envelope around the same inner body brotli would have
        // wrapped, and assert the decoded module hashes match the original.
        let original = build_test_module();
        let brotli_bytes = original.to_wavm_bytes().unwrap();

        // Decompress the brotli body to recover the raw inner body that the
        // encoder produced before wrapping it.
        let inner = {
            let mut env = Cursor::new(&brotli_bytes[WAVM_MAGIC.len() + 1..]);
            let tag = env.read_u8().unwrap();
            assert_eq!(tag, WAVM_COMPRESSION_BROTLI);
            let compressed = env.read_bytes().unwrap();
            brotli::decompress(&compressed, Dictionary::Empty).unwrap()
        };

        // Re-wrap with tag=0 (raw) and verify the decoder accepts it.
        let mut raw_envelope = Vec::with_capacity(WAVM_MAGIC.len() + 1 + 1 + 4 + inner.len());
        raw_envelope.extend_from_slice(WAVM_MAGIC);
        raw_envelope.push(WAVM_SERIALIZE_VERSION);
        raw_envelope.push(WAVM_COMPRESSION_NONE);
        raw_envelope.extend_from_slice(&(inner.len() as u32).to_be_bytes());
        raw_envelope.extend_from_slice(&inner);

        let rebuilt =
            Module::from_wavm_bytes(&raw_envelope).expect("raw-tagged envelope must decode");
        assert_eq!(
            rebuilt.hash(),
            original.hash(),
            "raw-tagged decode must yield the same module hash as brotli-tagged decode",
        );
    }

    #[test]
    fn corrupt_compressed_payload_is_rejected() {
        // Flip a byte inside the brotli stream. Decompression must fail
        // rather than producing partial / garbage output, and the error must
        // surface — not be silently fed to the inner cursor which could then
        // bail with a confusing wire-format error far from the real cause.
        let mut bytes = build_test_module().to_wavm_bytes().unwrap();
        // Skip header + tag + u32 length = 4 + 1 + 1 + 4 = 10 bytes; flip
        // a byte well inside the brotli payload.
        let target = WAVM_MAGIC.len() + 1 + 1 + 4 + 5;
        assert!(target < bytes.len(), "test setup: payload too small");
        bytes[target] ^= 0xFF;
        let err = Module::from_wavm_bytes(&bytes).unwrap_err();
        let msg = err.to_string();
        // The error should mention brotli — if we end up downstream of the
        // decompressor (e.g. with the inner cursor bailing on a garbage
        // count), the diagnostic for operators chasing a corrupt cache
        // entry becomes much harder to read.
        assert!(
            msg.contains("brotli"),
            "expected brotli decompression error, got: {err}",
        );
    }

    #[test]
    fn linkmodule_hash_mismatch_is_detectable() {
        // Pins the property the LinkModule runtime check (machine.rs ~line
        // 2785) relies on: if wasmdb is corrupted such that the bytes
        // stored under key K decode to a module whose hash is K' != K, the
        // mismatch is observable. The full bail! site requires driving a
        // `Machine` through the LinkModule opcode, which is heavy for a
        // unit test; this exercises the underlying detection logic
        // instead.
        //
        // Two distinct modules with different HASHED fields must produce
        // different `Module::hash()` outputs after a wire-format round trip.
        let mut module_a = build_test_module();
        let mut module_b = build_test_module();
        // Mutate a HASHED field on B so the hashes diverge. `extra_hash` is
        // covered by `Module::hash`, so changing it shifts B's hash.
        module_b.extra_hash = Arc::new(Bytes32([0xBB; 32]));

        let hash_a = module_a.hash();
        let hash_b = module_b.hash();
        assert_ne!(
            hash_a, hash_b,
            "test setup: A and B must have different hashes for the check to discriminate",
        );

        // Encode A's bytes and decode them. The decoded module's hash must
        // match A (the round-trip invariant) and must NOT match B (the
        // discriminating property the LinkModule check exploits).
        let bytes_a = module_a.to_wavm_bytes().expect("encode A");
        let decoded_hash = Module::from_wavm_bytes(&bytes_a).expect("decode A").hash();
        assert_eq!(
            decoded_hash, hash_a,
            "round-trip preserves the encoded module's hash",
        );
        assert_ne!(
            decoded_hash, hash_b,
            "if bytes for A were stored under B's wasmdb key, LinkModule's \
             `decoded.hash() != lookup_key` check would fire",
        );

        // Suppress unused-mut warnings; the modules are mutable so this
        // test can be extended to other hashed-field mutations without
        // restructuring.
        let _ = (&mut module_a, &mut module_b);
    }

    #[test]
    fn compression_actually_shrinks_realistic_modules() {
        // Activate a realistic WAT with branching, calls, and arithmetic.
        // The output of `to_wavm_bytes` must be smaller than the inner body
        // — otherwise compression is silently disabled and we've regressed
        // to the 20–80x disk growth the wire format was designed to avoid.
        // This is a guard against future changes that accidentally skip
        // the brotli step (e.g. swapping `compress` for a passthrough).
        let wat = br#"(module
            (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
            (memory (export "memory") 4 4)
            (table 4 funcref)
            (elem (i32.const 0) $user_entrypoint $h1 $h2 $h3)
            (func $h1 (param i32) (result i32) local.get 0 i32.const 1 i32.add)
            (func $h2 (param i32) (result i32) local.get 0 i32.const 2 i32.mul)
            (func $h3 (param i32) (result i32) local.get 0 i32.const 3 i32.sub)
            (func $user_entrypoint (export "user_entrypoint") (param i32) (result i32)
                (local $i i32) (local $acc i32)
                i32.const 0 local.set $acc
                i32.const 0 local.set $i
                (loop $L
                    local.get $i i32.const 4 i32.rem_s
                    local.get $acc i32.const 7 i32.add local.set $acc
                    drop
                    local.get $i i32.const 1 i32.add local.set $i
                    local.get $i i32.const 64 i32.lt_s br_if $L
                )
                local.get $acc
            )
        )"#;
        let wasm = wasmer::wat2wasm(wat).expect("wat2wasm");
        let codehash = Bytes32::default();
        let mut gas = u64::MAX;
        let (module, _) =
            Module::activate(&wasm, &codehash, 3u16, 0, 65535, false, &mut gas).expect("activate");

        let envelope = module.to_wavm_bytes().expect("to_wavm_bytes");
        // The envelope overhead is 10 bytes (MAGIC=4 + VERSION=1 + TAG=1 +
        // u32 LEN=4). Anything close to "inner body length + 10" means we
        // failed to compress.
        let envelope_overhead = WAVM_MAGIC.len() + 1 + 1 + 4;
        let compressed_payload_len = envelope.len() - envelope_overhead;

        // For a module of this shape, brotli q=0 reliably hits ~20x or
        // better. A 2x ratio is the floor we'd ever expect; if we drop
        // below that, something has gone badly wrong with compression.
        // Reconstruct the uncompressed body length the way the encoder
        // would have built it (round-trip through decoder, re-emit raw).
        let rebuilt = Module::from_wavm_bytes(&envelope).expect("from_wavm_bytes");
        let body_len = {
            // Re-emit then unwrap to count the inner body the encoder built.
            let again = rebuilt.to_wavm_bytes().unwrap();
            let mut env = Cursor::new(&again[WAVM_MAGIC.len() + 1..]);
            let _tag = env.read_u8().unwrap();
            let compressed = env.read_bytes().unwrap();
            brotli::decompress(&compressed, Dictionary::Empty)
                .unwrap()
                .len()
        };
        assert!(
            compressed_payload_len * 2 < body_len,
            "brotli payload ({compressed_payload_len}) must compress the body \
             ({body_len}) by at least 2x; if this fails, compression is silently \
             disabled or window settings have regressed",
        );
    }
}
