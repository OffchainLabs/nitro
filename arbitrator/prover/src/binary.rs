// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::value::{ArbValueType, FunctionType, IntegerValType, Value as LirValue};
use eyre::{bail, ensure, Result};
use fnv::FnvHashMap as HashMap;
use nom::{
    branch::alt,
    bytes::complete::tag,
    combinator::{all_consuming, map, value},
    sequence::{preceded, tuple},
};
use serde::{Deserialize, Serialize};
use std::{convert::TryInto, hash::Hash, str::FromStr};
use wasmparser::{
    Data, Element, Export, Global, Import, MemoryType, Name, NameSectionReader, Naming, Operator,
    Parser, Payload, TableType, TypeDef,
};

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum FloatType {
    F32,
    F64,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum FloatUnOp {
    Abs,
    Neg,
    Ceil,
    Floor,
    Trunc,
    Nearest,
    Sqrt,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum FloatBinOp {
    Add,
    Sub,
    Mul,
    Div,
    Min,
    Max,
    CopySign,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum FloatRelOp {
    Eq,
    Ne,
    Lt,
    Gt,
    Le,
    Ge,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash)]
pub enum FloatInstruction {
    UnOp(FloatType, FloatUnOp),
    BinOp(FloatType, FloatBinOp),
    RelOp(FloatType, FloatRelOp),
    /// The bools represent (saturating, signed)
    TruncIntOp(IntegerValType, FloatType, bool, bool),
    ConvertIntOp(FloatType, IntegerValType, bool),
    F32DemoteF64,
    F64PromoteF32,
}

impl FloatInstruction {
    pub fn signature(&self) -> FunctionType {
        match *self {
            FloatInstruction::UnOp(t, _) => FunctionType::new(vec![t.into()], vec![t.into()]),
            FloatInstruction::BinOp(t, _) => FunctionType::new(vec![t.into(); 2], vec![t.into()]),
            FloatInstruction::RelOp(t, _) => {
                FunctionType::new(vec![t.into(); 2], vec![ArbValueType::I32])
            }
            FloatInstruction::TruncIntOp(i, f, ..) => {
                FunctionType::new(vec![f.into()], vec![i.into()])
            }
            FloatInstruction::ConvertIntOp(f, i, _) => {
                FunctionType::new(vec![i.into()], vec![f.into()])
            }
            FloatInstruction::F32DemoteF64 => {
                FunctionType::new(vec![ArbValueType::F64], vec![ArbValueType::F32])
            }
            FloatInstruction::F64PromoteF32 => {
                FunctionType::new(vec![ArbValueType::F32], vec![ArbValueType::F64])
            }
        }
    }
}

impl FromStr for FloatInstruction {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        type IResult<'a, T> = nom::IResult<&'a str, T, nom::error::Error<&'a str>>;

        fn parse_fp_type(s: &str) -> IResult<FloatType> {
            alt((
                value(FloatType::F32, tag("f32")),
                value(FloatType::F64, tag("f64")),
            ))(s)
        }

        fn parse_signedness(s: &str) -> IResult<bool> {
            alt((value(true, tag("s")), value(false, tag("u"))))(s)
        }

        fn parse_int_type(s: &str) -> IResult<IntegerValType> {
            alt((
                value(IntegerValType::I32, tag("i32")),
                value(IntegerValType::I64, tag("i64")),
            ))(s)
        }

        fn parse_un_op(s: &str) -> IResult<FloatUnOp> {
            alt((
                value(FloatUnOp::Abs, tag("abs")),
                value(FloatUnOp::Neg, tag("neg")),
                value(FloatUnOp::Ceil, tag("ceil")),
                value(FloatUnOp::Floor, tag("floor")),
                value(FloatUnOp::Trunc, tag("trunc")),
                value(FloatUnOp::Nearest, tag("nearest")),
                value(FloatUnOp::Sqrt, tag("sqrt")),
            ))(s)
        }

        fn parse_bin_op(s: &str) -> IResult<FloatBinOp> {
            alt((
                value(FloatBinOp::Add, tag("add")),
                value(FloatBinOp::Sub, tag("sub")),
                value(FloatBinOp::Mul, tag("mul")),
                value(FloatBinOp::Div, tag("div")),
                value(FloatBinOp::Min, tag("min")),
                value(FloatBinOp::Max, tag("max")),
                value(FloatBinOp::CopySign, tag("copysign")),
            ))(s)
        }

        fn parse_rel_op(s: &str) -> IResult<FloatRelOp> {
            alt((
                value(FloatRelOp::Eq, tag("eq")),
                value(FloatRelOp::Ne, tag("ne")),
                value(FloatRelOp::Lt, tag("lt")),
                value(FloatRelOp::Gt, tag("gt")),
                value(FloatRelOp::Le, tag("le")),
                value(FloatRelOp::Ge, tag("ge")),
            ))(s)
        }

        let inst = alt((
            map(
                all_consuming(tuple((parse_fp_type, tag("_"), parse_un_op))),
                |(t, _, o)| FloatInstruction::UnOp(t, o),
            ),
            map(
                all_consuming(tuple((parse_fp_type, tag("_"), parse_bin_op))),
                |(t, _, o)| FloatInstruction::BinOp(t, o),
            ),
            map(
                all_consuming(tuple((parse_fp_type, tag("_"), parse_rel_op))),
                |(t, _, o)| FloatInstruction::RelOp(t, o),
            ),
            map(
                all_consuming(tuple((
                    parse_int_type,
                    alt((
                        value(true, tag("_trunc_sat_")),
                        value(false, tag("_trunc_")),
                    )),
                    parse_fp_type,
                    tag("_"),
                    parse_signedness,
                ))),
                |(i, sat, f, _, s)| FloatInstruction::TruncIntOp(i, f, sat, s),
            ),
            map(
                all_consuming(tuple((
                    parse_fp_type,
                    tag("_convert_"),
                    parse_int_type,
                    tag("_"),
                    parse_signedness,
                ))),
                |(f, _, i, _, s)| FloatInstruction::ConvertIntOp(f, i, s),
            ),
            value(
                FloatInstruction::F32DemoteF64,
                all_consuming(tag("f32_demote_f64")),
            ),
            value(
                FloatInstruction::F64PromoteF32,
                all_consuming(tag("f64_promote_f32")),
            ),
        ));

        let res = preceded(tag("wavm__"), inst)(s);

        res.map(|(_, i)| i).map_err(|e| e.to_string())
    }
}

pub fn op_as_const(op: Operator) -> Result<LirValue> {
    match op {
        Operator::I32Const { value } => Ok(LirValue::I32(value as u32)),
        Operator::I64Const { value } => Ok(LirValue::I64(value as u64)),
        Operator::F32Const { value } => Ok(LirValue::F32(f32::from_bits(value.bits()))),
        Operator::F64Const { value } => Ok(LirValue::F64(f64::from_bits(value.bits()))),
        _ => bail!("Opcode is not a constant"),
    }
}

#[derive(Clone, Debug, Default)]
pub struct Code<'a> {
    pub locals: Vec<Local>,
    pub expr: Vec<Operator<'a>>,
}

#[derive(Clone, Debug)]
pub struct Local {
    pub index: u32,
    pub value: ArbValueType,
}

#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct NameCustomSection {
    pub module: String,
    pub functions: HashMap<u32, String>,
}

#[derive(Clone, Default)]
pub struct WasmBinary<'a> {
    pub types: Vec<FunctionType>,
    pub imports: Vec<Import<'a>>,
    pub functions: Vec<u32>,
    pub tables: Vec<TableType>,
    pub memories: Vec<MemoryType>,
    pub globals: Vec<Global<'a>>,
    pub exports: Vec<Export<'a>>,
    pub start: Option<u32>,
    pub elements: Vec<Element<'a>>,
    pub codes: Vec<Code<'a>>,
    pub datas: Vec<Data<'a>>,
    pub names: NameCustomSection,
}

pub fn parse(input: &[u8]) -> eyre::Result<WasmBinary<'_>> {
    let features = wasmparser::WasmFeatures {
        mutable_global: true,
        saturating_float_to_int: true,
        sign_extension: true,
        reference_types: false,
        multi_value: true,
        bulk_memory: true, //we only support the MemoryCopy and MemoryFill opcodes
        simd: false,
        relaxed_simd: false,
        threads: false,
        tail_call: false,
        deterministic_only: false,
        multi_memory: false,
        exceptions: false,
        memory64: false,
        extended_const: false,
        component_model: false,
    };
    wasmparser::Validator::new_with_features(features).validate_all(input)?;

    let sections: Vec<_> = Parser::new(0).parse_all(input).collect::<Result<_, _>>()?;

    let mut binary = WasmBinary::default();

    for mut section in sections.into_iter() {
        use Payload::*;

        macro_rules! process {
            ($dest:expr, $source:expr) => {{
                for _ in 0..$source.get_count() {
                    let item = $source.read()?;
                    $dest.push(item.into())
                }
            }};
        }

        match &mut section {
            TypeSection(type_section) => {
                for _ in 0..type_section.get_count() {
                    let TypeDef::Func(ty) = type_section.read()?;
                    binary.types.push(ty.try_into()?);
                }
            }
            CodeSectionEntry(codes) => {
                let mut code = Code::default();
                let mut locals = codes.get_locals_reader()?;
                let mut ops = codes.get_operators_reader()?;

                let mut index = 0;

                for _ in 0..locals.get_count() {
                    let (count, value) = locals.read()?;
                    for _ in 0..count {
                        code.locals.push(Local {
                            index,
                            value: value.try_into()?,
                        });
                        index += 1;
                    }
                }
                while !ops.eof() {
                    code.expr.push(ops.read()?);
                }

                binary.codes.push(code);
            }
            ImportSection(imports) => process!(binary.imports, imports),
            FunctionSection(functions) => process!(binary.functions, functions),
            TableSection(tables) => process!(binary.tables, tables),
            MemorySection(memories) => process!(binary.memories, memories),
            GlobalSection(globals) => process!(binary.globals, globals),
            ExportSection(exports) => process!(binary.exports, exports),
            StartSection { func, .. } => binary.start = Some(*func),
            ElementSection(elements) => process!(binary.elements, elements),
            DataSection(datas) => process!(binary.datas, datas),
            CodeSectionStart { .. } => {}
            CustomSection {
                name,
                data_offset,
                data,
                ..
            } => {
                if *name != "name" {
                    continue;
                }

                let mut name_reader = NameSectionReader::new(data, *data_offset)?;

                while !name_reader.eof() {
                    match name_reader.read()? {
                        Name::Module(name) => binary.names.module = name.get_name()?.to_owned(),
                        Name::Function(namemap) => {
                            let mut map_reader = namemap.get_map()?;
                            for _ in 0..map_reader.get_count() {
                                let Naming { index, name } = map_reader.read()?;
                                binary.names.functions.insert(index, name.to_owned());
                            }
                        }
                        _ => {}
                    }
                }
            }
            Version { num, .. } => ensure!(*num == 1, "wasm format version not supported {}", num),
            UnknownSection { id, .. } => bail!("unsupported unknown section type {}", id),
            End(_offset) => {}
            x => bail!("unsupported section type {:?}", x),
        }
    }

    Ok(binary)
}
