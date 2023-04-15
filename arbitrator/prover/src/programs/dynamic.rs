// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use super::{
    config::CompilePricingParams,
    meter::{STYLUS_INK_LEFT, STYLUS_INK_STATUS},
    FuncMiddleware, Middleware, ModuleMod,
};
use eyre::{bail, Result};
use parking_lot::Mutex;
use wasmer_types::{GlobalIndex, GlobalInit, LocalFunctionIndex, Type};
use wasmparser::{Operator, Type as WpType, TypeOrFuncType};

#[derive(Debug)]
pub struct DynamicMeter {
    memory_fill: u64,
    memory_copy: u64,
    globals: Mutex<Option<[GlobalIndex; 3]>>,
}

impl DynamicMeter {
    const SCRATCH_GLOBAL: &str = "stylus_dynamic_scratch_global";

    pub fn new(pricing: &CompilePricingParams) -> Self {
        Self {
            memory_fill: pricing.memory_fill_ink,
            memory_copy: pricing.memory_copy_ink,
            globals: Mutex::new(None),
        }
    }
}

impl<M: ModuleMod> Middleware<M> for DynamicMeter {
    type FM<'a> = FuncDynamicMeter;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let ink = module.get_global(STYLUS_INK_LEFT)?;
        let status = module.get_global(STYLUS_INK_STATUS)?;
        let scratch = Self::SCRATCH_GLOBAL;
        let scratch = module.add_global(scratch, Type::I32, GlobalInit::I32Const(0))?;
        *self.globals.lock() = Some([ink, status, scratch]);
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        let globals = self.globals.lock().expect("missing globals");
        Ok(FuncDynamicMeter::new(
            self.memory_fill,
            self.memory_copy,
            globals,
        ))
    }

    fn name(&self) -> &'static str {
        "dynamic ink meter"
    }
}

#[derive(Debug)]
pub struct FuncDynamicMeter {
    memory_fill: u64,
    memory_copy: u64,
    globals: [GlobalIndex; 3],
}

impl FuncDynamicMeter {
    fn new(memory_fill: u64, memory_copy: u64, globals: [GlobalIndex; 3]) -> Self {
        Self {
            memory_fill,
            memory_copy,
            globals,
        }
    }
}

impl<'a> FuncMiddleware<'a> for FuncDynamicMeter {
    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>,
    {
        use Operator::*;
        macro_rules! dot {
            ($first:ident $(,$opcode:ident)* $(,)?) => {
                $first { .. } $(| $opcode { .. })*
            };
        }
        macro_rules! get {
            ($global:expr) => {
                GlobalGet {
                    global_index: $global,
                }
            };
        }
        macro_rules! set {
            ($global:expr) => {
                GlobalSet {
                    global_index: $global,
                }
            };
        }

        let [ink, status, scratch] = self.globals.map(|x| x.as_u32());
        let if_ty = TypeOrFuncType::Type(WpType::EmptyBlockType);

        #[rustfmt::skip]
        let linear = |coefficient| {
            [
                // [user] → move user value to scratch
                set!(scratch),
                get!(ink),
                get!(ink),
                get!(scratch),

                // [ink ink size] → cost = size * coefficient (can't overflow)
                I64ExtendI32U,
                I64Const { value: coefficient },
                I64Mul,

                // [ink ink cost] → ink -= cost
                I64Sub,
                set!(ink),
                get!(ink),

                // [old_ink, new_ink] → (old_ink < new_ink) (overflow detected)
                I64LtU,
                If { ty: if_ty },
                I32Const { value: 1 },
                set!(status),
                Unreachable,
                End,

                // [] → resume since user paid for ink
                get!(scratch),
            ]
        };

        match op {
            dot!(MemoryFill) => out.extend(linear(self.memory_fill as i64)),
            dot!(MemoryCopy) => out.extend(linear(self.memory_copy as i64)),
            dot!(
                MemoryInit, DataDrop, ElemDrop, TableInit, TableCopy, TableFill, TableGet,
                TableSet, TableGrow, TableSize
            ) => {
                bail!("opcode not supported")
            }
            _ => {}
        }
        out.extend([op]);
        Ok(())
    }

    fn name(&self) -> &'static str {
        "dynamic ink meter"
    }
}
