// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{DefaultFuncMiddleware, Middleware, ModuleMod};
use arbutil::Color;
use eyre::{bail, Result};
use wasmer_types::{Bytes, LocalFunctionIndex, Pages};

#[derive(Debug)]
pub struct HeapBound {
    /// Upper bounds the amount of heap memory a module may use
    limit: Bytes,
}

impl HeapBound {
    pub fn new(limit: Bytes) -> Result<Self> {
        Pages::try_from(limit)?;
        Ok(Self { limit })
    }
}

impl<M: ModuleMod> Middleware<M> for HeapBound {
    type FM<'a> = DefaultFuncMiddleware;

    fn update_module(&self, module: &mut M) -> Result<()> {
        let Bytes(static_size) = module.static_size();
        let Bytes(limit) = self.limit;
        if static_size > limit {
            bail!("module data exceeds memory limit: {} > {}", static_size.red(), limit.red())
        }
        let limit = Bytes(limit - static_size);
        let limit = Pages::try_from(limit).unwrap(); // checked in new()
        module.limit_heap(limit)
    }

    fn instrument<'a>(&self, _: LocalFunctionIndex) -> Result<Self::FM<'a>> {
        Ok(DefaultFuncMiddleware)
    }

    fn name(&self) -> &'static str {
        "heap bound"
    }
}
