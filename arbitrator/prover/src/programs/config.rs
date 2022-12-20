// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use super::{
    depth::DepthChecker, heap::HeapBound, meter::Meter, start::StartMover, MiddlewareWrapper,
};

use eyre::Result;
use wasmer::{wasmparser::Operator, CompilerConfig, Store};
use wasmer_compiler_singlepass::Singlepass;
use wasmer_types::{Bytes, Pages};

use std::sync::Arc;

pub type Pricing = fn(&Operator) -> u64;

#[repr(C)]
#[derive(Clone)]
pub struct PolyglotConfig {
    pub costs: Pricing,
    pub start_gas: u64,
    pub max_depth: u32,
    pub heap_bound: Bytes,
}

impl Default for PolyglotConfig {
    fn default() -> Self {
        let costs = |_: &Operator| 0;
        Self {
            costs,
            start_gas: 0,
            max_depth: u32::MAX,
            heap_bound: Bytes(0),
        }
    }
}

impl PolyglotConfig {
    pub fn new(costs: Pricing, start_gas: u64, max_depth: u32, heap_bound: Bytes) -> Result<Self> {
        Pages::try_from(heap_bound)?; // ensure the limit represents a number of pages
        Ok(Self {
            costs,
            start_gas,
            max_depth,
            heap_bound,
        })
    }

    pub fn store(&self) -> Store {
        let mut compiler = Singlepass::new();
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();

        let meter = MiddlewareWrapper::new(Meter::new(self.costs, self.start_gas));
        let depth = MiddlewareWrapper::new(DepthChecker::new(self.max_depth));
        let bound = MiddlewareWrapper::new(HeapBound::new(self.heap_bound).unwrap()); // checked in new()
        let start = MiddlewareWrapper::new(StartMover::default());

        // add the instrumentation in the order of application
        compiler.push_middleware(Arc::new(meter));
        compiler.push_middleware(Arc::new(depth));
        compiler.push_middleware(Arc::new(bound));
        compiler.push_middleware(Arc::new(start));

        Store::new(compiler)
    }
}
