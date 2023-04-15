// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use super::{config::CompileMemoryParams, DefaultFuncMiddleware, Middleware, ModuleMod};
use eyre::Result;
use wasmer_types::{LocalFunctionIndex, Pages};

#[derive(Debug)]
pub struct HeapBound {
    /// Upper bounds the amount of heap memory a module may use
    limit: Pages,
}

impl HeapBound {
    pub fn new(bounds: CompileMemoryParams) -> Self {
        let limit = bounds.heap_bound;
        Self { limit }
    }
}

impl<M: ModuleMod> Middleware<M> for HeapBound {
    type FM<'a> = DefaultFuncMiddleware;

    fn update_module(&self, module: &mut M) -> Result<()> {
        module.limit_heap(self.limit)
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(DefaultFuncMiddleware)
    }

    fn name(&self) -> &'static str {
        "heap bound"
    }
}
