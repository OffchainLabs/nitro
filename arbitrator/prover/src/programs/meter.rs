// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{FuncMiddleware, Middleware, ModuleMod};
use crate::Machine;
use arbutil::operator::OperatorInfo;
use eyre::Result;
use parking_lot::Mutex;
use std::fmt::{Debug, Display};
use wasmer_types::{GlobalIndex, GlobalInit, LocalFunctionIndex, Type};
use wasmparser::{Operator, Type as WpType, TypeOrFuncType};

pub const STYLUS_GAS_LEFT: &str = "stylus_gas_left";
pub const STYLUS_GAS_STATUS: &str = "stylus_gas_status";

pub trait OpcodePricer: Fn(&Operator) -> u64 + Send + Sync + Clone {}

impl<T> OpcodePricer for T where T: Fn(&Operator) -> u64 + Send + Sync + Clone {}

pub struct Meter<F: OpcodePricer> {
    gas_global: Mutex<Option<GlobalIndex>>,
    status_global: Mutex<Option<GlobalIndex>>,
    costs: F,
    start_gas: u64,
}

impl<F: OpcodePricer> Meter<F> {
    pub fn new(costs: F, start_gas: u64) -> Self {
        Self {
            gas_global: Mutex::new(None),
            status_global: Mutex::new(None),
            costs,
            start_gas,
        }
    }

    pub fn globals(&self) -> (GlobalIndex, GlobalIndex) {
        let gas_left = self.gas_global.lock().unwrap();
        let status = self.status_global.lock().unwrap();
        (gas_left, status)
    }
}

impl<M, F> Middleware<M> for Meter<F>
where
    M: ModuleMod,
    F: OpcodePricer + 'static,
{
    type FM<'a> = FuncMeter<'a, F>;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let start_gas = GlobalInit::I64Const(self.start_gas as i64);
        let start_status = GlobalInit::I32Const(0);
        let gas = module.add_global(STYLUS_GAS_LEFT, Type::I64, start_gas)?;
        let status = module.add_global(STYLUS_GAS_STATUS, Type::I32, start_status)?;
        *self.gas_global.lock() = Some(gas);
        *self.status_global.lock() = Some(status);
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        let gas = self.gas_global.lock().expect("no global");
        let status = self.status_global.lock().expect("no global");
        Ok(FuncMeter::new(gas, status, self.costs.clone()))
    }

    fn name(&self) -> &'static str {
        "gas meter"
    }
}

pub struct FuncMeter<'a, F: OpcodePricer> {
    /// Represents the amount of gas left for consumption
    gas_global: GlobalIndex,
    /// Represents whether the machine is out of gas
    status_global: GlobalIndex,
    /// Instructions of the current basic block
    block: Vec<Operator<'a>>,
    /// The accumulated cost of the current basic block
    block_cost: u64,
    /// Associates opcodes to their gas costs
    costs: F,
}

impl<'a, F: OpcodePricer> FuncMeter<'a, F> {
    fn new(gas_global: GlobalIndex, status_global: GlobalIndex, costs: F) -> Self {
        Self {
            gas_global,
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
            let gas = self.gas_global.as_u32();
            let status = self.status_global.as_u32();

            let mut header = vec![
                // if gas < cost => panic with status = 1
                GlobalGet { global_index: gas },
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
                // gas -= cost
                GlobalGet { global_index: gas },
                I64Const { value: cost as i64 },
                I64Sub,
                GlobalSet { global_index: gas },
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
        "gas meter"
    }
}

impl<F: OpcodePricer> Debug for Meter<F> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Meter")
            .field("gas_global", &self.gas_global)
            .field("status_global", &self.status_global)
            .field("costs", &"<function>")
            .field("start_gas", &self.start_gas)
            .finish()
    }
}

impl<F: OpcodePricer> Debug for FuncMeter<'_, F> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("FunctionMeter")
            .field("gas_global", &self.gas_global)
            .field("status_global", &self.status_global)
            .field("block", &self.block)
            .field("block_cost", &self.block_cost)
            .field("costs", &"<function>")
            .finish()
    }
}

#[derive(Debug, PartialEq, Eq)]
pub enum MachineMeter {
    Ready(u64),
    Exhausted,
}

/// We don't implement `From` since it's unclear what 0 would map to
#[allow(clippy::from_over_into)]
impl Into<u64> for MachineMeter {
    fn into(self) -> u64 {
        match self {
            Self::Ready(gas) => gas,
            Self::Exhausted => 0,
        }
    }
}

impl Display for MachineMeter {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Ready(gas) => write!(f, "{gas} gas"),
            Self::Exhausted => write!(f, "exhausted"),
        }
    }
}

/// Note: implementers may panic if uninstrumented
pub trait MeteredMachine {
    fn gas_left(&mut self) -> MachineMeter;
    fn set_gas(&mut self, gas: u64);
}

impl MeteredMachine for Machine {
    fn gas_left(&mut self) -> MachineMeter {
        macro_rules! convert {
            ($global:expr) => {{
                $global.unwrap().try_into().expect("type mismatch")
            }};
        }

        let gas = || convert!(self.get_global(STYLUS_GAS_LEFT));
        let status: u32 = convert!(self.get_global(STYLUS_GAS_STATUS));

        match status {
            0 => MachineMeter::Ready(gas()),
            _ => MachineMeter::Exhausted,
        }
    }

    fn set_gas(&mut self, gas: u64) {
        self.set_global(STYLUS_GAS_LEFT, gas.into()).unwrap();
        self.set_global(STYLUS_GAS_STATUS, 0_u32.into()).unwrap();
    }
}
