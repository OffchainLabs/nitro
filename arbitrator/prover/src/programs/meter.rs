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

pub const STYLUS_INK_LEFT: &str = "stylus_ink_left";
pub const STYLUS_INK_STATUS: &str = "stylus_ink_status";

pub trait OpcodePricer: Fn(&Operator) -> u64 + Send + Sync + Clone {}

impl<T> OpcodePricer for T where T: Fn(&Operator) -> u64 + Send + Sync + Clone {}

pub struct Meter<F: OpcodePricer> {
    costs: F,
    start_ink: u64,
    globals: Mutex<Option<[GlobalIndex; 2]>>,
}

impl<F: OpcodePricer> Meter<F> {
    pub fn new(costs: F, start_ink: u64) -> Self {
        Self {
            costs,
            start_ink,
            globals: Mutex::new(None),
        }
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
        let start_ink = GlobalInit::I64Const(self.start_ink as i64);
        let start_status = GlobalInit::I32Const(0);
        let ink = module.add_global(STYLUS_INK_LEFT, Type::I64, start_ink)?;
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

impl<F: OpcodePricer> Debug for Meter<F> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Meter")
            .field("globals", &self.globals)
            .field("costs", &"<function>")
            .field("start_ink", &self.start_ink)
            .finish()
    }
}

impl<F: OpcodePricer> Debug for FuncMeter<'_, F> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("FunctionMeter")
            .field("ink_global", &self.ink_global)
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
            Self::Ready(ink) => ink,
            Self::Exhausted => 0,
        }
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

/// Note: implementers may panic if uninstrumented
pub trait MeteredMachine {
    fn ink_left(&mut self) -> MachineMeter;
    fn set_ink(&mut self, ink: u64);
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

    fn set_ink(&mut self, ink: u64) {
        self.set_global(STYLUS_INK_LEFT, ink.into()).unwrap();
        self.set_global(STYLUS_INK_STATUS, 0_u32.into()).unwrap();
    }
}
