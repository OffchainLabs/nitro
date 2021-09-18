use std::{collections::HashMap, hash::Hash};

use crate::{
    value::{FunctionType, IntegerValType, Value as LirValue, ValueType},
    wavm::{IBinOpType, IRelOpType, IUnOpType, Opcode},
};
use nom::{
    branch::alt,
    bytes::streaming::tag,
    combinator::{all_consuming, eof, map, map_res, value},
    error::{context, ParseError, VerboseError},
    error::{Error, ErrorKind, FromExternalError},
    multi::{count, length_data, many0, many_till},
    sequence::{preceded, tuple},
    Err, Finish, Needed,
};
use nom_leb128::{leb128_i32, leb128_i64, leb128_u32};

type IResult<'a, O> = nom::IResult<&'a [u8], O, VerboseError<&'a [u8]>>;

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum BlockType {
    Empty,
    ValueType(ValueType),
    TypeIndex(u32),
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct MemoryArg {
    pub alignment: u32,
    pub offset: u32,
}

#[derive(Clone, Debug, PartialEq)]
pub enum HirInstruction {
    Simple(Opcode),
    WithIdx(Opcode, u32),
    /// Separate from LocalGet and LocalSet (which are in WithIdx),
    /// as this is translated out of existence.
    LocalTee(u32),
    LoadOrStore(Opcode, MemoryArg),
    Block(BlockType, Vec<HirInstruction>),
    Loop(BlockType, Vec<HirInstruction>),
    IfElse(BlockType, Vec<HirInstruction>, Vec<HirInstruction>),
    Branch(u32),
    BranchIf(u32),
    BranchTable(Vec<u32>, u32),
    I32Const(i32),
    I64Const(i64),
    F32Const(f32),
    F64Const(f64),
    CallIndirect(u32, u32),
    /// Warning: internal and should not be parseable
    CrossModuleCall(u32, u32),
}

impl HirInstruction {
    pub fn get_const_output(&self) -> Option<LirValue> {
        match *self {
            HirInstruction::I32Const(x) => Some(LirValue::I32(x as u32)),
            HirInstruction::I64Const(x) => Some(LirValue::I64(x as u64)),
            HirInstruction::F32Const(x) => Some(LirValue::F32(x)),
            HirInstruction::F64Const(x) => Some(LirValue::F64(x)),
            HirInstruction::WithIdx(Opcode::FuncRefConst, x) => Some(LirValue::FuncRef(x)),
            _ => None,
        }
    }
}

#[derive(Clone, Debug)]
pub enum ImportKind {
    Function(u32),
    Table(u32),
    Memory(u32),
    Global(u32),
}

#[derive(Clone, Debug)]
pub struct Import {
    pub module: String,
    pub name: String,
    pub kind: ImportKind,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Limits {
    pub minimum_size: u32,
    pub maximum_size: Option<u32>,
}

#[derive(Clone, Debug)]
pub struct Global {
    pub value_type: ValueType,
    pub mutable: bool,
    pub initializer: Vec<HirInstruction>,
}

#[derive(Clone, Debug)]
pub struct Code {
    pub locals: Vec<ValueType>,
    pub expr: Vec<HirInstruction>,
}

#[derive(Clone, Debug)]
pub enum ExportKind {
    Function(u32),
    Table(u32),
    Memory(u32),
    Global(u32),
}

#[derive(Clone, Debug)]
pub struct Export {
    pub name: String,
    pub kind: ExportKind,
}

#[derive(Clone, Debug)]
pub struct DataMemoryLocation {
    pub memory: u32,
    pub offset: Vec<HirInstruction>,
}

#[derive(Clone, Debug)]
pub struct Data {
    pub data: Vec<u8>,
    pub active_location: Option<DataMemoryLocation>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum RefType {
    FuncRef,
    ExternRef,
}

impl Into<ValueType> for RefType {
    fn into(self) -> ValueType {
        match self {
            RefType::FuncRef => ValueType::FuncRef,
            RefType::ExternRef => panic!("Extern refs not supported"),
        }
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct TableType {
    pub ty: RefType,
    pub limits: Limits,
}

#[derive(Clone, Debug)]
pub enum ElementMode {
    Passive,
    Declarative,
    Active(u32, Vec<HirInstruction>),
}

#[derive(Clone, Debug)]
pub struct ElementSegment {
    pub ty: RefType,
    pub init: Vec<Vec<HirInstruction>>,
    pub mode: ElementMode,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct NameCustomSection {
    pub module: String,
    pub functions: HashMap<u32, String>,
    pub locals: HashMap<u32, HashMap<u32, String>>,
}

#[derive(Clone, Debug)]
pub enum CustomSection {
    Name(NameCustomSection),
    Unknown(String, Vec<u8>),
}

#[derive(Clone, Debug)]
pub enum WasmSection {
    Custom(CustomSection),
    /// A function type, denoted as (parameters, return values)
    Types(Vec<FunctionType>),
    Imports(Vec<Import>),
    Functions(Vec<u32>),
    Tables(Vec<TableType>),
    Memories(Vec<Limits>),
    Globals(Vec<Global>),
    Exports(Vec<Export>),
    Start(u32),
    Elements(Vec<ElementSegment>),
    Code(Vec<Code>),
    Datas(Vec<Data>),
    DataCount(u32),
}

#[derive(Clone, Debug)]
pub struct WasmBinary {
    pub sections: Vec<WasmSection>,
}

fn wasm_s33(input: &[u8]) -> IResult<i64> {
    let i64res = leb128_i64(input);
    if let Ok((_, num)) = i64res {
        if num < -(1 << 32) || num >= (1 << 32) {
            return Err(Err::Error(VerboseError::from_error_kind(
                input,
                ErrorKind::TooLarge,
            )));
        }
    }
    i64res
}

fn wasm_bool(input: &[u8]) -> IResult<bool> {
    alt((value(false, tag(&[0])), value(true, tag(&[1]))))(input)
}

fn wasm_vec<'a: 'b, 'b: 'a, T>(
    mut parser: impl FnMut(&'a [u8]) -> IResult<'a, T>,
) -> impl FnMut(&'b [u8]) -> IResult<'b, Vec<T>> {
    move |input| {
        let (input, len) = leb128_u32(input)?;
        count(&mut parser, len as usize)(input)
    }
}

fn wasm_map<'a: 'b, 'b: 'a, K: Hash + Eq, V>(
    key_parser: impl FnMut(&'a [u8]) -> IResult<'a, K>,
    value_parser: impl FnMut(&'a [u8]) -> IResult<'a, V>,
) -> impl FnMut(&'b [u8]) -> IResult<'b, HashMap<K, V>> {
    map(wasm_vec(tuple((key_parser, value_parser))), |v| {
        v.into_iter().collect()
    })
}

fn name(input: &[u8]) -> IResult<&str> {
    let (input, data) = length_data(leb128_u32)(input)?;
    let s = std::str::from_utf8(data)
        .map_err(|e| Err::Error(VerboseError::from_external_error(input, ErrorKind::Char, e)))?;
    Ok((input, s))
}

fn owned_name(input: &[u8]) -> IResult<String> {
    map(name, Into::into)(input)
}

fn ref_type(input: &[u8]) -> IResult<RefType> {
    alt((
        value(RefType::FuncRef, tag(&[0x70])),
        value(RefType::ExternRef, tag(&[0x6F])),
    ))(input)
}

fn value_type(input: &[u8]) -> IResult<ValueType> {
    alt((
        value(ValueType::I32, tag(&[0x7F])),
        value(ValueType::I64, tag(&[0x7E])),
        value(ValueType::F32, tag(&[0x7D])),
        value(ValueType::F64, tag(&[0x7C])),
        map(ref_type, Into::into),
    ))(input)
}

fn result_type(input: &[u8]) -> IResult<Vec<ValueType>> {
    wasm_vec(value_type)(input)
}

fn ibinop(ty: IntegerValType, opcode_offset: u8) -> impl Fn(&[u8]) -> IResult<Opcode> {
    move |mut input| {
        if input.is_empty() {
            return Err(Err::Incomplete(Needed::Unknown));
        }
        let byte = input[0];
        input = &input[1..];
        if byte < opcode_offset {
            return Err(Err::Error(VerboseError::from_error_kind(
                input,
                ErrorKind::Tag,
            )));
        }
        let op = match byte - opcode_offset {
            0 => IBinOpType::Add,
            1 => IBinOpType::Sub,
            2 => IBinOpType::Mul,
            3 => IBinOpType::DivS,
            4 => IBinOpType::DivU,
            5 => IBinOpType::RemS,
            6 => IBinOpType::RemU,
            7 => IBinOpType::And,
            8 => IBinOpType::Or,
            9 => IBinOpType::Xor,
            10 => IBinOpType::Shl,
            11 => IBinOpType::ShrS,
            12 => IBinOpType::ShrU,
            13 => IBinOpType::Rotl,
            14 => IBinOpType::Rotr,
            _ => {
                return Err(Err::Error(VerboseError::from_error_kind(
                    input,
                    ErrorKind::Tag,
                )));
            }
        };
        assert_eq!(op as u8, byte - opcode_offset);
        let opcode = Opcode::IBinOp(ty, op);
        assert_eq!(opcode.repr(), u16::from(byte));
        Ok((input, opcode))
    }
}

fn iunop(ty: IntegerValType, opcode_offset: u8) -> impl Fn(&[u8]) -> IResult<Opcode> {
    move |mut input| {
        if input.is_empty() {
            return Err(Err::Incomplete(Needed::Unknown));
        }
        let byte = input[0];
        input = &input[1..];
        if byte < opcode_offset {
            return Err(Err::Error(VerboseError::from_error_kind(
                input,
                ErrorKind::Tag,
            )));
        }
        let op = match byte - opcode_offset {
            0 => IUnOpType::Clz,
            1 => IUnOpType::Ctz,
            2 => IUnOpType::Popcnt,
            _ => {
                return Err(Err::Error(VerboseError::from_error_kind(
                    input,
                    ErrorKind::Tag,
                )));
            }
        };
        let opcode = Opcode::IUnOp(ty, op);
        assert_eq!(opcode.repr(), u16::from(byte));
        Ok((input, opcode))
    }
}

fn irelop(ty: IntegerValType, opcode_offset: u8) -> impl Fn(&[u8]) -> IResult<Opcode> {
    move |mut input| {
        if input.is_empty() {
            return Err(Err::Incomplete(Needed::Unknown));
        }
        let byte = input[0];
        input = &input[1..];
        if byte < opcode_offset {
            return Err(Err::Error(VerboseError::from_error_kind(
                input,
                ErrorKind::Tag,
            )));
        }
        let (op, signed) = match byte - opcode_offset {
            0 => (IRelOpType::Eq, false),
            1 => (IRelOpType::Ne, false),
            2 => (IRelOpType::Lt, true),
            3 => (IRelOpType::Lt, false),
            4 => (IRelOpType::Gt, true),
            5 => (IRelOpType::Gt, false),
            6 => (IRelOpType::Le, true),
            7 => (IRelOpType::Le, false),
            8 => (IRelOpType::Ge, true),
            9 => (IRelOpType::Ge, false),
            _ => {
                return Err(Err::Error(VerboseError::from_error_kind(
                    input,
                    ErrorKind::Tag,
                )));
            }
        };
        let opcode = Opcode::IRelOp(ty, op, signed);
        assert_eq!(opcode.repr(), u16::from(byte));
        Ok((input, opcode))
    }
}

fn simple_opcode(input: &[u8]) -> IResult<Opcode> {
    alt((
        value(Opcode::Unreachable, tag(&[0x00])),
        value(Opcode::Nop, tag(&[0x01])),
        value(Opcode::Return, tag(&[0x0F])),
        value(Opcode::Drop, tag(&[0x1A])),
        value(Opcode::Select, tag(&[0x1B])),
        value(Opcode::MemorySize, tag(&[0x3F, 0x00])),
        value(Opcode::MemoryGrow, tag(&[0x40, 0x00])),
        value(Opcode::I32Eqz, tag(&[0x45])),
        irelop(IntegerValType::I32, 0x46),
        value(Opcode::I64Eqz, tag(&[0x50])),
        irelop(IntegerValType::I64, 0x51),
        iunop(IntegerValType::I32, 0x67),
        ibinop(IntegerValType::I32, 0x6A),
        iunop(IntegerValType::I64, 0x79),
        ibinop(IntegerValType::I64, 0x7C),
        value(Opcode::I32WrapI64, tag(&[0xA7])),
        value(Opcode::I64ExtendI32(true), tag(&[0xAC])),
        value(Opcode::I64ExtendI32(false), tag(&[0xAD])),
    ))(input)
}

fn block_type(input: &[u8]) -> IResult<BlockType> {
    alt((
        value(BlockType::Empty, tag(&[0x40])),
        map(value_type, BlockType::ValueType),
        map_res(wasm_s33, |x| {
            if x.is_positive() {
                Ok(BlockType::TypeIndex(x as u32))
            } else {
                Err(Err::Error(Error::new(input, ErrorKind::Tag)))
            }
        }),
    ))(input)
}

fn block_instruction(input: &[u8]) -> IResult<HirInstruction> {
    alt((
        map(
            preceded(tag(&[0x02]), tuple((block_type, instructions))),
            |(t, i)| HirInstruction::Block(t, i),
        ),
        map(
            preceded(tag(&[0x03]), tuple((block_type, instructions))),
            |(t, i)| HirInstruction::Loop(t, i),
        ),
        map(
            preceded(tag(&[0x04]), tuple((block_type, instructions_with_else))),
            |(t, (i, e))| HirInstruction::IfElse(t, i, e),
        ),
    ))(input)
}

fn inst_with_idx(opcode: Opcode) -> impl Fn(u32) -> HirInstruction {
    move |i| HirInstruction::WithIdx(opcode, i)
}

fn branch_instruction(input: &[u8]) -> IResult<HirInstruction> {
    alt((
        preceded(tag(&[0x0C]), map(leb128_u32, HirInstruction::Branch)),
        preceded(tag(&[0x0D]), map(leb128_u32, HirInstruction::BranchIf)),
        preceded(
            tag(&[0x0E]),
            map(tuple((wasm_vec(leb128_u32), leb128_u32)), |(l, d)| {
                HirInstruction::BranchTable(l, d)
            }),
        ),
    ))(input)
}

fn call_instruction(input: &[u8]) -> IResult<HirInstruction> {
    alt((
        preceded(tag(&[0x10]), map(leb128_u32, inst_with_idx(Opcode::Call))),
        preceded(
            tag(&[0x11]),
            map(tuple((leb128_u32, leb128_u32)), |(y, x)| {
                HirInstruction::CallIndirect(x, y)
            }),
        ),
    ))(input)
}

fn variables_instruction(input: &[u8]) -> IResult<HirInstruction> {
    alt((
        preceded(
            tag(&[0x20]),
            map(leb128_u32, inst_with_idx(Opcode::LocalGet)),
        ),
        preceded(
            tag(&[0x21]),
            map(leb128_u32, inst_with_idx(Opcode::LocalSet)),
        ),
        preceded(
            tag(&[0x22]),
            map(leb128_u32, |x| HirInstruction::LocalTee(x)),
        ),
        preceded(
            tag(&[0x23]),
            map(leb128_u32, inst_with_idx(Opcode::GlobalGet)),
        ),
        preceded(
            tag(&[0x24]),
            map(leb128_u32, inst_with_idx(Opcode::GlobalSet)),
        ),
    ))(input)
}

fn memory_arg(input: &[u8]) -> IResult<MemoryArg> {
    map(tuple((leb128_u32, leb128_u32)), |(a, o)| MemoryArg {
        alignment: a,
        offset: o,
    })(input)
}

fn load_instruction(input: &[u8]) -> IResult<HirInstruction> {
    macro_rules! mload_matcher {
        { $($x:literal => ($t:ident, $b:literal, $s:literal),)* } => {
            alt((
                $(
                    value(Opcode::MemoryLoad {
                        ty: ValueType::$t,
                        bytes: $b,
                        signed: $s,
                    }, tag(&[$x])),
                )*
            ))
        }
    }
    let opcode = mload_matcher! {
        0x28 => (I32, 4, false),
        0x29 => (I64, 8, false),
        0x2A => (F32, 4, false),
        0x2B => (F64, 8, false),
        0x2C => (I32, 1, true),
        0x2D => (I32, 1, false),
        0x2E => (I32, 2, true),
        0x2F => (I32, 2, false),
        0x30 => (I64, 1, true),
        0x31 => (I64, 1, false),
        0x32 => (I64, 2, true),
        0x33 => (I64, 2, false),
        0x34 => (I64, 4, true),
        0x35 => (I64, 4, false),
    };
    map(tuple((opcode, memory_arg)), |(op, arg)| {
        HirInstruction::LoadOrStore(op, arg)
    })(input)
}

fn store_instruction(input: &[u8]) -> IResult<HirInstruction> {
    macro_rules! mstore_matcher {
        { $($x:literal => ($t:ident, $b:literal),)* } => {
            alt((
                $(
                    value(Opcode::MemoryStore {
                        ty: ValueType::$t,
                        bytes: $b,
                    }, tag(&[$x])),
                )*
            ))
        }
    }
    let opcode = mstore_matcher! {
        0x36 => (I32, 4),
        0x37 => (I64, 8),
        0x38 => (F32, 4),
        0x39 => (F64, 8),
        0x3A => (I32, 1),
        0x3B => (I32, 2),
        0x3C => (I64, 1),
        0x3D => (I64, 2),
        0x3E => (I64, 4),
    };
    map(tuple((opcode, memory_arg)), |(op, arg)| {
        HirInstruction::LoadOrStore(op, arg)
    })(input)
}

fn const_instruction(input: &[u8]) -> IResult<HirInstruction> {
    alt((
        preceded(tag(&[0x41]), map(leb128_i32, HirInstruction::I32Const)),
        preceded(tag(&[0x42]), map(leb128_i64, HirInstruction::I64Const)),
        preceded(
            tag(&[0x43]),
            map(
                map(nom::number::streaming::le_u32, f32::from_bits),
                HirInstruction::F32Const,
            ),
        ),
        preceded(
            tag(&[0x44]),
            map(
                map(nom::number::streaming::le_u64, f64::from_bits),
                HirInstruction::F64Const,
            ),
        ),
    ))(input)
}

fn instruction(input: &[u8]) -> IResult<HirInstruction> {
    alt((
        map(simple_opcode, HirInstruction::Simple),
        block_instruction,
        branch_instruction,
        call_instruction,
        variables_instruction,
        load_instruction,
        store_instruction,
        const_instruction,
    ))(input)
}

fn instructions(input: &[u8]) -> IResult<Vec<HirInstruction>> {
    map(
        many_till(context("instruction", instruction), tag(&[0x0B])),
        |(x, _)| x,
    )(input)
}

fn instructions_with_else(input: &[u8]) -> IResult<(Vec<HirInstruction>, Vec<HirInstruction>)> {
    let term_parser = alt((tag(&[0x05]), tag(&[0x0B])));
    let (mut input, (if_instructions, terminator)) = many_till(instruction, term_parser)(input)?;
    let mut else_instructions = Vec::new();
    if terminator == &[0x05] {
        let res = instructions(input)?;
        input = res.0;
        else_instructions = res.1;
    }
    Ok((input, (if_instructions, else_instructions)))
}

fn function_type(input: &[u8]) -> IResult<FunctionType> {
    let inner = map(tuple((result_type, result_type)), |(i, o)| FunctionType {
        inputs: i,
        outputs: o,
    });
    preceded(tag(&[0x60]), inner)(input)
}

fn global(input: &[u8]) -> IResult<Global> {
    map(tuple((value_type, wasm_bool, instructions)), |(t, m, i)| {
        Global {
            value_type: t,
            mutable: m,
            initializer: i,
        }
    })(input)
}

fn locals(input: &[u8]) -> IResult<Vec<ValueType>> {
    map(wasm_vec(tuple((leb128_u32, value_type))), |v| {
        v.into_iter()
            .flat_map(|(c, t)| std::iter::repeat(t).take(c as usize))
            .collect::<Vec<_>>()
    })(input)
}

fn limits(input: &[u8]) -> IResult<Limits> {
    let no_max = map(leb128_u32, |x| Limits {
        minimum_size: x,
        maximum_size: None,
    });
    let with_max = map(tuple((leb128_u32, leb128_u32)), |(x, y)| Limits {
        minimum_size: x,
        maximum_size: Some(y),
    });
    alt((
        preceded(tag(&[0x00]), no_max),
        preceded(tag(&[0x01]), with_max),
    ))(input)
}

fn export_kind(input: &[u8]) -> IResult<ExportKind> {
    alt((
        map(preceded(tag(&[0x00]), leb128_u32), ExportKind::Function),
        map(preceded(tag(&[0x01]), leb128_u32), ExportKind::Table),
        map(preceded(tag(&[0x02]), leb128_u32), ExportKind::Memory),
        map(preceded(tag(&[0x03]), leb128_u32), ExportKind::Global),
    ))(input)
}

fn export(input: &[u8]) -> IResult<Export> {
    map(tuple((name, export_kind)), |(n, k)| Export {
        name: n.into(),
        kind: k,
    })(input)
}

fn code_func(input: &[u8]) -> IResult<Code> {
    let (remaining, input) = length_data(leb128_u32)(input)?;
    let (extra, code) = map(tuple((locals, instructions)), |(l, i)| Code {
        locals: l,
        expr: i,
    })(input)?;
    if !extra.is_empty() {
        return Err(Err::Error(VerboseError::from_error_kind(
            extra,
            ErrorKind::Eof,
        )));
    }
    Ok((remaining, code))
}

fn element_segment(mut input: &[u8]) -> IResult<ElementSegment> {
    let format = match input.get(0) {
        Some(x) if *x < 8 => *x,
        _ => {
            return Err(Err::Incomplete(Needed::Unknown));
        }
    };
    input = &input[1..];
    let (input, mode) = match format & 3 {
        0 => map(instructions, |o| ElementMode::Active(0, o))(input),
        1 => Ok((input, ElementMode::Passive)),
        2 => map(tuple((leb128_u32, instructions)), |(t, o)| {
            ElementMode::Active(t, o)
        })(input),
        3 => Ok((input, ElementMode::Declarative)),
        _ => unreachable!(),
    }?;
    let ref_general = format & 4 != 0;
    let (input, ty) = if format & 3 == 0 {
        Ok((input, RefType::FuncRef))
    } else if ref_general {
        ref_type(input)
    } else {
        value(RefType::FuncRef, tag(&[0x00]))(input)
    }?;
    let (input, init) = wasm_vec(|input| {
        if ref_general {
            instructions(input)
        } else {
            map(leb128_u32, |i| {
                vec![HirInstruction::WithIdx(Opcode::FuncRefConst, i)]
            })(input)
        }
    })(input)?;
    Ok((input, ElementSegment { ty, mode, init }))
}

fn data_segment(input: &[u8]) -> IResult<Data> {
    alt((
        map(
            tuple((tag(&[0x00]), instructions, length_data(leb128_u32))),
            |(_, offset, data)| Data {
                data: data.into(),
                active_location: Some(DataMemoryLocation { memory: 0, offset }),
            },
        ),
        map(
            tuple((tag(&[0x01]), length_data(leb128_u32))),
            |(_, data): (_, &[u8])| Data {
                data: data.into(),
                active_location: None,
            },
        ),
        map(
            tuple((
                tag(&[0x02]),
                leb128_u32,
                instructions,
                length_data(leb128_u32),
            )),
            |(_, memory, offset, data)| Data {
                data: data.into(),
                active_location: Some(DataMemoryLocation { memory, offset }),
            },
        ),
    ))(input)
}

fn import_kind(input: &[u8]) -> IResult<ImportKind> {
    alt((
        preceded(tag(&[0x00]), map(leb128_u32, ImportKind::Function)),
        preceded(tag(&[0x01]), map(leb128_u32, ImportKind::Table)),
        preceded(tag(&[0x02]), map(leb128_u32, ImportKind::Memory)),
        preceded(tag(&[0x03]), map(leb128_u32, ImportKind::Global)),
    ))(input)
}

fn tables_section(input: &[u8]) -> IResult<Vec<TableType>> {
    wasm_vec(map(tuple((ref_type, limits)), |(t, l)| TableType {
        ty: t,
        limits: l,
    }))(input)
}

fn import(input: &[u8]) -> IResult<Import> {
    map(
        tuple((owned_name, owned_name, import_kind)),
        |(module, name, kind)| Import { module, name, kind },
    )(input)
}

fn name_custom_section(input: &[u8]) -> IResult<NameCustomSection> {
    let (extra, sections) = many0(tuple((
        nom::bytes::complete::take(1usize),
        length_data(leb128_u32),
    )))(input)?;
    let mut names = NameCustomSection::default();
    let mut last_sect_id = None;
    for (id, sect) in sections {
        let id = id[0];
        if matches!(last_sect_id, Some(x) if x >= id) {
            return Err(Err::Error(VerboseError::from_error_kind(
                input,
                ErrorKind::Verify,
            )));
        }
        last_sect_id = Some(id);
        match id {
            0 => {
                let (_, module) = all_consuming(owned_name)(sect)?;
                names.module = module;
            }
            1 => {
                let (_, functions) = all_consuming(wasm_map(leb128_u32, owned_name))(sect)?;
                names.functions = functions;
            }
            2 => {
                let (_, locals) =
                    all_consuming(wasm_map(leb128_u32, wasm_map(leb128_u32, owned_name)))(sect)?;
                names.locals = locals;
            }
            _ => {}
        }
    }
    Ok((extra, names))
}

fn custom_section(input: &[u8]) -> IResult<CustomSection> {
    let (input, sect_name) = name(input)?;
    match sect_name {
        "name" => map(name_custom_section, CustomSection::Name)(input),
        _ => Ok((&[], CustomSection::Unknown(sect_name.into(), input.into()))),
    }
}

fn section(mut input: &[u8]) -> IResult<WasmSection> {
    if input.is_empty() {
        return Err(Err::Incomplete(Needed::Unknown));
    }
    let section_type = input[0];
    input = &input[1..];
    let (remaining, data) = length_data(leb128_u32)(input)?;
    let (extra, sect) = match section_type {
        0 => context("custom section", map(custom_section, WasmSection::Custom))(data),
        1 => context(
            "types section",
            map(wasm_vec(function_type), WasmSection::Types),
        )(data),
        2 => context(
            "imports section",
            map(wasm_vec(import), WasmSection::Imports),
        )(data),
        3 => context(
            "functions section",
            map(wasm_vec(leb128_u32), WasmSection::Functions),
        )(data),
        4 => context("tables section", map(tables_section, WasmSection::Tables))(data),
        5 => context(
            "memories section",
            map(wasm_vec(limits), WasmSection::Memories),
        )(data),
        6 => context(
            "globals section",
            map(wasm_vec(global), WasmSection::Globals),
        )(data),
        7 => context(
            "exports section",
            map(wasm_vec(export), WasmSection::Exports),
        )(data),
        8 => context("start section", map(leb128_u32, WasmSection::Start))(data),
        9 => context(
            "elements section",
            map(wasm_vec(element_segment), WasmSection::Elements),
        )(data),
        10 => context("code section", map(wasm_vec(code_func), WasmSection::Code))(data),
        11 => context(
            "data section",
            map(wasm_vec(data_segment), WasmSection::Datas),
        )(data),
        12 => context(
            "data count section",
            map(leb128_u32, WasmSection::DataCount),
        )(data),
        _ => Err(Err::Error(VerboseError::from_error_kind(
            input,
            ErrorKind::Tag,
        ))),
    }?;
    if !extra.is_empty() {
        return Err(Err::Error(VerboseError::from_error_kind(
            extra,
            ErrorKind::Eof,
        )));
    }
    Ok((remaining, sect))
}

const HEADER: &[u8] = b"\0asm\x01\0\0\0";

fn module(mut input: &[u8]) -> IResult<WasmBinary> {
    input = tag(HEADER)(input)?.0;
    map(many_till(section, eof), |(s, _)| WasmBinary { sections: s })(input)
}

pub fn parse(input: &[u8]) -> Result<WasmBinary, nom::error::VerboseError<&[u8]>> {
    module(input).finish().map(|(_, x)| x)
}
