// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::{bail, Result};
use std::fmt::Debug;
use wasmer_types::Bytes;
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

pub type OpCosts = fn(&Operator) -> u64;

#[derive(Clone)]
pub struct StylusConfig {
    pub costs: OpCosts, // requires recompilation
    pub start_gas: u64,
    pub heap_bound: Bytes, // requires recompilation
    pub depth: DepthParams,
    pub pricing: PricingParams,
}

#[derive(Clone, Copy, Debug)]
pub struct DepthParams {
    pub max_depth: u32,
    pub max_frame_size: u32, // requires recompilation
}

#[derive(Clone, Copy, Debug, Default)]
pub struct PricingParams {
    /// The price of wasm gas, measured in bips of an evm gas
    pub wasm_gas_price: u64,
    /// The amount of wasm gas one pays to do a user_host call
    pub hostio_cost: u64,
}

impl Default for StylusConfig {
    fn default() -> Self {
        let costs = |_: &Operator| 0;
        Self {
            costs,
            start_gas: 0,
            heap_bound: Bytes(u32::MAX as usize),
            depth: DepthParams::default(),
            pricing: PricingParams::default(),
        }
    }
}

impl Default for DepthParams {
    fn default() -> Self {
        Self {
            max_depth: u32::MAX,
            max_frame_size: u32::MAX,
        }
    }
}

impl StylusConfig {
    pub fn version(version: u32) -> Self {
        let mut config = Self::default();
        match version {
            0 => {}
            1 => config.costs = |_| 1,
            _ => panic!("no config exists for Stylus version {version}"),
        }
        config
    }
}

impl DepthParams {
    pub fn new(max_depth: u32, max_frame_size: u32) -> Self {
        Self {
            max_depth,
            max_frame_size,
        }
    }
}

#[allow(clippy::inconsistent_digit_grouping)]
impl PricingParams {
    pub fn new(wasm_gas_price: u64, hostio_cost: u64) -> Self {
        Self {
            wasm_gas_price,
            hostio_cost,
        }
    }

    pub fn evm_to_wasm(&self, evm_gas: u64) -> Result<u64> {
        if self.wasm_gas_price == 0 {
            bail!("gas price is zero");
        }
        Ok(evm_gas.saturating_mul(100_00) / self.wasm_gas_price)
    }

    pub fn wasm_to_evm(&self, wasm_gas: u64) -> u64 {
        wasm_gas.saturating_mul(self.wasm_gas_price) / 100_00
    }
}

impl StylusConfig {
    #[cfg(feature = "native")]
    pub fn store(&self) -> Store {
        let mut compiler = Singlepass::new();
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();

        let meter = MiddlewareWrapper::new(Meter::new(self.costs, self.start_gas));
        let depth = MiddlewareWrapper::new(DepthChecker::new(self.depth));
        let bound = MiddlewareWrapper::new(HeapBound::new(self.heap_bound).unwrap()); // checked in new()
        let start = MiddlewareWrapper::new(StartMover::default());

        // add the instrumentation in the order of application
        // note: this must be consistent with the prover
        compiler.push_middleware(Arc::new(meter));
        compiler.push_middleware(Arc::new(depth));
        compiler.push_middleware(Arc::new(bound));
        compiler.push_middleware(Arc::new(start));

        Store::new(compiler)
    }

}

impl Debug for StylusConfig {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("StylusConfig")
            .field("costs", &"λ(op) -> u64")
            .field("start_gas", &self.start_gas)
            .field("heap_bound", &self.heap_bound)
            .field("depth", &self.depth)
            .field("pricing", &self.pricing)
            .finish()
    }
}

#[repr(C)]
pub struct GoParams {
    version: u32,
    max_depth: u32,
    max_frame_size: u32,
    heap_bound: u32,
    wasm_gas_price: u64,
    hostio_cost: u64,
}

impl GoParams {
    pub fn config(self) -> StylusConfig {
        let mut config = StylusConfig::version(self.version);
        config.depth = DepthParams::new(self.max_depth, self.max_frame_size);
        config.heap_bound = Bytes(self.heap_bound as usize);
        config.pricing.wasm_gas_price = self.wasm_gas_price;
        config.pricing.hostio_cost = self.hostio_cost;
        config
    }
}
