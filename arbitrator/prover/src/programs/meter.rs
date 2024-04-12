// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    programs::{
        config::{CompilePricingParams, PricingParams, SigMap},
        FuncMiddleware, Middleware, ModuleMod,
    },
    value::FunctionType,
    Machine,
};
use arbutil::{evm, operator::OperatorInfo};
use derivative::Derivative;
use eyre::Result;
use fnv::FnvHashMap as HashMap;
use parking_lot::RwLock;
use std::{
    fmt::{Debug, Display},
    sync::Arc,
};
use wasmer_types::{GlobalIndex, GlobalInit, LocalFunctionIndex, SignatureIndex, Type};
use wasmparser::{BlockType, Operator};

use super::config::OpCosts;

pub const STYLUS_INK_LEFT: &str = "stylus_ink_left";
pub const STYLUS_INK_STATUS: &str = "stylus_ink_status";

pub trait OpcodePricer: Fn(&Operator, &SigMap) -> u64 + Send + Sync + Clone {}

impl<T> OpcodePricer for T where T: Fn(&Operator, &SigMap) -> u64 + Send + Sync + Clone {}

#[derive(Derivative)]
#[derivative(Debug)]
pub struct Meter<F: OpcodePricer> {
    /// Associates opcodes to their ink costs.
    #[derivative(Debug = "ignore")]
    costs: F,
    /// Cost of checking the amount of ink left.
    header_cost: u64,
    /// Ink and ink status globals.
    globals: RwLock<Option<[GlobalIndex; 2]>>,
    /// The types of the module being instrumented
    sigs: RwLock<Option<Arc<SigMap>>>,
}

impl Meter<OpCosts> {
    pub fn new(pricing: &CompilePricingParams) -> Meter<OpCosts> {
        Self {
            costs: pricing.costs,
            header_cost: pricing.ink_header_cost,
            globals: RwLock::default(),
            sigs: RwLock::default(),
        }
    }
}

impl<F: OpcodePricer> Meter<F> {
    pub fn globals(&self) -> [GlobalIndex; 2] {
        self.globals.read().expect("missing globals")
    }
}

impl<M, F> Middleware<M> for Meter<F>
where
    M: ModuleMod,
    F: OpcodePricer + 'static,
{
    type FM<'a> = FuncMeter<'a, F>;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let start_status = GlobalInit::I32Const(0);
        let ink = module.add_global(STYLUS_INK_LEFT, Type::I64, GlobalInit::I64Const(0))?;
        let status = module.add_global(STYLUS_INK_STATUS, Type::I32, start_status)?;
        *self.globals.write() = Some([ink, status]);
        *self.sigs.write() = Some(Arc::new(module.all_signatures()?));
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        let [ink, status] = self.globals();
        let sigs = self.sigs.read();
        let sigs = sigs.as_ref().expect("no types");
        Ok(FuncMeter::new(
            ink,
            status,
            self.costs.clone(),
            self.header_cost,
            sigs.clone(),
        ))
    }

    fn name(&self) -> &'static str {
        "ink meter"
    }
}

#[derive(Derivative)]
#[derivative(Debug)]
pub struct FuncMeter<'a, F: OpcodePricer> {
    /// Represents the amount of ink left for consumption.
    ink_global: GlobalIndex,
    /// Represents whether the machine is out of ink.
    status_global: GlobalIndex,
    /// Instructions of the current basic block.
    block: Vec<Operator<'a>>,
    /// The accumulated cost of the current basic block.
    block_cost: u64,
    /// Cost of checking the amount of ink left.
    header_cost: u64,
    /// Associates opcodes to their ink costs.
    #[derivative(Debug = "ignore")]
    costs: F,
    /// The types of the module being instrumented.
    sigs: Arc<SigMap>,
}

impl<'a, F: OpcodePricer> FuncMeter<'a, F> {
    fn new(
        ink_global: GlobalIndex,
        status_global: GlobalIndex,
        costs: F,
        header_cost: u64,
        sigs: Arc<SigMap>,
    ) -> Self {
        Self {
            ink_global,
            status_global,
            block: vec![],
            block_cost: 0,
            header_cost,
            costs,
            sigs,
        }
    }
}

impl<'a, F: OpcodePricer> FuncMiddleware<'a> for FuncMeter<'a, F> {
    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>,
    {
        use Operator::*;

        let end = op.ends_basic_block();

        let op_cost = (self.costs)(&op, &self.sigs);
        let mut cost = self.block_cost.saturating_add(op_cost);
        self.block_cost = cost;
        self.block.push(op);

        if end {
            let ink = self.ink_global.as_u32();
            let status = self.status_global.as_u32();
            let blockty = BlockType::Empty;

            // include the cost of executing the header
            cost = cost.saturating_add(self.header_cost);

            out.extend([
                // if ink < cost => panic with status = 1
                GlobalGet { global_index: ink },
                I64Const { value: cost as i64 },
                I64LtU,
                If { blockty },
                I32Const { value: 1 },
                GlobalSet {
                    global_index: status,
                },
                Unreachable,
                End,
                // ink -= cost
                GlobalGet { global_index: ink },
                I64Const { value: cost as i64 },
                I64Sub,
                GlobalSet { global_index: ink },
            ]);
            out.extend(self.block.drain(..));
            self.block_cost = 0;
        }
        Ok(())
    }

    fn name(&self) -> &'static str {
        "ink meter"
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum MachineMeter {
    Ready(u64),
    Exhausted,
}

impl MachineMeter {
    pub fn ink(self) -> u64 {
        match self {
            Self::Ready(ink) => ink,
            Self::Exhausted => 0,
        }
    }

    pub fn status(self) -> u32 {
        match self {
            Self::Ready(_) => 0,
            Self::Exhausted => 1,
        }
    }
}

/// We don't implement `From` since it's unclear what 0 would map to
#[allow(clippy::from_over_into)]
impl Into<u64> for MachineMeter {
    fn into(self) -> u64 {
        self.ink()
    }
}

impl Display for MachineMeter {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Ready(ink) => write!(f, "{ink} ink"),
            Self::Exhausted => write!(f, "exhausted"),
        }
    }
}

#[derive(Debug)]
pub struct OutOfInkError;

impl std::error::Error for OutOfInkError {}

impl Display for OutOfInkError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "out of ink")
    }
}

/// Note: implementers may panic if uninstrumented
pub trait MeteredMachine {
    fn ink_left(&self) -> MachineMeter;
    fn set_meter(&mut self, meter: MachineMeter);

    fn set_ink(&mut self, ink: u64) {
        self.set_meter(MachineMeter::Ready(ink));
    }

    fn out_of_ink<T>(&mut self) -> Result<T, OutOfInkError> {
        self.set_meter(MachineMeter::Exhausted);
        Err(OutOfInkError)
    }

    fn ink_ready(&mut self) -> Result<u64, OutOfInkError> {
        let MachineMeter::Ready(ink_left) = self.ink_left() else {
            return self.out_of_ink();
        };
        Ok(ink_left)
    }

    fn buy_ink(&mut self, ink: u64) -> Result<(), OutOfInkError> {
        let ink_left = self.ink_ready()?;
        if ink_left < ink {
            return self.out_of_ink();
        }
        self.set_ink(ink_left - ink);
        Ok(())
    }

    /// Checks if the user has enough ink, but doesn't burn any
    fn require_ink(&mut self, ink: u64) -> Result<(), OutOfInkError> {
        let ink_left = self.ink_ready()?;
        if ink_left < ink {
            return self.out_of_ink();
        }
        Ok(())
    }

    /// Pays for a write into the client.
    fn pay_for_write(&mut self, bytes: u32) -> Result<(), OutOfInkError> {
        self.buy_ink(sat_add_mul(5040, 30, bytes.saturating_sub(32)))
    }

    /// Pays for a read into the host.
    fn pay_for_read(&mut self, bytes: u32) -> Result<(), OutOfInkError> {
        self.buy_ink(sat_add_mul(16381, 55, bytes.saturating_sub(32)))
    }

    /// Pays for both I/O and keccak.
    fn pay_for_keccak(&mut self, bytes: u32) -> Result<(), OutOfInkError> {
        let words = evm::evm_words(bytes).saturating_sub(2);
        self.buy_ink(sat_add_mul(121800, 21000, words))
    }

    /// Pays for copying bytes from geth.
    fn pay_for_geth_bytes(&mut self, bytes: u32) -> Result<(), OutOfInkError> {
        self.pay_for_read(bytes) // TODO: determine value
    }
}

pub trait GasMeteredMachine: MeteredMachine {
    fn pricing(&self) -> PricingParams;

    fn gas_left(&self) -> Result<u64, OutOfInkError> {
        let pricing = self.pricing();
        match self.ink_left() {
            MachineMeter::Ready(ink) => Ok(pricing.ink_to_gas(ink)),
            MachineMeter::Exhausted => Err(OutOfInkError),
        }
    }

    fn buy_gas(&mut self, gas: u64) -> Result<(), OutOfInkError> {
        let pricing = self.pricing();
        self.buy_ink(pricing.gas_to_ink(gas))
    }

    /// Checks if the user has enough gas, but doesn't burn any
    fn require_gas(&mut self, gas: u64) -> Result<(), OutOfInkError> {
        let pricing = self.pricing();
        self.require_ink(pricing.gas_to_ink(gas))
    }

    fn pay_for_evm_log(&mut self, topics: u32, data_len: u32) -> Result<(), OutOfInkError> {
        let cost = (1 + topics as u64) * evm::LOG_TOPIC_GAS;
        let cost = cost.saturating_add(data_len as u64 * evm::LOG_DATA_GAS);
        self.buy_gas(cost)
    }
}

fn sat_add_mul(base: u64, per: u64, count: u32) -> u64 {
    base.saturating_add(per.saturating_mul(count.into()))
}

impl MeteredMachine for Machine {
    fn ink_left(&self) -> MachineMeter {
        macro_rules! convert {
            ($global:expr) => {{
                $global.unwrap().try_into().expect("type mismatch")
            }};
        }

        let ink = || convert!(self.get_global(STYLUS_INK_LEFT));
        let status: u32 = convert!(self.get_global(STYLUS_INK_STATUS));

        match status {
            0 => MachineMeter::Ready(ink()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        let ink = meter.ink();
        let status = meter.status();
        self.set_global(STYLUS_INK_LEFT, ink.into()).unwrap();
        self.set_global(STYLUS_INK_STATUS, status.into()).unwrap();
    }
}

pub fn pricing_v1(op: &Operator, tys: &HashMap<SignatureIndex, FunctionType>) -> u64 {
    use Operator::*;

    macro_rules! op {
        ($first:ident $(,$opcode:ident)*) => {
            $first $(| $opcode)*
        };
    }
    macro_rules! dot {
        ($first:ident $(,$opcode:ident)*) => {
            $first { .. } $(| $opcode { .. })*
        };
    }

    #[rustfmt::skip]
    let ink = match op {
        op!(Unreachable, Return) => 1,
        op!(Nop) | dot!(I32Const, I64Const) => 1,

        op!(Drop) => 9, // could be 1, but using a higher number helps limit the number of ops in BOLD

        dot!(Block, Loop) | op!(Else, End) => 1,
        dot!(Br, BrIf, If) => 765,
        dot!(Select) => 1250, // TODO: improve wasmer codegen
        dot!(Call) => 3800,
        dot!(LocalGet, LocalTee) => 75,
        dot!(LocalSet) => 210,
        dot!(GlobalGet) => 225,
        dot!(GlobalSet) => 575,
        dot!(I32Load, I32Load8S, I32Load8U, I32Load16S, I32Load16U) => 670,
        dot!(I64Load, I64Load8S, I64Load8U, I64Load16S, I64Load16U, I64Load32S, I64Load32U) => 680,
        dot!(I32Store, I32Store8, I32Store16) => 825,
        dot!(I64Store, I64Store8, I64Store16, I64Store32) => 950,
        dot!(MemorySize) => 3000,
        dot!(MemoryGrow) => 1, // cost handled by memory pricer

        op!(I32Eqz, I32Eq, I32Ne, I32LtS, I32LtU, I32GtS, I32GtU, I32LeS, I32LeU, I32GeS, I32GeU) => 170,
        op!(I64Eqz, I64Eq, I64Ne, I64LtS, I64LtU, I64GtS, I64GtU, I64LeS, I64LeU, I64GeS, I64GeU) => 225,

        op!(I32Clz, I32Ctz) => 210,
        op!(I32Add, I32Sub) => 70,
        op!(I32Mul) => 160,
        op!(I32DivS, I32DivU, I32RemS, I32RemU) => 1120,
        op!(I32And, I32Or, I32Xor, I32Shl, I32ShrS, I32ShrU, I32Rotl, I32Rotr) => 70,

        op!(I64Clz, I64Ctz) => 210,
        op!(I64Add, I64Sub) => 100,
        op!(I64Mul) => 160,
        op!(I64DivS, I64DivU, I64RemS, I64RemU) => 1270,
        op!(I64And, I64Or, I64Xor, I64Shl, I64ShrS, I64ShrU, I64Rotl, I64Rotr) => 100,

        op!(I32Popcnt) => 2650, // slow on ARM, fast on x86
        op!(I64Popcnt) => 6000, // slow on ARM, fast on x86

        op!(I32WrapI64, I64ExtendI32S, I64ExtendI32U) => 100,
        op!(I32Extend8S, I32Extend16S, I64Extend8S, I64Extend16S, I64Extend32S) => 100,
        dot!(MemoryCopy) => 950,
        dot!(MemoryFill) => 950,

        BrTable { targets } => {
            2400 + 325 * targets.len() as u64
        },
        CallIndirect { type_index, .. } => {
            let ty = tys.get(&SignatureIndex::from_u32(*type_index)).expect("no type");
            13610 + 650 * ty.inputs.len() as u64
        },

        // we don't support the following, so return u64::MAX
        dot!(
            Try, Catch, CatchAll, Delegate, Throw, Rethrow,

            RefNull, RefIsNull, RefFunc,

            TypedSelect, ReturnCall, ReturnCallIndirect,

            MemoryInit, DataDrop, TableInit, ElemDrop,
            TableCopy, TableFill, TableGet, TableSet, TableGrow, TableSize,

            F32Load, F64Load, F32Store, F64Store, F32Const, F64Const,
            F32Eq, F32Ne, F32Lt, F32Gt, F32Le, F32Ge,
            F64Eq, F64Ne, F64Lt, F64Gt, F64Le, F64Ge,
            F32Abs, F32Neg, F32Ceil, F32Floor, F32Trunc, F32Nearest, F32Sqrt, F32Add, F32Sub, F32Mul,
            F32Div, F32Min, F32Max, F32Copysign, F64Abs, F64Neg, F64Ceil, F64Floor, F64Trunc,
            F64Nearest, F64Sqrt, F64Add, F64Sub, F64Mul, F64Div, F64Min, F64Max, F64Copysign,
            I32TruncF32S, I32TruncF32U, I32TruncF64S, I32TruncF64U,
            I64TruncF32S, I64TruncF32U, I64TruncF64S, I64TruncF64U,
            F32ConvertI32S, F32ConvertI32U, F32ConvertI64S, F32ConvertI64U, F32DemoteF64,
            F64ConvertI32S, F64ConvertI32U, F64ConvertI64S, F64ConvertI64U, F64PromoteF32,
            I32ReinterpretF32, I64ReinterpretF64, F32ReinterpretI32, F64ReinterpretI64,
            I32TruncSatF32S, I32TruncSatF32U, I32TruncSatF64S, I32TruncSatF64U,
            I64TruncSatF32S, I64TruncSatF32U, I64TruncSatF64S, I64TruncSatF64U,

            MemoryAtomicNotify, MemoryAtomicWait32, MemoryAtomicWait64, AtomicFence, I32AtomicLoad,
            I64AtomicLoad, I32AtomicLoad8U, I32AtomicLoad16U, I64AtomicLoad8U, I64AtomicLoad16U,
            I64AtomicLoad32U, I32AtomicStore, I64AtomicStore, I32AtomicStore8, I32AtomicStore16,
            I64AtomicStore8, I64AtomicStore16, I64AtomicStore32, I32AtomicRmwAdd, I64AtomicRmwAdd,
            I32AtomicRmw8AddU, I32AtomicRmw16AddU, I64AtomicRmw8AddU, I64AtomicRmw16AddU, I64AtomicRmw32AddU,
            I32AtomicRmwSub, I64AtomicRmwSub, I32AtomicRmw8SubU, I32AtomicRmw16SubU, I64AtomicRmw8SubU,
            I64AtomicRmw16SubU, I64AtomicRmw32SubU, I32AtomicRmwAnd, I64AtomicRmwAnd, I32AtomicRmw8AndU,
            I32AtomicRmw16AndU, I64AtomicRmw8AndU, I64AtomicRmw16AndU, I64AtomicRmw32AndU, I32AtomicRmwOr,
            I64AtomicRmwOr, I32AtomicRmw8OrU, I32AtomicRmw16OrU, I64AtomicRmw8OrU, I64AtomicRmw16OrU,
            I64AtomicRmw32OrU, I32AtomicRmwXor, I64AtomicRmwXor, I32AtomicRmw8XorU, I32AtomicRmw16XorU,
            I64AtomicRmw8XorU, I64AtomicRmw16XorU, I64AtomicRmw32XorU, I32AtomicRmwXchg, I64AtomicRmwXchg,
            I32AtomicRmw8XchgU, I32AtomicRmw16XchgU, I64AtomicRmw8XchgU, I64AtomicRmw16XchgU,
            I64AtomicRmw32XchgU, I32AtomicRmwCmpxchg, I64AtomicRmwCmpxchg, I32AtomicRmw8CmpxchgU,
            I32AtomicRmw16CmpxchgU, I64AtomicRmw8CmpxchgU, I64AtomicRmw16CmpxchgU, I64AtomicRmw32CmpxchgU,

            V128Load, V128Load8x8S, V128Load8x8U, V128Load16x4S, V128Load16x4U, V128Load32x2S, V128Load32x2U,
            V128Load8Splat, V128Load16Splat, V128Load32Splat, V128Load64Splat, V128Load32Zero, V128Load64Zero,
            V128Store, V128Load8Lane, V128Load16Lane, V128Load32Lane, V128Load64Lane, V128Store8Lane,
            V128Store16Lane, V128Store32Lane, V128Store64Lane, V128Const,
            I8x16Shuffle, I8x16ExtractLaneS, I8x16ExtractLaneU, I8x16ReplaceLane, I16x8ExtractLaneS,
            I16x8ExtractLaneU, I16x8ReplaceLane, I32x4ExtractLane, I32x4ReplaceLane, I64x2ExtractLane,
            I64x2ReplaceLane, F32x4ExtractLane, F32x4ReplaceLane, F64x2ExtractLane, F64x2ReplaceLane,
            I8x16Swizzle, I8x16Splat, I16x8Splat, I32x4Splat, I64x2Splat, F32x4Splat, F64x2Splat, I8x16Eq,
            I8x16Ne, I8x16LtS, I8x16LtU, I8x16GtS, I8x16GtU, I8x16LeS, I8x16LeU, I8x16GeS, I8x16GeU, I16x8Eq,
            I16x8Ne, I16x8LtS, I16x8LtU, I16x8GtS, I16x8GtU, I16x8LeS, I16x8LeU, I16x8GeS, I16x8GeU, I32x4Eq,
            I32x4Ne, I32x4LtS, I32x4LtU, I32x4GtS, I32x4GtU, I32x4LeS, I32x4LeU, I32x4GeS, I32x4GeU, I64x2Eq,
            I64x2Ne, I64x2LtS, I64x2GtS, I64x2LeS, I64x2GeS,
            F32x4Eq, F32x4Ne, F32x4Lt, F32x4Gt, F32x4Le, F32x4Ge,
            F64x2Eq, F64x2Ne, F64x2Lt, F64x2Gt, F64x2Le, F64x2Ge,
            V128Not, V128And, V128AndNot, V128Or, V128Xor, V128Bitselect, V128AnyTrue,
            I8x16Abs, I8x16Neg, I8x16Popcnt, I8x16AllTrue, I8x16Bitmask, I8x16NarrowI16x8S, I8x16NarrowI16x8U,
            I8x16Shl, I8x16ShrS, I8x16ShrU, I8x16Add, I8x16AddSatS, I8x16AddSatU, I8x16Sub, I8x16SubSatS,
            I8x16SubSatU, I8x16MinS, I8x16MinU, I8x16MaxS, I8x16MaxU, I8x16AvgrU,
            I16x8ExtAddPairwiseI8x16S, I16x8ExtAddPairwiseI8x16U, I16x8Abs, I16x8Neg, I16x8Q15MulrSatS,
            I16x8AllTrue, I16x8Bitmask, I16x8NarrowI32x4S, I16x8NarrowI32x4U, I16x8ExtendLowI8x16S,
            I16x8ExtendHighI8x16S, I16x8ExtendLowI8x16U, I16x8ExtendHighI8x16U, I16x8Shl, I16x8ShrS, I16x8ShrU,
            I16x8Add, I16x8AddSatS, I16x8AddSatU, I16x8Sub, I16x8SubSatS, I16x8SubSatU, I16x8Mul, I16x8MinS,
            I16x8MinU, I16x8MaxS, I16x8MaxU, I16x8AvgrU, I16x8ExtMulLowI8x16S,
            I16x8ExtMulHighI8x16S, I16x8ExtMulLowI8x16U, I16x8ExtMulHighI8x16U, I32x4ExtAddPairwiseI16x8S,
            I32x4ExtAddPairwiseI16x8U, I32x4Abs, I32x4Neg, I32x4AllTrue, I32x4Bitmask, I32x4ExtendLowI16x8S,
            I32x4ExtendHighI16x8S, I32x4ExtendLowI16x8U, I32x4ExtendHighI16x8U, I32x4Shl, I32x4ShrS, I32x4ShrU,
            I32x4Add, I32x4Sub, I32x4Mul, I32x4MinS, I32x4MinU, I32x4MaxS, I32x4MaxU, I32x4DotI16x8S,
            I32x4ExtMulLowI16x8S, I32x4ExtMulHighI16x8S, I32x4ExtMulLowI16x8U, I32x4ExtMulHighI16x8U, I64x2Abs,
            I64x2Neg, I64x2AllTrue, I64x2Bitmask, I64x2ExtendLowI32x4S, I64x2ExtendHighI32x4S,
            I64x2ExtendLowI32x4U, I64x2ExtendHighI32x4U, I64x2Shl, I64x2ShrS, I64x2ShrU, I64x2Add, I64x2Sub,
            I64x2Mul, I64x2ExtMulLowI32x4S, I64x2ExtMulHighI32x4S, I64x2ExtMulLowI32x4U, I64x2ExtMulHighI32x4U,
            F32x4Ceil, F32x4Floor, F32x4Trunc, F32x4Nearest, F32x4Abs, F32x4Neg, F32x4Sqrt, F32x4Add, F32x4Sub,
            F32x4Mul, F32x4Div, F32x4Min, F32x4Max, F32x4PMin, F32x4PMax, F64x2Ceil, F64x2Floor, F64x2Trunc,
            F64x2Nearest, F64x2Abs, F64x2Neg, F64x2Sqrt, F64x2Add, F64x2Sub, F64x2Mul, F64x2Div, F64x2Min,
            F64x2Max, F64x2PMin, F64x2PMax, I32x4TruncSatF32x4S, I32x4TruncSatF32x4U, F32x4ConvertI32x4S,
            F32x4ConvertI32x4U, I32x4TruncSatF64x2SZero, I32x4TruncSatF64x2UZero, F64x2ConvertLowI32x4S,
            F64x2ConvertLowI32x4U, F32x4DemoteF64x2Zero, F64x2PromoteLowF32x4, I8x16RelaxedSwizzle,
            I32x4RelaxedTruncSatF32x4S, I32x4RelaxedTruncSatF32x4U, I32x4RelaxedTruncSatF64x2SZero,
            I32x4RelaxedTruncSatF64x2UZero, F32x4RelaxedFma, F32x4RelaxedFnma, F64x2RelaxedFma,
            F64x2RelaxedFnma, I8x16RelaxedLaneselect, I16x8RelaxedLaneselect, I32x4RelaxedLaneselect,
            I64x2RelaxedLaneselect, F32x4RelaxedMin, F32x4RelaxedMax, F64x2RelaxedMin, F64x2RelaxedMax,
            I16x8RelaxedQ15mulrS, I16x8DotI8x16I7x16S, I32x4DotI8x16I7x16AddS, F32x4RelaxedDotBf16x8AddF32x4
        ) => u64::MAX,
    };
    ink
}
