// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{config::{CompileMemoryParams, SigMap}, FuncMiddleware, Middleware, ModuleMod};
use crate::{host::InternalFunc, value::FunctionType, Machine};

use arbutil::Color;
use eyre::{bail, Result};
use fnv::FnvHashMap as HashMap;
use parking_lot::RwLock;
use std::sync::Arc;
use wasmer_types::{
    FunctionIndex, GlobalIndex, GlobalInit, LocalFunctionIndex, SignatureIndex, Type,
};
use wasmparser::{Operator, Type as WpType, TypeOrFuncType as BlockType};

pub const STYLUS_STACK_LEFT: &str = "stylus_stack_left";

/// This middleware ensures stack overflows are deterministic across different compilers and targets.
/// The internal notion of "stack space left" that makes this possible is strictly smaller than that of
/// the real stack space consumed on any target platform and is formed by inspecting the contents of each
/// function's frame.
/// Setting a limit smaller than that of any native platform's ensures stack overflows will have the same,
/// logical effect rather than actually exhausting the space provided by the OS.
#[derive(Debug)]
pub struct DepthChecker {
    /// The amount of stack space left
    global: RwLock<Option<GlobalIndex>>,
    /// The maximum size of a stack frame, measured in words
    frame_limit: u32,
    /// The maximum number of overlapping value lifetimes in a frame
    frame_contention: u16,
    /// The function types of the module being instrumented
    funcs: RwLock<Option<Arc<HashMap<FunctionIndex, FunctionType>>>>,
    /// The types of the module being instrumented
    sigs: RwLock<Option<Arc<SigMap>>>,
}

impl DepthChecker {
    pub fn new(params: CompileMemoryParams) -> Self {
        Self {
            global: RwLock::default(),
            frame_limit: params.max_frame_size,
            frame_contention: params.max_frame_contention,
            funcs: RwLock::default(),
            sigs: RwLock::default(),
        }
    }

    pub fn globals(&self) -> GlobalIndex {
        self.global.read().unwrap()
    }
}

impl<M: ModuleMod> Middleware<M> for DepthChecker {
    type FM<'a> = FuncDepthChecker<'a>;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let limit = GlobalInit::I32Const(0);
        let space = module.add_global(STYLUS_STACK_LEFT, Type::I32, limit)?;
        *self.global.write() = Some(space);
        *self.funcs.write() = Some(Arc::new(module.all_functions()?));
        *self.sigs.write() = Some(Arc::new(module.all_signatures()?));
        Ok(())
    }

    fn instrument<'a>(&self, func: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(FuncDepthChecker::new(
            self.global.read().expect("no global"),
            self.funcs.read().clone().expect("no funcs"),
            self.sigs.read().clone().expect("no sigs"),
            self.frame_limit,
            self.frame_contention,
            func,
        ))
    }

    fn name(&self) -> &'static str {
        "depth checker"
    }
}

#[derive(Debug)]
pub struct FuncDepthChecker<'a> {
    /// The amount of stack space left
    global: GlobalIndex,
    /// The function types in this function's module
    funcs: Arc<HashMap<FunctionIndex, FunctionType>>,
    /// All the types in this function's modules
    sigs: Arc<HashMap<SignatureIndex, FunctionType>>,
    /// The number of local variables this func has
    locals: Option<usize>,
    /// The function being instrumented
    func: LocalFunctionIndex,
    /// The maximum size of a stack frame, measured in words
    frame_limit: u32,
    /// The maximum number of overlapping value lifetimes in a frame
    frame_contention: u16,
    /// The number of open scopes
    scopes: isize,
    /// The entirety of the func's original instructions
    code: Vec<Operator<'a>>,
    /// True once it's statically known feed() won't be called again
    done: bool,
}

impl<'a> FuncDepthChecker<'a> {
    fn new(
        global: GlobalIndex,
        funcs: Arc<HashMap<FunctionIndex, FunctionType>>,
        sigs: Arc<HashMap<SignatureIndex, FunctionType>>,
        frame_limit: u32,
        frame_contention: u16,
        func: LocalFunctionIndex,
    ) -> Self {
        Self {
            global,
            funcs,
            sigs,
            locals: None,
            func,
            frame_limit,
            frame_contention,
            scopes: 1, // a function starts with an open scope
            code: vec![],
            done: false,
        }
    }
}

impl<'a> FuncMiddleware<'a> for FuncDepthChecker<'a> {
    fn locals_info(&mut self, locals: &[WpType]) {
        self.locals = Some(locals.len());
    }

    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>,
    {
        use Operator::*;

        // Knowing when the feed ends requires detecting the final instruction, which is
        // guaranteed to be an "End" opcode closing out function's initial opening scope.
        if self.done {
            bail!("finalized too soon");
        }

        let scopes = &mut self.scopes;
        match op {
            Block { .. } | Loop { .. } | If { .. } => *scopes += 1,
            End => *scopes -= 1,
            _ => {}
        }
        if *scopes < 0 {
            bail!("malformed scoping detected");
        }

        let last = *scopes == 0 && matches!(op, End); // true when the feed ends
        self.code.push(op);
        if !last {
            return Ok(());
        }

        // We've reached the final instruction and can instrument the function as follows:
        //   - When entering, check that the stack has sufficient space and deduct the amount used
        //   - When returning, credit back the amount used

        let size = self.worst_case_depth()?;
        let global_index = self.global.as_u32();

        if size > self.frame_limit {
            let limit = self.frame_limit.red();
            bail!("frame too large: {} > {}-word limit", size.red(), limit);
        }

        out.extend([
            // if space <= size => panic with depth = 0
            GlobalGet { global_index },
            I32Const { value: size as i32 },
            I32LeU,
            If {
                ty: BlockType::Type(WpType::EmptyBlockType),
            },
            I32Const { value: 0 },
            GlobalSet { global_index },
            Unreachable,
            End,
            // space -= size
            GlobalGet { global_index },
            I32Const { value: size as i32 },
            I32Sub,
            GlobalSet { global_index },
        ]);

        let reclaim = |out: &mut O| {
            out.extend([
                // space += size
                GlobalGet { global_index },
                I32Const { value: size as i32 },
                I32Add,
                GlobalSet { global_index },
            ])
        };

        // add an extraneous return instruction to the end to match Arbitrator
        let mut code = std::mem::take(&mut self.code);
        let last = code.pop().unwrap();
        code.push(Return);
        code.push(last);

        for op in code {
            let exit = matches!(op, Return);
            if exit {
                reclaim(out);
            }
            out.extend([op]);
        }

        self.done = true;
        Ok(())
    }

    fn name(&self) -> &'static str {
        "depth checker"
    }
}

impl<'a> FuncDepthChecker<'a> {
    fn worst_case_depth(&self) -> Result<u32> {
        use Operator::*;

        let mut worst: u32 = 0;
        let mut stack: u32 = 0;

        macro_rules! push {
            ($count:expr) => {{
                stack += $count;
                worst = worst.max(stack);
            }};
            () => {
                push!(1)
            };
        }
        macro_rules! pop {
            ($count:expr) => {{
                stack = stack.saturating_sub($count);
            }};
            () => {
                pop!(1)
            };
        }
        macro_rules! ins_and_outs {
            ($ty:expr) => {{
                let ins = $ty.inputs.len() as u32;
                let outs = $ty.outputs.len() as u32;
                push!(outs);
                pop!(ins);
            }};
        }
        macro_rules! op {
            ($first:ident $(,$opcode:ident)* $(,)?) => {
                $first $(| $opcode)*
            };
        }
        macro_rules! dot {
            ($first:ident $(,$opcode:ident)* $(,)?) => {
                $first { .. } $(| $opcode { .. })*
            };
        }
        #[rustfmt::skip]
        macro_rules! block_type {
            ($ty:expr) => {{
                match $ty {
                    BlockType::Type(WpType::EmptyBlockType) => {}
                    BlockType::Type(_) => push!(1),
                    BlockType::FuncType(id) => {
                        let index = SignatureIndex::from_u32(*id);
                        let Some(ty) = self.sigs.get(&index) else {
                            bail!("missing type for func {}", id.red())
                        };
                        ins_and_outs!(ty);
                    }
                }
            }};
        }

        let mut scopes = vec![stack];

        for op in &self.code {
            #[rustfmt::skip]
            match op {
                Block { ty } => {
                    block_type!(ty); // we'll say any return slots have been pre-allocated
                    scopes.push(stack);
                }
                Loop { ty } => {
                    block_type!(ty); // return slots
                    scopes.push(stack);
                }
                If { ty } => {
                    pop!();          // pop the conditional
                    block_type!(ty); // return slots
                    scopes.push(stack);
                }
                Else => {
                    stack = match scopes.last() {
                        Some(scope) => *scope,
                        None => bail!("malformed if-else scope"),
                    };
                }
                End => {
                    stack = match scopes.pop() {
                        Some(stack) => stack,
                        None => bail!("malformed scoping detected at end of block"),
                    };
                }

                Call { function_index } => {
                    let index = FunctionIndex::from_u32(*function_index);
                    let Some(ty) = self.funcs.get(&index) else {
                        bail!("missing type for func {}", function_index.red())
                    };
                    ins_and_outs!(ty)
                }
                CallIndirect { index, .. } => {
                    let index = SignatureIndex::from_u32(*index);
                    let Some(ty) = self.sigs.get(&index) else {
                        bail!("missing type for signature {}", index.as_u32().red())
                    };
                    ins_and_outs!(ty);
                    pop!() // the table index
                }

                MemoryFill { .. } => ins_and_outs!(InternalFunc::MemoryFill.ty()),
                MemoryCopy { .. } => ins_and_outs!(InternalFunc::MemoryCopy.ty()),

                op!(
                    Nop, Unreachable,
                    I32Eqz, I64Eqz, I32Clz, I32Ctz, I32Popcnt, I64Clz, I64Ctz, I64Popcnt,
                )
                | dot!(
                    Br, Return,
                    LocalTee, MemoryGrow,
                    I32Load, I64Load, F32Load, F64Load,
                    I32Load8S, I32Load8U, I32Load16S, I32Load16U, I64Load8S, I64Load8U,
                    I64Load16S, I64Load16U, I64Load32S, I64Load32U,
                    I32WrapI64, I64ExtendI32S, I64ExtendI32U,
                    I32Extend8S, I32Extend16S, I64Extend8S, I64Extend16S, I64Extend32S,
                    F32Abs, F32Neg, F32Ceil, F32Floor, F32Trunc, F32Nearest, F32Sqrt,
                    F64Abs, F64Neg, F64Ceil, F64Floor, F64Trunc, F64Nearest, F64Sqrt,
                    I32TruncF32S, I32TruncF32U, I32TruncF64S, I32TruncF64U,
                    I64TruncF32S, I64TruncF32U, I64TruncF64S, I64TruncF64U,
                    F32ConvertI32S, F32ConvertI32U, F32ConvertI64S, F32ConvertI64U, F32DemoteF64,
                    F64ConvertI32S, F64ConvertI32U, F64ConvertI64S, F64ConvertI64U, F64PromoteF32,
                    I32ReinterpretF32, I64ReinterpretF64, F32ReinterpretI32, F64ReinterpretI64,
                    I32TruncSatF32S, I32TruncSatF32U, I32TruncSatF64S, I32TruncSatF64U,
                    I64TruncSatF32S, I64TruncSatF32U, I64TruncSatF64S, I64TruncSatF64U,
                ) => {}

                dot!(
                    LocalGet, GlobalGet, MemorySize,
                    I32Const, I64Const, F32Const, F64Const,
                ) => push!(),

                op!(
                    Drop,
                    I32Eq, I32Ne, I32LtS, I32LtU, I32GtS, I32GtU, I32LeS, I32LeU, I32GeS, I32GeU,
                    I64Eq, I64Ne, I64LtS, I64LtU, I64GtS, I64GtU, I64LeS, I64LeU, I64GeS, I64GeU,
                    F32Eq, F32Ne, F32Lt, F32Gt, F32Le, F32Ge,
                    F64Eq, F64Ne, F64Lt, F64Gt, F64Le, F64Ge,
                    I32Add, I32Sub, I32Mul, I32DivS, I32DivU, I32RemS, I32RemU,
                    I64Add, I64Sub, I64Mul, I64DivS, I64DivU, I64RemS, I64RemU,
                    I32And, I32Or, I32Xor, I32Shl, I32ShrS, I32ShrU, I32Rotl, I32Rotr,
                    I64And, I64Or, I64Xor, I64Shl, I64ShrS, I64ShrU, I64Rotl, I64Rotr,
                    F32Add, F32Sub, F32Mul, F32Div, F32Min, F32Max, F32Copysign,
                    F64Add, F64Sub, F64Mul, F64Div, F64Min, F64Max, F64Copysign,
                )
                | dot!(BrIf, BrTable, LocalSet, GlobalSet) => pop!(),

                dot!(
                    Select,
                    I32Store, I64Store, F32Store, F64Store, I32Store8, I32Store16, I64Store8, I64Store16, I64Store32,
                ) => pop!(2),

                unsupported @ dot!(Try, Catch, Throw, Rethrow) => {
                    bail!("exception-handling extension not supported {:?}", unsupported)
                },

                unsupported @ dot!(ReturnCall, ReturnCallIndirect) => {
                    bail!("tail-call extension not supported {:?}", unsupported)
                }

                unsupported @ (dot!(Delegate) | op!(CatchAll)) => {
                    bail!("exception-handling extension not supported {:?}", unsupported)
                },

                unsupported @ (op!(RefIsNull) | dot!(TypedSelect, RefNull, RefFunc)) => {
                    bail!("reference-types extension not supported {:?}", unsupported)
                },

                unsupported @ (
                    dot!(
                        MemoryInit, DataDrop, TableInit, ElemDrop,
                        TableCopy, TableFill, TableGet, TableSet, TableGrow, TableSize
                    )
                ) => bail!("bulk-memory-operations extension not fully supported {:?}", unsupported),

                unsupported @ (
                    dot!(
                        MemoryAtomicNotify, MemoryAtomicWait32, MemoryAtomicWait64, AtomicFence, I32AtomicLoad,
                        I64AtomicLoad, I32AtomicLoad8U, I32AtomicLoad16U, I64AtomicLoad8U, I64AtomicLoad16U,
                        I64AtomicLoad32U, I32AtomicStore, I64AtomicStore, I32AtomicStore8, I32AtomicStore16,
                        I64AtomicStore8, I64AtomicStore16, I64AtomicStore32, I32AtomicRmwAdd, I64AtomicRmwAdd,
                        I32AtomicRmw8AddU, I32AtomicRmw16AddU,
                        I64AtomicRmw8AddU, I64AtomicRmw16AddU, I64AtomicRmw32AddU,
                        I32AtomicRmwSub, I64AtomicRmwSub, I32AtomicRmw8SubU, I32AtomicRmw16SubU, I64AtomicRmw8SubU,
                        I64AtomicRmw16SubU, I64AtomicRmw32SubU, I32AtomicRmwAnd, I64AtomicRmwAnd, I32AtomicRmw8AndU,
                        I32AtomicRmw16AndU, I64AtomicRmw8AndU, I64AtomicRmw16AndU, I64AtomicRmw32AndU, I32AtomicRmwOr,
                        I64AtomicRmwOr, I32AtomicRmw8OrU, I32AtomicRmw16OrU, I64AtomicRmw8OrU, I64AtomicRmw16OrU,
                        I64AtomicRmw32OrU, I32AtomicRmwXor, I64AtomicRmwXor, I32AtomicRmw8XorU, I32AtomicRmw16XorU,
                        I64AtomicRmw8XorU, I64AtomicRmw16XorU, I64AtomicRmw32XorU, I32AtomicRmwXchg, I64AtomicRmwXchg,
                        I32AtomicRmw8XchgU, I32AtomicRmw16XchgU, I64AtomicRmw8XchgU, I64AtomicRmw16XchgU,
                        I64AtomicRmw32XchgU, I32AtomicRmwCmpxchg, I64AtomicRmwCmpxchg, I32AtomicRmw8CmpxchgU,
                        I32AtomicRmw16CmpxchgU, I64AtomicRmw8CmpxchgU, I64AtomicRmw16CmpxchgU, I64AtomicRmw32CmpxchgU
                    )
                ) => bail!("threads extension not supported {:?}", unsupported),

                unsupported @ (
                    dot!(
                        V128Load, V128Load8x8S, V128Load8x8U, V128Load16x4S, V128Load16x4U, V128Load32x2S,
                        V128Load8Splat, V128Load16Splat, V128Load32Splat, V128Load64Splat, V128Load32Zero,
                        V128Load64Zero, V128Load32x2U,
                        V128Store, V128Load8Lane, V128Load16Lane, V128Load32Lane, V128Load64Lane, V128Store8Lane,
                        V128Store16Lane, V128Store32Lane, V128Store64Lane, V128Const,
                        I8x16Shuffle, I8x16ExtractLaneS, I8x16ExtractLaneU, I8x16ReplaceLane, I16x8ExtractLaneS,
                        I16x8ExtractLaneU, I16x8ReplaceLane, I32x4ExtractLane, I32x4ReplaceLane, I64x2ExtractLane,
                        I64x2ReplaceLane, F32x4ExtractLane, F32x4ReplaceLane, F64x2ExtractLane, F64x2ReplaceLane,
                        I8x16Swizzle, I8x16Splat, I16x8Splat, I32x4Splat, I64x2Splat, F32x4Splat, F64x2Splat, I8x16Eq,
                        I8x16Ne, I8x16LtS, I8x16LtU, I8x16GtS, I8x16GtU, I8x16LeS, I8x16LeU, I8x16GeS, I8x16GeU,
                        I16x8Eq,  I16x8Ne, I16x8LtS, I16x8LtU, I16x8GtS, I16x8GtU, I16x8LeS, I16x8LeU, I16x8GeS,
                        I16x8GeU, I32x4Eq,  I32x4Ne, I32x4LtS, I32x4LtU, I32x4GtS, I32x4GtU, I32x4LeS, I32x4LeU,
                        I32x4GeS, I32x4GeU, I64x2Eq, I64x2Ne, I64x2LtS, I64x2GtS, I64x2LeS, I64x2GeS,
                        F32x4Eq, F32x4Ne, F32x4Lt, F32x4Gt, F32x4Le, F32x4Ge,
                        F64x2Eq, F64x2Ne, F64x2Lt, F64x2Gt, F64x2Le, F64x2Ge,
                        V128Not, V128And, V128AndNot, V128Or, V128Xor, V128Bitselect, V128AnyTrue,
                        I8x16Abs, I8x16Neg, I8x16Popcnt, I8x16AllTrue, I8x16Bitmask,
                        I8x16NarrowI16x8S, I8x16NarrowI16x8U,
                        I8x16Shl, I8x16ShrS, I8x16ShrU, I8x16Add, I8x16AddSatS, I8x16AddSatU, I8x16Sub, I8x16SubSatS,
                        I8x16SubSatU, I8x16MinS, I8x16MinU, I8x16MaxS, I8x16MaxU, I8x16RoundingAverageU,
                        I16x8ExtAddPairwiseI8x16S, I16x8ExtAddPairwiseI8x16U, I16x8Abs, I16x8Neg, I16x8Q15MulrSatS,
                        I16x8AllTrue, I16x8Bitmask, I16x8NarrowI32x4S, I16x8NarrowI32x4U, I16x8ExtendLowI8x16S,
                        I16x8ExtendHighI8x16S, I16x8ExtendLowI8x16U, I16x8ExtendHighI8x16U,
                        I16x8Shl, I16x8ShrS, I16x8ShrU, I16x8Add, I16x8AddSatS, I16x8AddSatU,
                        I16x8Sub, I16x8SubSatS, I16x8SubSatU, I16x8Mul, I16x8MinS, I16x8MinU,
                        I16x8MaxS, I16x8MaxU, I16x8RoundingAverageU, I16x8ExtMulLowI8x16S,
                        I16x8ExtMulHighI8x16S, I16x8ExtMulLowI8x16U, I16x8ExtMulHighI8x16U,
                        I32x4ExtAddPairwiseI16x8U, I32x4Abs, I32x4Neg, I32x4AllTrue, I32x4Bitmask,
                        I32x4ExtAddPairwiseI16x8S, I32x4ExtendLowI16x8S, I32x4ExtendHighI16x8S, I32x4ExtendLowI16x8U,
                        I32x4ExtendHighI16x8U, I32x4Shl, I32x4ShrS, I32x4ShrU, I32x4Add, I32x4Sub, I32x4Mul,
                        I32x4MinS, I32x4MinU, I32x4MaxS, I32x4MaxU, I32x4DotI16x8S,
                        I32x4ExtMulLowI16x8S, I32x4ExtMulHighI16x8S, I32x4ExtMulLowI16x8U, I32x4ExtMulHighI16x8U,
                        I64x2Abs, I64x2Neg, I64x2AllTrue, I64x2Bitmask, I64x2ExtendLowI32x4S, I64x2ExtendHighI32x4S,
                        I64x2ExtendLowI32x4U, I64x2ExtendHighI32x4U, I64x2Shl, I64x2ShrS, I64x2ShrU, I64x2Add,
                        I64x2ExtMulLowI32x4S, I64x2ExtMulHighI32x4S, I64x2Sub, I64x2Mul,
                        I64x2ExtMulLowI32x4U, I64x2ExtMulHighI32x4U, F32x4Ceil, F32x4Floor, F32x4Trunc,
                        F32x4Nearest, F32x4Abs, F32x4Neg, F32x4Sqrt, F32x4Add, F32x4Sub, F32x4Mul, F32x4Div,
                        F32x4Min, F32x4Max, F32x4PMin, F32x4PMax, F64x2Ceil, F64x2Floor, F64x2Trunc,
                        F64x2Nearest, F64x2Abs, F64x2Neg, F64x2Sqrt, F64x2Add, F64x2Sub, F64x2Mul, F64x2Div, F64x2Min,
                        F64x2Max, F64x2PMin, F64x2PMax, I32x4TruncSatF32x4S, I32x4TruncSatF32x4U, F32x4ConvertI32x4S,
                        F32x4ConvertI32x4U, I32x4TruncSatF64x2SZero, I32x4TruncSatF64x2UZero, F64x2ConvertLowI32x4S,
                        F64x2ConvertLowI32x4U, F32x4DemoteF64x2Zero, F64x2PromoteLowF32x4, I8x16RelaxedSwizzle,
                        I32x4RelaxedTruncSatF32x4S, I32x4RelaxedTruncSatF32x4U, I32x4RelaxedTruncSatF64x2SZero,
                        I32x4RelaxedTruncSatF64x2UZero, F32x4Fma, F32x4Fms, F64x2Fma, F64x2Fms, I8x16LaneSelect,
                        I16x8LaneSelect, I32x4LaneSelect, I64x2LaneSelect, F32x4RelaxedMin, F32x4RelaxedMax,
                        F64x2RelaxedMin, F64x2RelaxedMax
                    )
                ) => bail!("SIMD extension not supported {:?}", unsupported),
            };
        }

        if self.locals.is_none() {
            bail!("missing locals info for func {}", self.func.as_u32().red())
        }

        let contention = worst;
        if contention > self.frame_contention.into() {
            bail!(
                "too many values on the stack at once in func {}: {} > {}",
                self.func.as_u32().red(),
                contention.red(),
                self.frame_contention.red()
            );
        }

        let locals = self.locals.unwrap_or_default();
        Ok(worst + locals as u32 + 4)
    }
}

/// Note: implementers may panic if uninstrumented
pub trait DepthCheckedMachine {
    fn stack_left(&mut self) -> u32;
    fn set_stack(&mut self, size: u32);
}

impl DepthCheckedMachine for Machine {
    fn stack_left(&mut self) -> u32 {
        let global = self.get_global(STYLUS_STACK_LEFT).unwrap();
        global.try_into().expect("instrumentation type mismatch")
    }

    fn set_stack(&mut self, size: u32) {
        self.set_global(STYLUS_STACK_LEFT, size.into()).unwrap();
    }
}
