// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::value::{ArbValueType, FunctionType};

use super::{
    config::CompileMemoryParams, dynamic::SCRATCH_GLOBAL, FuncMiddleware, Middleware, ModuleMod,
};
use arbutil::Color;
use eyre::{bail, Result};
use parking_lot::RwLock;
use wasmer_types::{FunctionIndex, GlobalIndex, ImportIndex, LocalFunctionIndex, Pages};
use wasmparser::Operator;

#[derive(Debug)]
pub struct HeapBound {
    /// Upper bounds the amount of heap memory a module may use
    limit: Pages,
    /// Import called when allocating new pages
    pay_for_memory_grow: RwLock<Option<FunctionIndex>>,
    /// Scratch global shared among middlewares
    scratch: RwLock<Option<GlobalIndex>>,
}

impl HeapBound {
    pub fn new(bounds: CompileMemoryParams) -> Self {
        Self {
            limit: bounds.heap_bound,
            pay_for_memory_grow: RwLock::default(),
            scratch: RwLock::default(),
        }
    }
}

impl<M: ModuleMod> Middleware<M> for HeapBound {
    type FM<'a> = FuncHeapBound;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let scratch = module.get_global(SCRATCH_GLOBAL)?;
        *self.scratch.write() = Some(scratch);

        let memory = module.memory_info()?;
        let min = memory.min;
        let max = memory.max;
        let lim = self.limit;

        if min > lim {
            bail!("memory size {} exceeds bound {}", min.0.red(), lim.0.red());
        }
        if max == Some(min) {
            return Ok(());
        }

        let ImportIndex::Function(import) = module.get_import("vm_hooks", "pay_for_memory_grow")?
        else {
            bail!("wrong import kind for {}", "pay_for_memory_grow".red());
        };

        let ty = module.get_function(import)?;
        if ty != FunctionType::new(vec![ArbValueType::I32], vec![]) {
            bail!(
                "wrong type for {}: {}",
                "pay_for_memory_grow".red(),
                ty.red()
            );
        }

        *self.pay_for_memory_grow.write() = Some(import);
        Ok(())
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(FuncHeapBound {
            scratch: self.scratch.read().expect("no scratch global"),
            pay_for_memory_grow: *self.pay_for_memory_grow.read(),
        })
    }

    fn name(&self) -> &'static str {
        "heap bound"
    }
}

#[derive(Debug)]
pub struct FuncHeapBound {
    pay_for_memory_grow: Option<FunctionIndex>,
    scratch: GlobalIndex,
}

impl<'a> FuncMiddleware<'a> for FuncHeapBound {
    fn feed<O>(&mut self, op: Operator<'a>, out: &mut O) -> Result<()>
    where
        O: Extend<Operator<'a>>,
    {
        use Operator::*;

        let Some(pay_for_memory_grow) = self.pay_for_memory_grow else {
            out.extend([op]);
            return Ok(());
        };

        let global_index = self.scratch.as_u32();
        let function_index = pay_for_memory_grow.as_u32();

        if let MemoryGrow { .. } = op {
            out.extend([
                GlobalSet { global_index },
                GlobalGet { global_index },
                GlobalGet { global_index },
                Call { function_index },
            ]);
        }
        out.extend([op]);
        Ok(())
    }

    fn name(&self) -> &'static str {
        "heap bound"
    }
}
