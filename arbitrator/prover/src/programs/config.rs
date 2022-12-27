// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::Result;
use wasmer_types::{Bytes, Pages};
use wasmparser::Operator;

#[cfg(feature = "native")]
use {
    super::{
        depth::DepthChecker, heap::HeapBound, meter::Meter, start::StartMover, MiddlewareWrapper,
    },
    std::sync::Arc,
    wasmer::{CompilerConfig, Store},
    wasmer_compiler_singlepass::Singlepass,
};

pub type Pricing = fn(&Operator) -> u64;

#[repr(C)]
#[derive(Clone)]
pub struct PolyglotConfig {
    pub costs: Pricing,
    pub start_gas: u64,
    pub max_depth: u32,
    pub heap_bound: Bytes,
    /// The price of wasm gas, measured in bips of an evm gas
    pub wasm_gas_price: u64,
    pub hostio_cost: u64,
}

impl Default for PolyglotConfig {
    fn default() -> Self {
        let costs = |_: &Operator| 0;
        Self {
            costs,
            start_gas: 0,
            max_depth: u32::MAX,
            heap_bound: Bytes(u32::MAX as usize),
            wasm_gas_price: 0,
            hostio_cost: 0,
        }
    }
}

impl PolyglotConfig {
    pub fn new(
        costs: Pricing,
        start_gas: u64,
        max_depth: u32,
        heap_bound: Bytes,
        wasm_gas_price: u64,
        hostio_cost: u64,
    ) -> Result<Self> {
        Pages::try_from(heap_bound)?; // ensure the limit represents a number of pages
        Ok(Self {
            costs,
            start_gas,
            max_depth,
            heap_bound,
            wasm_gas_price,
            hostio_cost,
        })
    }

    #[cfg(feature = "native")]
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
