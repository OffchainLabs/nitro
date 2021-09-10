use crate::{
    lir::Opcode,
    value::{Value as LirValue, ValueType},
};
use nom::{
    branch::alt,
    bytes::streaming::tag,
    combinator::{eof, map, map_res, value},
    error::{context, ParseError, VerboseError},
    error::{Error, ErrorKind},
    multi::{count, length_data, many_till},
    sequence::{preceded, tuple},
    Err, Finish, Needed,
};
use nom_leb128::{leb128_i32, leb128_i64, leb128_u32, leb128_u64};

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
    LoadOrStore(Opcode, MemoryArg),
    Block(BlockType, Vec<HirInstruction>),
    Loop(BlockType, Vec<HirInstruction>),
    IfElse(BlockType, Vec<HirInstruction>, Vec<HirInstruction>),
    Branch(u32),
    BranchIf(u32),
    I32Const(i32),
    I64Const(i64),
    F32Const(f32),
    F64Const(f64),
}

impl HirInstruction {
    pub fn get_const_output(&self) -> Option<LirValue> {
        match *self {
            HirInstruction::I32Const(x) => Some(LirValue::I32(x as u32)),
            HirInstruction::I64Const(x) => Some(LirValue::I64(x as u64)),
            HirInstruction::F32Const(x) => Some(LirValue::F32(x)),
            HirInstruction::F64Const(x) => Some(LirValue::F64(x)),
            _ => None,
        }
    }
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct FunctionType {
    pub inputs: Vec<ValueType>,
    pub outputs: Vec<ValueType>,
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
pub enum WasmSection {
    /// Ignored (usually debugging info)
    Custom(Vec<u8>),
    /// A function type, denoted as (parameters, return values)
    Types(Vec<FunctionType>),
    Functions(Vec<u32>),
    Memories(Vec<Limits>),
    Globals(Vec<Global>),
    Start(u32),
    Code(Vec<Code>),
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

fn value_type(input: &[u8]) -> IResult<ValueType> {
    alt((
        value(ValueType::I32, tag(&[0x7F])),
        value(ValueType::I64, tag(&[0x7E])),
        value(ValueType::F32, tag(&[0x7D])),
        value(ValueType::F64, tag(&[0x7C])),
        value(ValueType::FuncRef, tag(&[0x70])),
        value(ValueType::ExternRef, tag(&[0x6F])),
    ))(input)
}

fn result_type(input: &[u8]) -> IResult<Vec<ValueType>> {
    wasm_vec(value_type)(input)
}

fn simple_opcode(input: &[u8]) -> IResult<Opcode> {
    alt((
        value(Opcode::Unreachable, tag(&[0x00])),
        value(Opcode::Nop, tag(&[0x01])),
        value(Opcode::Return, tag(&[0x0F])),
        value(Opcode::Drop, tag(&[0x1A])),
        value(Opcode::I32Add, tag(&[0x6A])),
        value(Opcode::I64Add, tag(&[0x7C])),
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
    ))(input)
}

fn call_instruction(input: &[u8]) -> IResult<HirInstruction> {
    preceded(tag(&[0x10]), map(leb128_u32, inst_with_idx(Opcode::Call)))(input)
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
            map(map(leb128_u32, f32::from_bits), HirInstruction::F32Const),
        ),
        preceded(
            tag(&[0x44]),
            map(map(leb128_u64, f64::from_bits), HirInstruction::F64Const),
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
    map(many_till(instruction, tag(&[0x0B])), |(x, _)| x)(input)
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

fn types_section(input: &[u8]) -> IResult<Vec<FunctionType>> {
    wasm_vec(function_type)(input)
}

fn functions_section(input: &[u8]) -> IResult<Vec<u32>> {
    wasm_vec(leb128_u32)(input)
}

fn memories_section(input: &[u8]) -> IResult<Vec<Limits>> {
    wasm_vec(limits)(input)
}

fn globals_section(input: &[u8]) -> IResult<Vec<Global>> {
    wasm_vec(global)(input)
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

fn code_section(input: &[u8]) -> IResult<Vec<Code>> {
    wasm_vec(code_func)(input)
}

fn section(mut input: &[u8]) -> IResult<WasmSection> {
    if input.is_empty() {
        return Err(Err::Incomplete(Needed::Unknown));
    }
    let section_type = input[0];
    input = &input[1..];
    let (remaining, data) = length_data(leb128_u32)(input)?;
    let (extra, sect) = match section_type {
        0 => Ok((input, WasmSection::Custom(data.into()))),
        1 => context("types section", map(types_section, WasmSection::Types))(data),
        3 => context(
            "functions section",
            map(functions_section, WasmSection::Functions),
        )(data),
        5 => context(
            "memories section",
            map(memories_section, WasmSection::Memories),
        )(data),
        6 => context(
            "globals section",
            map(globals_section, WasmSection::Globals),
        )(data),
        8 => context("start section", map(leb128_u32, WasmSection::Start))(data),
        10 => context("code section", map(code_section, WasmSection::Code))(data),
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
