// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{config::PricingParams, FuncMiddleware, Middleware, ModuleMod};
use crate::Machine;
use arbutil::{evm, operator::OperatorInfo};
use derivative::Derivative;
use eyre::Result;
use parking_lot::Mutex;
use std::fmt::{Debug, Display};
use wasmer_types::{GlobalIndex, GlobalInit, LocalFunctionIndex, Type};
use wasmparser::{Operator, Type as WpType, TypeOrFuncType};

pub const STYLUS_INK_LEFT: &str = "stylus_ink_left";
pub const STYLUS_INK_STATUS: &str = "stylus_ink_status";

pub trait OpcodePricer: Fn(&Operator) -> u64 + Send + Sync + Clone {}

impl<T> OpcodePricer for T where T: Fn(&Operator) -> u64 + Send + Sync + Clone {}

#[derive(Derivative)]
#[derivative(Debug)]
pub struct Meter<F: OpcodePricer> {
    #[derivative(Debug = "ignore")]
    costs: F,
    globals: Mutex<Option<[GlobalIndex; 2]>>,
}

impl<F: OpcodePricer> Meter<F> {
    pub fn new(costs: F) -> Self {
        let globals = Mutex::new(None);
        Self { costs, globals }
    }

    pub fn globals(&self) -> [GlobalIndex; 2] {
        self.globals.lock().expect("missing globals")
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
        *self.globals.lock() = Some([ink, status]);
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        let [ink, status] = self.globals();
        Ok(FuncMeter::new(ink, status, self.costs.clone()))
    }

    fn name(&self) -> &'static str {
        "ink meter"
    }
}

#[derive(Derivative)]
#[derivative(Debug)]
pub struct FuncMeter<'a, F: OpcodePricer> {
    /// Represents the amount of ink left for consumption
    ink_global: GlobalIndex,
    /// Represents whether the machine is out of ink
    status_global: GlobalIndex,
    /// Instructions of the current basic block
    block: Vec<Operator<'a>>,
    /// The accumulated cost of the current basic block
    block_cost: u64,
    /// Associates opcodes to their ink costs
    #[derivative(Debug = "ignore")]
    costs: F,
}

impl<'a, F: OpcodePricer> FuncMeter<'a, F> {
    fn new(ink_global: GlobalIndex, status_global: GlobalIndex, costs: F) -> Self {
        Self {
            ink_global,
            status_global,
            block: vec![],
            block_cost: 0,
            costs,
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

        let mut cost = self.block_cost.saturating_add((self.costs)(&op));
        self.block_cost = cost;
        self.block.push(op);

        if end {
            let ink = self.ink_global.as_u32();
            let status = self.status_global.as_u32();

            let mut header = [
                // if ink < cost => panic with status = 1
                GlobalGet { global_index: ink },
                I64Const { value: cost as i64 },
                I64LtU,
                If {
                    ty: TypeOrFuncType::Type(WpType::EmptyBlockType),
                },
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
            ];

            // include the cost of executing the header
            for op in &header {
                cost = cost.saturating_add((self.costs)(op))
            }
            header[1] = I64Const { value: cost as i64 };
            header[9] = I64Const { value: cost as i64 };

            out.extend(header);
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
    fn ink_left(&mut self) -> MachineMeter;
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
            return self.out_of_ink()
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
}

pub trait GasMeteredMachine: MeteredMachine {
    fn pricing(&mut self) -> PricingParams;

    fn gas_left(&mut self) -> Result<u64, OutOfInkError> {
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

    fn pay_for_evm_copy(&mut self, bytes: u64) -> Result<(), OutOfInkError> {
        let gas = evm::evm_words(bytes).saturating_mul(evm::COPY_WORD_GAS);
        self.buy_gas(gas)
    }

    fn pay_for_evm_keccak(&mut self, bytes: u64) -> Result<(), OutOfInkError> {
        let gas = evm::evm_words(bytes).saturating_mul(evm::KECCAK_WORD_GAS);
        self.buy_gas(gas.saturating_add(evm::KECCAK_256_GAS))
    }

    fn pay_for_evm_log(&mut self, topics: u32, data_len: u32) -> Result<(), OutOfInkError> {
        let cost = (1 + topics as u64) * evm::LOG_TOPIC_GAS;
        let cost = cost.saturating_add(data_len as u64 * evm::LOG_DATA_GAS);
        self.buy_gas(cost)
    }
}

impl MeteredMachine for Machine {
    fn ink_left(&mut self) -> MachineMeter {
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
