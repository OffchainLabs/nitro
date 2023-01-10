// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::operator::OperatorCode;
use eyre::Result;
use fnv::FnvHashMap as HashMap;
use parking_lot::Mutex;
use wasmer_types::{Bytes, Pages};
use wasmparser::Operator;

#[cfg(feature = "native")]
use {
    super::{
        counter::Counter, depth::DepthChecker, heap::HeapBound, meter::Meter, start::StartMover,
        MiddlewareWrapper,
    },
    std::sync::Arc,
    wasmer::{CompilerConfig, Store},
    wasmer_compiler_singlepass::Singlepass,
};

pub type OpCosts = fn(&Operator) -> u64;

#[repr(C)]
#[derive(Clone)]
pub struct StylusDebugConfig {
    pub enable_operator_count: bool,
    pub max_unique_operator_count: usize,
    pub opcode_indexes: Arc<Mutex<HashMap<OperatorCode, usize>>>,
}

impl Default for StylusDebugConfig {
    fn default() -> Self {
        Self {
            enable_operator_count: false,
            max_unique_operator_count: 0,
            opcode_indexes: Arc::new(Mutex::new(HashMap::default())),
        }
    }
}

impl StylusDebugConfig {
    pub fn new(enable_operator_count: bool, max_unique_operator_count: usize) -> Result<Self> {
        let opcode_indexes =
            HashMap::with_capacity_and_hasher(max_unique_operator_count, Default::default());
        Ok(Self {
            enable_operator_count,
            max_unique_operator_count,
            opcode_indexes: Arc::new(Mutex::new(opcode_indexes)),
        })
    }
}

#[repr(C)]
#[derive(Clone)]
pub struct StylusConfig {
    pub costs: OpCosts,
    pub start_gas: u64,
    pub max_depth: u32,
    pub heap_bound: Bytes,
    pub pricing: PricingParams,
    pub debug: Option<StylusDebugConfig>,
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
            max_depth: u32::MAX,
            heap_bound: Bytes(u32::MAX as usize),
            pricing: PricingParams::default(),
            debug: None,
        }
    }
}

impl PricingParams {
    pub fn new(wasm_gas_price: u64, hostio_cost: u64) -> Self {
        Self {
            wasm_gas_price,
            hostio_cost,
        }
    }
}

impl StylusConfig {
    pub fn new(
        costs: OpCosts,
        start_gas: u64,
        max_depth: u32,
        heap_bound: Bytes,
        wasm_gas_price: u64,
        hostio_cost: u64,
        debug: Option<StylusDebugConfig>,
    ) -> Result<Self> {
        let pricing = PricingParams::new(wasm_gas_price, hostio_cost);
        Pages::try_from(heap_bound)?; // ensure the limit represents a number of pages
        Ok(Self {
            costs,
            start_gas,
            max_depth,
            heap_bound,
            pricing,
            debug,
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
        // note: this must be consistent with the prover
        compiler.push_middleware(Arc::new(meter));
        compiler.push_middleware(Arc::new(depth));
        compiler.push_middleware(Arc::new(bound));
        compiler.push_middleware(Arc::new(start));

        if self.debug.is_some() {
            let debug = self.debug.as_ref();

            if debug.unwrap().enable_operator_count {
                let counter = Counter::new(
                    debug.unwrap().max_unique_operator_count,
                    debug.unwrap().opcode_indexes.clone(),
                );
                compiler.push_middleware(Arc::new(MiddlewareWrapper::new(counter)));
            }
        }

        Store::new(compiler)
    }
}
