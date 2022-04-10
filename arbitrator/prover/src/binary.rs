// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    value::{ArbValueType, FunctionType, IntegerValType, Value as LirValue},
    wavm::{IBinOpType, IRelOpType, IUnOpType, Opcode},
};
use fnv::FnvHashMap as HashMap;
use nom::{
    branch::alt,
    bytes::complete::tag,
    combinator::{all_consuming, map, value},
    sequence::{preceded, tuple},
};
use std::{hash::Hash, str::FromStr};
use wasmparser::{
    ExternalKind, FuncType, MemoryType, Parser, Payload, Range, TableType, Type, TypeDef, TypeRef,
};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum BlockType {
    Empty,
    ArbValueType(ArbValueType),
    TypeIndex(u32),
}

#[derive(Clone, Copy, Debug, PartialEq)]
pub struct MemoryArg {
    pub alignment: u32,
    pub offset: u32,
}

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
    TruncIntOp(IntegerValType, FloatType, bool),
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
            FloatInstruction::TruncIntOp(i, f, _) => {
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
                    tag("_trunc_"),
                    parse_fp_type,
                    tag("_"),
                    parse_signedness,
                ))),
                |(i, _, f, _, s)| FloatInstruction::TruncIntOp(i, f, s),
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
    FloatingPointOp(FloatInstruction),
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
            _ => None,
        }
    }
}

#[derive(Clone, Debug)]
pub struct Code {
    pub locals: Vec<ArbValueType>,
    pub expr: Vec<HirInstruction>,
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

#[derive(Clone, Debug, Default)]
pub struct WasmBinary {
    pub unknown_custom_sections: Vec<(String, Vec<u8>)>,
    pub types: Vec<FunctionType>,
    pub imports: Vec<Import>, // fix compare to element
    pub functions: Vec<u32>,
    pub tables: Vec<TableType>, // 
    pub memories: Vec<MemoryType>, // check initial
    pub globals: Vec<Global>,      // finish init
    pub exports: Vec<Export>,      //
    pub start: Option<u32>,        //
    pub elements: Vec<Element>,    // finish init
    pub code: Vec<Code>,
    pub datas: Vec<Data>,
    pub names: NameCustomSection,
}

pub fn parse(input: &[u8]) -> eyre::Result<WasmBinary> {
    wasmparser::validate(&input)?;

    let sections: Vec<_> = Parser::new(0)
        .parse_all(input)
        .into_iter()
        .collect::<Result<_, _>>()?;

    let mut binary = WasmBinary::default();

    for (index, mut section) in sections.into_iter().enumerate() {
        use Payload::*;

        println!("{} {:?}", index, &section);

        macro_rules! process {
            ($dest:expr, $source:expr) => {{
                for _ in 0..$source.get_count() {
                    let item = $source.read()?;
                    $dest.push(item.into())
                }
            }};
        }

        match &mut section {
            Version {
                num,
                encoding,
                range,
            } => {}
            Payload::TypeSection(type_section) => {
                /*let TypeDef::Func(ty) = type_section.read()?;
                let same = FunctionType::new(ty.params.to_owned(), ty.returns.clone());
                binary.types.push(ty);*/
            }
            Payload::ImportSection(imports) => process!(binary.imports, imports),
            FunctionSection(functions) => {}
            TableSection(tables) => process!(binary.tables, tables),
            MemorySection(memories) => process!(binary.memories, memories),
            GlobalSection(globals) => process!(binary.globals, globals),
            ExportSection(exports) => process!(binary.exports, exports),
            StartSection { func, range: _ } => binary.start = Some(*func),
            ElementSection(elements) => process!(binary.elements, elements),
            CodeSectionStart { count, range, size } => {}
            CodeSectionEntry(codes) => {}
            DataSection(datas) => process!(binary.datas, datas),
            AliasSection(names) => {}
            End(offset) => {}
            x => eyre::bail!("unsupported section type {:?}", x),
        }
    }
    panic!();
}

#[derive(Clone, Debug)]
pub struct Import {
    pub module: String,
    pub name: String,
    pub ty: TypeRef,
}

impl From<wasmparser::Import<'_>> for Import {
    fn from(import: wasmparser::Import<'_>) -> Self {
        Self {
            module: import.module.to_owned(),
            name: import.name.to_owned(),
            ty: import.ty,
        }
    }
}

#[derive(Clone, Debug)]
pub struct Global {
    pub ty: Type,
    pub mutable: bool,
    pub initializer: Vec<HirInstruction>,
}

impl From<wasmparser::Global<'_>> for Global {
    fn from(global: wasmparser::Global<'_>) -> Self {
        Self {
            ty: global.ty.content_type,
            mutable: global.ty.mutable,
            initializer: vec![], // TODO: global.init_expr
        }
    }
}

#[derive(Clone, Debug)]
pub struct Export {
    pub name: String,
    pub kind: ExternalKind,
    pub index: u32,
}

impl From<wasmparser::Export<'_>> for Export {
    fn from(export: wasmparser::Export<'_>) -> Self {
        Self {
            name: export.name.to_owned(),
            kind: export.kind,
            index: export.index,
        }
    }
}

#[derive(Clone, Debug)]
pub enum ElementKind {
    Passive,
    Active(u32, Vec<HirInstruction>),
    Declared,
}

impl From<wasmparser::ElementKind<'_>> for ElementKind {
    fn from(kind: wasmparser::ElementKind<'_>) -> Self {
        use wasmparser::ElementKind::*;
        match kind {
            Passive => Self::Passive,
            Declared => Self::Declared,
            Active {
                table_index,
                init_expr,
            } => Self::Active(
                table_index,
                vec![], // TODO: init_expr
            ),
        }
    }
}

#[derive(Clone, Debug)]
pub struct Element {
    pub kind: ElementKind,
    pub items: Vec<LirValue>,
    pub ty: Type,
    pub range: Range,
}

impl From<wasmparser::Element<'_>> for Element {
    fn from(element: wasmparser::Element<'_>) -> Self {
        Self {
            kind: element.kind.into(),
            items: vec![], // TODO: element.items
            ty: element.ty,
            range: element.range,
        }
    }
}

pub enum DataKind {
    Passive,
    Active {
        memory_index: u32,
        initializer: Vec<HirInstruction>,
    },
}

impl From<wasmparser::DataKind<'_>> for DataKind {
    fn from(kind: wasmparser::DataKind<'_>) -> Self {
        use wasmparser::DataKind::*;
        match kind {
            Passive => Self::Passive,
            Active {
                memory_index,
                init_expr,
            } => Self::Active(
                memory_index,
                vec![], // TODO: init_expr
            ),
        }
    }
}

pub struct Data {
    pub data: Vec<>,
    pub range: Range,
}

impl From<wasmparser::Element<'_>> for Data {
    fn from(data: wasmparser::Data<'_>) -> Self {
        Self {
            kind: element.kind.into(),
            data: 
            range: element.range,
        }
    }
}
