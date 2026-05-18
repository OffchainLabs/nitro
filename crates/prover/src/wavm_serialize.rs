// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Stable wire format for persisting WAVM `Module`s in the wasmdb.
//!
//! The format is **positional**: a header (`WAVM_MAGIC` + `WAVM_SERIALIZE_VERSION`)
//! followed by every persisted field in `Module`'s declaration order, each with
//! its own internal length/count framing. There are **no per-field tags and no
//! skip-unknown machinery** — the schema is locked by `WAVM_SERIALIZE_VERSION`,
//! and any change to it requires bumping that version (the Go-side
//! `validateOrUpgradeWavmSerializeVersion` then purges incompatible entries on
//! next start).
//!
//! Opcodes are carried by their stable `Opcode::repr()` value via
//! `Opcode::from_repr` (see `wavm.rs`). Maps are sorted by key before emission
//! for byte-canonical output. `RefType` is carried as a stable 1-byte kind to
//! avoid depending on `wasmparser`'s internal struct layout.

use arbutil::Bytes32;
use eyre::{Result, bail, ensure};
use fnv::FnvHashMap as HashMap;
use wasmparser::{RefType, TableType};

use crate::{
    binary::{ExportKind, ExportMap, NameCustomSection},
    value::{ArbValueType, FunctionType, ProgramCounter, Value},
    wavm::{Instruction, Opcode},
};

// ===== format header =====

pub const WAVM_MAGIC: &[u8; 4] = b"WAVM";
// Bump on any on-disk shape change; the Go validator purges incompatible
// entries on next boot. Brotli is part of the contract — switching schemes
// requires a version bump.
pub const WAVM_SERIALIZE_VERSION: u32 = 1;

// ===== ref type kind byte =====
//
// `wasmparser::RefType` is an opaque struct whose byte layout is not part of
// its public API. Rather than `transmute` it, we encode the kind explicitly.
// Today only `FUNCREF` is reachable in WAVM modules (see `Module::from_binary`
// in `machine.rs`, which rejects other ref types). Adding a new variant later
// is an explicit, reviewable extension here.
const REF_TYPE_FUNCREF: u8 = 0x70;

// ===== primitive writers =====

pub fn write_u8(out: &mut Vec<u8>, v: u8) {
    out.push(v);
}

pub fn write_u16(out: &mut Vec<u8>, v: u16) {
    out.extend_from_slice(&v.to_be_bytes());
}

pub fn write_u32(out: &mut Vec<u8>, v: u32) {
    out.extend_from_slice(&v.to_be_bytes());
}

pub fn write_u64(out: &mut Vec<u8>, v: u64) {
    out.extend_from_slice(&v.to_be_bytes());
}

pub fn write_bytes32(out: &mut Vec<u8>, v: &Bytes32) {
    out.extend_from_slice(&v.0);
}

pub fn write_bytes(out: &mut Vec<u8>, bytes: &[u8]) -> Result<()> {
    write_count(out, bytes.len())?;
    out.extend_from_slice(bytes);
    Ok(())
}

pub fn write_str(out: &mut Vec<u8>, s: &str) -> Result<()> {
    write_bytes(out, s.as_bytes())
}

pub fn write_count(out: &mut Vec<u8>, n: usize) -> Result<()> {
    let n = u32::try_from(n).map_err(|_| eyre::eyre!("wavm encode: count {n} exceeds u32"))?;
    write_u32(out, n);
    Ok(())
}

pub fn write_optional_u32(out: &mut Vec<u8>, v: Option<u32>) {
    match v {
        Some(x) => {
            write_u8(out, 1);
            write_u32(out, x);
        }
        None => write_u8(out, 0),
    }
}

pub fn write_optional_u64(out: &mut Vec<u8>, v: Option<u64>) {
    match v {
        Some(x) => {
            write_u8(out, 1);
            write_u64(out, x);
        }
        None => write_u8(out, 0),
    }
}

pub fn write_optional_bytes32(out: &mut Vec<u8>, v: Option<&Bytes32>) {
    match v {
        Some(b) => {
            write_u8(out, 1);
            write_bytes32(out, b);
        }
        None => write_u8(out, 0),
    }
}

// ===== primitive readers (Cursor) =====

/// Lightweight byte cursor for stable decoding. Each `read_*` advances the position
/// and returns `Err` on truncation rather than panicking.
pub struct Cursor<'a> {
    data: &'a [u8],
    pos: usize,
}

impl<'a> Cursor<'a> {
    pub fn new(data: &'a [u8]) -> Self {
        Self { data, pos: 0 }
    }

    pub fn pos(&self) -> usize {
        self.pos
    }

    pub fn remaining(&self) -> usize {
        self.data.len().saturating_sub(self.pos)
    }

    pub fn is_empty(&self) -> bool {
        self.remaining() == 0
    }

    fn read_array<const N: usize>(&mut self) -> Result<[u8; N]> {
        ensure!(
            self.remaining() >= N,
            "wavm decode: needed {N} bytes, only {} remaining",
            self.remaining(),
        );
        let mut buf = [0u8; N];
        buf.copy_from_slice(&self.data[self.pos..self.pos + N]);
        self.pos += N;
        Ok(buf)
    }

    pub fn read_u8(&mut self) -> Result<u8> {
        Ok(self.read_array::<1>()?[0])
    }

    pub fn read_u16(&mut self) -> Result<u16> {
        Ok(u16::from_be_bytes(self.read_array::<2>()?))
    }

    pub fn read_u32(&mut self) -> Result<u32> {
        Ok(u32::from_be_bytes(self.read_array::<4>()?))
    }

    pub fn read_u64(&mut self) -> Result<u64> {
        Ok(u64::from_be_bytes(self.read_array::<8>()?))
    }

    pub fn read_bytes32(&mut self) -> Result<Bytes32> {
        Ok(Bytes32(self.read_array::<32>()?))
    }

    pub fn read_bytes(&mut self) -> Result<Vec<u8>> {
        let len = self.read_u32()? as usize;
        ensure!(
            self.remaining() >= len,
            "wavm decode: length-prefixed slice needs {len} bytes, only {} remaining",
            self.remaining(),
        );
        let v = self.data[self.pos..self.pos + len].to_vec();
        self.pos += len;
        Ok(v)
    }

    pub fn read_str(&mut self) -> Result<String> {
        let bytes = self.read_bytes()?;
        String::from_utf8(bytes).map_err(|e| eyre::eyre!("wavm decode: invalid UTF-8: {e}"))
    }

    pub fn read_optional_u32(&mut self) -> Result<Option<u32>> {
        let present = self.read_u8()?;
        match present {
            0 => Ok(None),
            1 => Ok(Some(self.read_u32()?)),
            other => bail!("wavm decode: optional u32 flag must be 0 or 1, got {other}"),
        }
    }

    pub fn read_optional_u64(&mut self) -> Result<Option<u64>> {
        let present = self.read_u8()?;
        match present {
            0 => Ok(None),
            1 => Ok(Some(self.read_u64()?)),
            other => bail!("wavm decode: optional u64 flag must be 0 or 1, got {other}"),
        }
    }

    pub fn read_optional_bytes32(&mut self) -> Result<Option<Bytes32>> {
        let present = self.read_u8()?;
        match present {
            0 => Ok(None),
            1 => Ok(Some(self.read_bytes32()?)),
            other => bail!("wavm decode: optional bytes32 flag must be 0 or 1, got {other}"),
        }
    }
}

/// Read a `u32` count and bound it against the cursor's remaining bytes,
/// preventing an attacker-controlled count from triggering a multi-GB
/// `Vec::with_capacity` before the per-element reads error out.
///
/// `min_elem_bytes` must be `>= 1`: every WAVM-encoded element carries at
/// least a tag byte (including `Option<...>` payloads, which spend a byte
/// on the present/absent flag).
pub fn read_count(c: &mut Cursor<'_>, min_elem_bytes: usize) -> Result<usize> {
    if min_elem_bytes == 0 {
        bail!("wavm decode: read_count called with min_elem_bytes = 0 (internal bug)");
    }
    let n = c.read_u32()? as usize;
    let needed = n.saturating_mul(min_elem_bytes);
    ensure!(
        needed <= c.remaining(),
        "wavm decode: count {n} \u{00d7} {min_elem_bytes} = {needed} bytes exceeds {} remaining",
        c.remaining(),
    );
    Ok(n)
}

// ===== ArbValueType =====

pub fn write_arb_value_type(out: &mut Vec<u8>, v: ArbValueType) {
    write_u8(out, v.serialize());
}

pub fn read_arb_value_type(c: &mut Cursor<'_>) -> Result<ArbValueType> {
    ArbValueType::from_u8(c.read_u8()?)
}

// ===== Value =====
// Variant id matches ArbValueType::serialize() so the wire is independent
// of Rust enum declaration order.

pub fn write_value(out: &mut Vec<u8>, v: Value) {
    write_arb_value_type(out, v.ty());
    match v {
        Value::I32(x) => write_u32(out, x),
        Value::I64(x) => write_u64(out, x),
        Value::F32(x) => write_u32(out, x.to_bits()),
        Value::F64(x) => write_u64(out, x.to_bits()),
        Value::RefNull => {}
        Value::FuncRef(x) => write_u32(out, x),
        Value::InternalRef(pc) => {
            write_u32(out, pc.module);
            write_u32(out, pc.func);
            write_u32(out, pc.inst);
        }
    }
}

pub fn read_value(c: &mut Cursor<'_>) -> Result<Value> {
    let ty = read_arb_value_type(c)?;
    Ok(match ty {
        ArbValueType::I32 => Value::I32(c.read_u32()?),
        ArbValueType::I64 => Value::I64(c.read_u64()?),
        ArbValueType::F32 => Value::F32(f32::from_bits(c.read_u32()?)),
        ArbValueType::F64 => Value::F64(f64::from_bits(c.read_u64()?)),
        ArbValueType::RefNull => Value::RefNull,
        ArbValueType::FuncRef => Value::FuncRef(c.read_u32()?),
        ArbValueType::InternalRef => Value::InternalRef(ProgramCounter {
            module: c.read_u32()?,
            func: c.read_u32()?,
            inst: c.read_u32()?,
        }),
    })
}

// ===== FunctionType =====

/// Minimum bytes per type list element: a single tag byte.
const FUNCTION_TYPE_MIN_ELEM_BYTES: usize = 1;

pub fn write_function_type(out: &mut Vec<u8>, ty: &FunctionType) -> Result<()> {
    write_count(out, ty.inputs.len())?;
    for t in &ty.inputs {
        write_arb_value_type(out, *t);
    }
    write_count(out, ty.outputs.len())?;
    for t in &ty.outputs {
        write_arb_value_type(out, *t);
    }
    Ok(())
}

pub fn read_function_type(c: &mut Cursor<'_>) -> Result<FunctionType> {
    let n_in = read_count(c, FUNCTION_TYPE_MIN_ELEM_BYTES)?;
    let mut inputs = Vec::with_capacity(n_in);
    for _ in 0..n_in {
        inputs.push(read_arb_value_type(c)?);
    }
    let n_out = read_count(c, FUNCTION_TYPE_MIN_ELEM_BYTES)?;
    let mut outputs = Vec::with_capacity(n_out);
    for _ in 0..n_out {
        outputs.push(read_arb_value_type(c)?);
    }
    Ok(FunctionType { inputs, outputs })
}

// ===== Instruction (Opcode via stable u16 repr, plus argument_data + optional proving data) =====

/// Minimum bytes per instruction on the wire: u16 repr + u64 argument_data + 1-byte
/// proving-data presence flag.
const INSTRUCTION_MIN_BYTES: usize = 2 + 8 + 1;

pub fn write_instruction(out: &mut Vec<u8>, inst: &Instruction) {
    write_u16(out, inst.opcode.repr());
    write_u64(out, inst.argument_data);
    write_optional_bytes32(out, inst.proving_argument_data.as_ref());
}

pub fn read_instruction(c: &mut Cursor<'_>) -> Result<Instruction> {
    let repr = c.read_u16()?;
    let opcode = Opcode::from_repr(repr)?;
    let argument_data = c.read_u64()?;
    let proving_argument_data = c.read_optional_bytes32()?;
    Ok(Instruction {
        opcode,
        argument_data,
        proving_argument_data,
    })
}

// ===== Function payload (locals + ty + instructions) =====

pub struct FunctionParts {
    pub local_types: Vec<ArbValueType>,
    pub ty: FunctionType,
    pub code: Vec<Instruction>,
}

pub fn write_function_parts(out: &mut Vec<u8>, parts: &FunctionParts) -> Result<()> {
    write_count(out, parts.local_types.len())?;
    for t in &parts.local_types {
        write_arb_value_type(out, *t);
    }
    write_function_type(out, &parts.ty)?;
    write_count(out, parts.code.len())?;
    for inst in &parts.code {
        write_instruction(out, inst);
    }
    Ok(())
}

pub fn read_function_parts(c: &mut Cursor<'_>) -> Result<FunctionParts> {
    let n_local = read_count(c, 1)?; // one tag byte per local
    let mut local_types = Vec::with_capacity(n_local);
    for _ in 0..n_local {
        local_types.push(read_arb_value_type(c)?);
    }
    let ty = read_function_type(c)?;
    let n_inst = read_count(c, INSTRUCTION_MIN_BYTES)?;
    let mut code = Vec::with_capacity(n_inst);
    for _ in 0..n_inst {
        code.push(read_instruction(c)?);
    }
    Ok(FunctionParts {
        local_types,
        ty,
        code,
    })
}

// ===== TableType =====
// Stable 1-byte kind for `RefType`; rejected if it doesn't match what
// `Module::from_binary` accepts (today: FUNCREF only).

pub fn write_table_type(out: &mut Vec<u8>, ty: &TableType) -> Result<()> {
    let kind = if ty.element_type == RefType::FUNCREF {
        REF_TYPE_FUNCREF
    } else {
        bail!(
            "wavm encode: unsupported RefType {:?}; only FUNCREF is allowed",
            ty.element_type,
        )
    };
    write_u8(out, kind);
    write_u64(out, ty.initial);
    write_optional_u64(out, ty.maximum);
    write_u8(out, ty.table64 as u8);
    write_u8(out, ty.shared as u8);
    Ok(())
}

pub fn read_table_type(c: &mut Cursor<'_>) -> Result<TableType> {
    let kind = c.read_u8()?;
    let element_type = match kind {
        REF_TYPE_FUNCREF => RefType::FUNCREF,
        other => bail!("wavm decode: unsupported RefType kind 0x{other:02X}"),
    };
    let initial = c.read_u64()?;
    let maximum = c.read_optional_u64()?;
    let table64 = match c.read_u8()? {
        0 => false,
        1 => true,
        other => bail!("wavm decode: table64 must be 0 or 1, got {other}"),
    };
    let shared = match c.read_u8()? {
        0 => false,
        1 => true,
        other => bail!("wavm decode: shared must be 0 or 1, got {other}"),
    };
    Ok(TableType {
        element_type,
        initial,
        maximum,
        table64,
        shared,
    })
}

// ===== TableElement parts (func_ty + val) =====

pub struct TableElementParts {
    pub func_ty: FunctionType,
    pub val: Value,
}

/// Minimum bytes per element: empty FunctionType (two zero u32 counts) + Value
/// tag byte. Conservative lower bound.
const TABLE_ELEMENT_MIN_BYTES: usize = 4 + 4 + 1;

pub fn write_table_element_parts(out: &mut Vec<u8>, te: &TableElementParts) -> Result<()> {
    write_function_type(out, &te.func_ty)?;
    write_value(out, te.val);
    Ok(())
}

pub fn read_table_element_parts(c: &mut Cursor<'_>) -> Result<TableElementParts> {
    let func_ty = read_function_type(c)?;
    let val = read_value(c)?;
    Ok(TableElementParts { func_ty, val })
}

// ===== Table parts (ty + elems) =====

pub struct TableParts {
    pub ty: TableType,
    pub elems: Vec<TableElementParts>,
}

pub fn write_table_parts(out: &mut Vec<u8>, tbl: &TableParts) -> Result<()> {
    write_table_type(out, &tbl.ty)?;
    write_count(out, tbl.elems.len())?;
    for e in &tbl.elems {
        write_table_element_parts(out, e)?;
    }
    Ok(())
}

pub fn read_table_parts(c: &mut Cursor<'_>) -> Result<TableParts> {
    let ty = read_table_type(c)?;
    let n = read_count(c, TABLE_ELEMENT_MIN_BYTES)?;
    let mut elems = Vec::with_capacity(n);
    for _ in 0..n {
        elems.push(read_table_element_parts(c)?);
    }
    Ok(TableParts { ty, elems })
}

// ===== ExportKind =====
// Explicit stable byte ids, not the Rust enum declaration order.

pub fn write_export_kind(out: &mut Vec<u8>, k: ExportKind) {
    let b: u8 = match k {
        ExportKind::Func => 0,
        ExportKind::Table => 1,
        ExportKind::Memory => 2,
        ExportKind::Global => 3,
        ExportKind::Tag => 4,
        ExportKind::FuncExact => 5,
    };
    write_u8(out, b);
}

pub fn read_export_kind(c: &mut Cursor<'_>) -> Result<ExportKind> {
    Ok(match c.read_u8()? {
        0 => ExportKind::Func,
        1 => ExportKind::Table,
        2 => ExportKind::Memory,
        3 => ExportKind::Global,
        4 => ExportKind::Tag,
        5 => ExportKind::FuncExact,
        other => bail!("wavm decode: unknown ExportKind byte {other}"),
    })
}

// ===== ExportMap (sorted-key for canonical bytes) =====

/// Minimum bytes per ExportMap entry: empty-string length prefix (4) + idx (4) +
/// kind byte (1). Conservative lower bound.
const EXPORT_MAP_MIN_ELEM_BYTES: usize = 4 + 4 + 1;

pub fn write_export_map(out: &mut Vec<u8>, m: &ExportMap) -> Result<()> {
    let mut entries: Vec<_> = m.iter().collect();
    entries.sort_by(|a, b| a.0.cmp(b.0));
    write_count(out, entries.len())?;
    for (name, (idx, kind)) in entries {
        write_str(out, name)?;
        write_u32(out, *idx);
        write_export_kind(out, *kind);
    }
    Ok(())
}

pub fn read_export_map(c: &mut Cursor<'_>) -> Result<ExportMap> {
    let n = read_count(c, EXPORT_MAP_MIN_ELEM_BYTES)?;
    let mut m = ExportMap::default();
    for _ in 0..n {
        let name = c.read_str()?;
        let idx = c.read_u32()?;
        let kind = read_export_kind(c)?;
        m.insert(name, (idx, kind));
    }
    Ok(m)
}

// ===== func_exports: HashMap<String, u32> (sorted-key for canonical bytes) =====

/// Minimum bytes per func_exports entry: empty-string length prefix (4) + idx (4).
const FUNC_EXPORTS_MIN_ELEM_BYTES: usize = 4 + 4;

pub fn write_func_exports(out: &mut Vec<u8>, m: &HashMap<String, u32>) -> Result<()> {
    let mut entries: Vec<_> = m.iter().collect();
    entries.sort_by(|a, b| a.0.cmp(b.0));
    write_count(out, entries.len())?;
    for (name, idx) in entries {
        write_str(out, name)?;
        write_u32(out, *idx);
    }
    Ok(())
}

pub fn read_func_exports(c: &mut Cursor<'_>) -> Result<HashMap<String, u32>> {
    let n = read_count(c, FUNC_EXPORTS_MIN_ELEM_BYTES)?;
    let mut m: HashMap<String, u32> = HashMap::default();
    m.reserve(n);
    for _ in 0..n {
        let name = c.read_str()?;
        let idx = c.read_u32()?;
        m.insert(name, idx);
    }
    Ok(m)
}

// ===== host_call_hooks: Vec<Option<(String, String)>> =====

/// Minimum bytes per hook entry: a single presence flag.
const HOOK_MIN_ELEM_BYTES: usize = 1;

pub fn write_host_call_hooks(out: &mut Vec<u8>, hooks: &[Option<(String, String)>]) -> Result<()> {
    write_count(out, hooks.len())?;
    for h in hooks {
        match h {
            Some((m, n)) => {
                write_u8(out, 1);
                write_str(out, m)?;
                write_str(out, n)?;
            }
            None => write_u8(out, 0),
        }
    }
    Ok(())
}

pub fn read_host_call_hooks(c: &mut Cursor<'_>) -> Result<Vec<Option<(String, String)>>> {
    let n = read_count(c, HOOK_MIN_ELEM_BYTES)?;
    let mut hooks = Vec::with_capacity(n);
    for _ in 0..n {
        let present = c.read_u8()?;
        hooks.push(match present {
            0 => None,
            1 => {
                let m = c.read_str()?;
                let nm = c.read_str()?;
                Some((m, nm))
            }
            other => bail!("wavm decode: host call hook flag must be 0 or 1, got {other}"),
        });
    }
    Ok(hooks)
}

// ===== NameCustomSection =====
//
// Both `module` (used by diagnostic helpers like `Machine::main_module_name`)
// and `functions` (used in error paths like `Machine::jump_into_func`'s name
// lookup) are non-consensus. They round-trip through the format so post-load
// diagnostics work without crashing.

/// Minimum bytes per function-name entry: func index (4) + empty-name length (4).
const FUNCTION_NAMES_MIN_ELEM_BYTES: usize = 4 + 4;

pub fn write_names(out: &mut Vec<u8>, names: &NameCustomSection) -> Result<()> {
    write_str(out, &names.module)?;
    let mut entries: Vec<_> = names.functions.iter().collect();
    entries.sort_by_key(|(idx, _)| *idx);
    write_count(out, entries.len())?;
    for (idx, name) in entries {
        write_u32(out, *idx);
        write_str(out, name)?;
    }
    Ok(())
}

pub fn read_names(c: &mut Cursor<'_>) -> Result<NameCustomSection> {
    let module = c.read_str()?;
    let n = read_count(c, FUNCTION_NAMES_MIN_ELEM_BYTES)?;
    let mut functions: HashMap<u32, String> = HashMap::default();
    functions.reserve(n);
    for _ in 0..n {
        let idx = c.read_u32()?;
        let name = c.read_str()?;
        functions.insert(idx, name);
    }
    Ok(NameCustomSection { module, functions })
}

#[cfg(test)]
mod tests {
    use ArbValueType::{F32, F64, I32 as VI32, I64 as VI64};

    use super::*;
    use crate::{
        value::IntegerValType::{I32, I64},
        wavm::{IBinOpType::*, IRelOpType::*, IUnOpType::*},
    };

    /// Exhaustive fixture of every WAVM `Opcode` value the wire format must handle.
    ///
    /// **Sync requirement** — when adding a new variant (or a new valid sub-argument
    /// combination) to `Opcode` in `wavm.rs`:
    /// 1. add a `repr()` mapping in `wavm.rs`,
    /// 2. add an `Opcode::from_repr` arm in `wavm.rs`,
    /// 3. add an entry below.
    ///
    /// The tests `every_known_opcode_round_trips` and
    /// `every_decodable_repr_round_trips` together guarantee that any divergence
    /// between this list, `repr()`, and `Opcode::from_repr` is caught at test time.
    fn all_known_opcodes() -> Vec<Opcode> {
        let mut ops = vec![
            Opcode::Unreachable,
            Opcode::Nop,
            Opcode::Return,
            Opcode::Call,
            Opcode::CallIndirect,
            Opcode::Drop,
            Opcode::Select,
            Opcode::LocalGet,
            Opcode::LocalSet,
            Opcode::GlobalGet,
            Opcode::GlobalSet,
            Opcode::MemorySize,
            Opcode::MemoryGrow,
            Opcode::I32Const,
            Opcode::I64Const,
            Opcode::F32Const,
            Opcode::F64Const,
            Opcode::I32Eqz,
            Opcode::I64Eqz,
            Opcode::I32WrapI64,
            Opcode::I64ExtendI32(true),
            Opcode::I64ExtendI32(false),
            Opcode::InitFrame,
            Opcode::ArbitraryJump,
            Opcode::ArbitraryJumpIf,
            Opcode::MoveFromStackToInternal,
            Opcode::MoveFromInternalToStack,
            Opcode::Dup,
            Opcode::CrossModuleCall,
            Opcode::CrossModuleForward,
            Opcode::CrossModuleInternalCall,
            Opcode::CallerModuleInternalCall,
            Opcode::GetGlobalStateBytes32,
            Opcode::SetGlobalStateBytes32,
            Opcode::GetGlobalStateU64,
            Opcode::SetGlobalStateU64,
            Opcode::ValidateCertificate,
            Opcode::ReadPreImage,
            Opcode::ReadInboxMessage,
            Opcode::LinkModule,
            Opcode::UnlinkModule,
            Opcode::HaltAndSetFinished,
            Opcode::NewCoThread,
            Opcode::PopCoThread,
            Opcode::SwitchThread,
        ];

        // MemoryLoad: every (ty, bytes, signed) combination that `repr()` maps explicitly.
        for (ty, bytes, signed) in [
            (VI32, 4, false),
            (VI64, 8, false),
            (F32, 4, false),
            (F64, 8, false),
            (VI32, 1, true),
            (VI32, 1, false),
            (VI32, 2, true),
            (VI32, 2, false),
            (VI64, 1, true),
            (VI64, 1, false),
            (VI64, 2, true),
            (VI64, 2, false),
            (VI64, 4, true),
            (VI64, 4, false),
        ] {
            ops.push(Opcode::MemoryLoad { ty, bytes, signed });
        }
        // MemoryStore: every (ty, bytes) combination that `repr()` maps explicitly.
        for (ty, bytes) in [
            (VI32, 4),
            (VI64, 8),
            (F32, 4),
            (F64, 8),
            (VI32, 1),
            (VI32, 2),
            (VI64, 1),
            (VI64, 2),
            (VI64, 4),
        ] {
            ops.push(Opcode::MemoryStore { ty, bytes });
        }

        // IRelOp: Eq/Ne ignore `signed` (canonical = false); Lt/Gt/Le/Ge use both signs.
        for w in [I32, I64] {
            ops.push(Opcode::IRelOp(w, Eq, false));
            ops.push(Opcode::IRelOp(w, Ne, false));
            for rel in [Lt, Gt, Le, Ge] {
                for signed in [true, false] {
                    ops.push(Opcode::IRelOp(w, rel, signed));
                }
            }
        }

        // IUnOp
        for w in [I32, I64] {
            for op in [Clz, Ctz, Popcnt] {
                ops.push(Opcode::IUnOp(w, op));
            }
        }

        // IBinOp: all 15 variants per integer width.
        for w in [I32, I64] {
            for op in [
                Add, Sub, Mul, DivS, DivU, RemS, RemU, And, Or, Xor, Shl, ShrS, ShrU, Rotl, Rotr,
            ] {
                ops.push(Opcode::IBinOp(w, op));
            }
        }

        // Reinterpret
        ops.push(Opcode::Reinterpret(VI32, F32));
        ops.push(Opcode::Reinterpret(VI64, F64));
        ops.push(Opcode::Reinterpret(F32, VI32));
        ops.push(Opcode::Reinterpret(F64, VI64));

        // ExtendS (the only widths that `repr()` maps explicitly)
        for x in [8u8, 16] {
            ops.push(Opcode::I32ExtendS(x));
        }
        for x in [8u8, 16, 32] {
            ops.push(Opcode::I64ExtendS(x));
        }

        ops
    }

    #[test]
    fn every_known_opcode_round_trips() {
        // Every Opcode value listed in `all_known_opcodes` must be decodable, and
        // re-encoding the decoded value must produce the same repr. This catches
        // the case where someone adds an opcode to `wavm.rs` (and to this fixture)
        // but forgets to add a matching arm to `Opcode::from_repr`.
        for op in all_known_opcodes() {
            let r = op.repr();
            let back = Opcode::from_repr(r).unwrap_or_else(|e| {
                panic!(
                    "Opcode::from_repr failed for known opcode {op:?} (repr 0x{r:04X}): {e}\n\
                     If you added a new Opcode variant in wavm.rs, also update Opcode::from_repr."
                )
            });
            assert_eq!(
                back.repr(),
                r,
                "opcode round-trip mismatch: {op:?} (repr 0x{r:04X}) decoded as {back:?} \
                 (repr 0x{:04X})",
                back.repr(),
            );
        }
    }

    #[test]
    fn every_decodable_repr_round_trips() {
        // Sweep every u16: if `Opcode::from_repr` accepts it, the decoded opcode's
        // `repr()` must echo the input. Catches mistakes where `from_repr` returns
        // the wrong opcode for some repr (e.g., copy-paste typo).
        for r in 0u16..=u16::MAX {
            if let Ok(op) = Opcode::from_repr(r) {
                assert_eq!(
                    op.repr(),
                    r,
                    "Opcode::from_repr(0x{r:04X}) returned {op:?} whose repr() is 0x{:04X}",
                    op.repr(),
                );
            }
        }
    }

    #[test]
    fn unknown_opcode_repr_errors_cleanly() {
        // Pick reprs that we know `Opcode::from_repr` doesn't handle and ensure
        // they produce a clean error rather than panicking or silently succeeding.
        for r in [
            0x02u16, 0x0E, 0x12, 0x25, 0x5B, 0x66, 0x8B, 0x9F, 0xC5, 0x8001,
        ] {
            let result = Opcode::from_repr(r);
            assert!(
                result.is_err(),
                "Opcode::from_repr(0x{r:04X}) unexpectedly succeeded: {:?}",
                result.ok(),
            );
        }
    }

    #[test]
    fn primitives_round_trip() {
        let mut out = Vec::new();
        write_u8(&mut out, 0xAB);
        write_u16(&mut out, 0x1234);
        write_u32(&mut out, 0xDEADBEEF);
        write_u64(&mut out, 0x0102030405060708);
        write_str(&mut out, "hello").unwrap();
        write_bytes(&mut out, &[1, 2, 3]).unwrap();
        write_optional_u32(&mut out, Some(42));
        write_optional_u32(&mut out, None);

        let mut c = Cursor::new(&out);
        assert_eq!(c.read_u8().unwrap(), 0xAB);
        assert_eq!(c.read_u16().unwrap(), 0x1234);
        assert_eq!(c.read_u32().unwrap(), 0xDEADBEEF);
        assert_eq!(c.read_u64().unwrap(), 0x0102030405060708);
        assert_eq!(c.read_str().unwrap(), "hello");
        assert_eq!(c.read_bytes().unwrap(), vec![1, 2, 3]);
        assert_eq!(c.read_optional_u32().unwrap(), Some(42));
        assert_eq!(c.read_optional_u32().unwrap(), None);
        assert!(c.is_empty());
    }

    #[test]
    fn value_round_trip_all_variants() {
        let values = [
            Value::I32(123),
            Value::I64(456),
            Value::F32(1.5),
            Value::F64(2.25),
            Value::RefNull,
            Value::FuncRef(7),
            Value::InternalRef(ProgramCounter {
                module: 1,
                func: 2,
                inst: 3,
            }),
        ];
        for v in values {
            let mut out = Vec::new();
            write_value(&mut out, v);
            let mut c = Cursor::new(&out);
            let got = read_value(&mut c).unwrap();
            assert_eq!(v, got);
        }
    }

    #[test]
    fn instruction_round_trip() {
        let inst = Instruction {
            opcode: Opcode::MemoryLoad {
                ty: ArbValueType::I32,
                bytes: 4,
                signed: false,
            },
            argument_data: 0x1234_5678_9ABC_DEF0,
            proving_argument_data: Some(Bytes32([0xAA; 32])),
        };
        let mut out = Vec::new();
        write_instruction(&mut out, &inst);
        let mut c = Cursor::new(&out);
        let got = read_instruction(&mut c).unwrap();
        assert_eq!(got.opcode.repr(), inst.opcode.repr());
        assert_eq!(got.argument_data, inst.argument_data);
        assert_eq!(got.proving_argument_data, inst.proving_argument_data);
    }

    #[test]
    fn read_count_rejects_unrealistic_count() {
        // A forged stream that claims a billion elements where only 4 bytes remain
        // must error before allocating, not OOM.
        let mut out = Vec::new();
        write_u32(&mut out, 1_000_000_000);
        let mut c = Cursor::new(&out);
        let result = read_count(&mut c, 1);
        assert!(
            result.is_err(),
            "read_count should reject impossible counts"
        );
    }

    #[test]
    fn read_count_rejects_zero_min_elem_bytes() {
        // min_elem_bytes = 0 is an internal-bug signal, not a valid call.
        let mut out = Vec::new();
        write_u32(&mut out, 0);
        let mut c = Cursor::new(&out);
        let err = read_count(&mut c, 0).unwrap_err().to_string();
        assert!(
            err.contains("min_elem_bytes = 0"),
            "expected read_count to bail on min_elem_bytes = 0, got: {err}",
        );
    }

    #[test]
    fn read_count_accepts_exact_match() {
        // n * min_elem_bytes == remaining() must succeed (bound is inclusive).
        let mut out = Vec::new();
        write_u32(&mut out, 3);
        out.extend_from_slice(&[0xAA, 0xBB, 0xCC]); // exactly 3 bytes follow
        let mut c = Cursor::new(&out);
        assert_eq!(read_count(&mut c, 1).unwrap(), 3);
    }

    #[test]
    fn read_count_rejects_off_by_one() {
        // n claims one byte more than is available; must reject.
        let mut out = Vec::new();
        write_u32(&mut out, 4);
        out.extend_from_slice(&[0xAA, 0xBB, 0xCC]); // only 3 bytes follow
        let mut c = Cursor::new(&out);
        assert!(
            read_count(&mut c, 1).is_err(),
            "off-by-one count should be rejected"
        );
    }

    #[test]
    fn read_count_saturating_mul_does_not_overflow() {
        // u32::MAX * 8 would wrap to a tiny value with `*`/`wrapping_mul`, but
        // `saturating_mul` caps at usize::MAX so the bound check rejects.
        // Catches a regression that swaps the saturating multiply for a normal
        // one — the test would otherwise pass and the decoder would OOM.
        let mut out = Vec::new();
        write_u32(&mut out, u32::MAX);
        out.push(0); // 1 byte of pretend payload
        let mut c = Cursor::new(&out);
        assert!(
            read_count(&mut c, 8).is_err(),
            "u32::MAX count with 8-byte elements must be rejected even though wrapping_mul would underflow",
        );
    }

    #[test]
    fn table_type_round_trip_funcref() {
        let ty = TableType {
            element_type: RefType::FUNCREF,
            initial: 7,
            maximum: Some(42),
            table64: false,
            shared: false,
        };
        let mut out = Vec::new();
        write_table_type(&mut out, &ty).unwrap();
        let mut c = Cursor::new(&out);
        let back = read_table_type(&mut c).unwrap();
        assert_eq!(back.element_type, ty.element_type);
        assert_eq!(back.initial, ty.initial);
        assert_eq!(back.maximum, ty.maximum);
        assert_eq!(back.table64, ty.table64);
        assert_eq!(back.shared, ty.shared);
    }

    #[test]
    fn table_type_rejects_unknown_ref_kind() {
        // Corrupted/forged byte stream with a non-FUNCREF kind byte must error,
        // not transmute into a poisoned `RefType` value.
        let bytes = vec![
            0x6F, // wrong kind (would be externref)
            0, 0, 0, 0, 0, 0, 0, 0, // initial = 0
            0, // maximum: None
            0, 0, // table64=false, shared=false
        ];
        let mut c = Cursor::new(&bytes);
        let result = read_table_type(&mut c);
        assert!(result.is_err(), "non-FUNCREF kind must be rejected");
    }

    #[test]
    fn names_round_trip() {
        let mut functions: HashMap<u32, String> = HashMap::default();
        functions.insert(2, "two".to_owned());
        functions.insert(0, "zero".to_owned());
        functions.insert(1, "one".to_owned());
        let names = NameCustomSection {
            module: "user".to_owned(),
            functions,
        };
        let mut out = Vec::new();
        write_names(&mut out, &names).unwrap();
        let mut c = Cursor::new(&out);
        let back = read_names(&mut c).unwrap();
        assert_eq!(back.module, names.module);
        assert_eq!(back.functions, names.functions);
        assert!(c.is_empty());
    }

    #[test]
    fn names_serialization_is_canonical() {
        // Functions inserted in different orders must serialize identically.
        let mut a: HashMap<u32, String> = HashMap::default();
        a.insert(5, "e".to_owned());
        a.insert(1, "a".to_owned());
        a.insert(3, "c".to_owned());
        let mut b: HashMap<u32, String> = HashMap::default();
        b.insert(3, "c".to_owned());
        b.insert(5, "e".to_owned());
        b.insert(1, "a".to_owned());
        let na = NameCustomSection {
            module: "x".to_owned(),
            functions: a,
        };
        let nb = NameCustomSection {
            module: "x".to_owned(),
            functions: b,
        };
        let mut bytes_a = Vec::new();
        let mut bytes_b = Vec::new();
        write_names(&mut bytes_a, &na).unwrap();
        write_names(&mut bytes_b, &nb).unwrap();
        assert_eq!(bytes_a, bytes_b, "name section must be deterministic");
    }
}
