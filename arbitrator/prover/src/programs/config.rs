// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::field_reassign_with_default)]

use crate::{programs::meter, value::FunctionType};
use derivative::Derivative;
use fnv::FnvHashMap as HashMap;
use std::fmt::Debug;
use wasmer_types::{Pages, SignatureIndex, WASM_PAGE_SIZE};
use wasmparser::Operator;

#[cfg(feature = "native")]
use {
    super::{
        counter::Counter, depth::DepthChecker, dynamic::DynamicMeter, heap::HeapBound,
        meter::Meter, start::StartMover, MiddlewareWrapper,
    },
    std::sync::Arc,
    wasmer::{Cranelift, CraneliftOptLevel, Engine, Store},
    wasmer_compiler_singlepass::Singlepass,
};

#[derive(Clone, Copy, Debug)]
#[repr(C)]
pub struct StylusConfig {
    /// Version the program was compiled against
    pub version: u16,
    /// The maximum size of the stack, measured in words
    pub max_depth: u32,
    /// Pricing parameters supplied at runtime
    pub pricing: PricingParams,
}

#[derive(Clone, Copy, Debug)]
#[repr(C)]
pub struct PricingParams {
    /// The price of ink, measured in bips of an evm gas
    pub ink_price: u32,
}

impl Default for StylusConfig {
    fn default() -> Self {
        Self {
            version: 0,
            max_depth: u32::MAX,
            pricing: PricingParams::default(),
        }
    }
}

impl Default for PricingParams {
    fn default() -> Self {
        Self { ink_price: 1 }
    }
}

impl StylusConfig {
    pub const fn new(version: u16, max_depth: u32, ink_price: u32) -> Self {
        let pricing = PricingParams::new(ink_price);
        Self {
            version,
            max_depth,
            pricing,
        }
    }
}

#[allow(clippy::inconsistent_digit_grouping)]
impl PricingParams {
    pub const fn new(ink_price: u32) -> Self {
        Self { ink_price }
    }

    pub fn gas_to_ink(&self, gas: u64) -> u64 {
        gas.saturating_mul(self.ink_price.into())
    }

    pub fn ink_to_gas(&self, ink: u64) -> u64 {
        ink / self.ink_price as u64 // never 0
    }
}

pub type SigMap = HashMap<SignatureIndex, FunctionType>;
pub type OpCosts = fn(&Operator, &SigMap) -> u64;

#[derive(Clone, Debug, Default)]
pub struct CompileConfig {
    /// Version of the compiler to use
    pub version: u16,
    /// Pricing parameters used for metering
    pub pricing: CompilePricingParams,
    /// Memory bounds
    pub bounds: CompileMemoryParams,
    /// Debug parameters for test chains
    pub debug: CompileDebugParams,
}

#[derive(Clone, Copy, Debug)]
pub struct CompileMemoryParams {
    /// The maximum number of pages a program may start with
    pub heap_bound: Pages,
    /// The maximum size of a stack frame, measured in words
    pub max_frame_size: u32,
    /// The maximum number of overlapping value lifetimes in a frame
    pub max_frame_contention: u16,
}

#[derive(Clone, Derivative)]
#[derivative(Debug)]
pub struct CompilePricingParams {
    /// Associates opcodes to their ink costs
    #[derivative(Debug = "ignore")]
    pub costs: OpCosts,
    /// Cost of checking the amount of ink left.
    pub ink_header_cost: u64,
    /// Per-byte `MemoryFill` cost
    pub memory_fill_ink: u64,
    /// Per-byte `MemoryCopy` cost
    pub memory_copy_ink: u64,
}

#[derive(Clone, Debug, Default)]
pub struct CompileDebugParams {
    /// Allow debug functions
    pub debug_funcs: bool,
    /// Add instrumentation to count the number of times each kind of opcode is executed
    pub count_ops: bool,
    /// Whether to use the Cranelift compiler
    pub cranelift: bool,
}

impl Default for CompilePricingParams {
    fn default() -> Self {
        Self {
            costs: |_, _| 0,
            ink_header_cost: 0,
            memory_fill_ink: 0,
            memory_copy_ink: 0,
        }
    }
}

impl Default for CompileMemoryParams {
    fn default() -> Self {
        Self {
            heap_bound: Pages(u32::MAX / WASM_PAGE_SIZE as u32),
            max_frame_size: u32::MAX,
            max_frame_contention: u16::MAX,
        }
    }
}

impl CompileConfig {
    pub fn version(version: u16, debug_chain: bool) -> Self {
        let mut config = Self::default();
        config.version = version;
        config.debug.debug_funcs = debug_chain;

        match version {
            0 => {}
            1 => {
                // TODO: settle on reasonable values for the v1 release
                config.bounds.heap_bound = Pages(128); // 8 mb
                config.bounds.max_frame_size = 10 * 1024;
                config.bounds.max_frame_contention = 4096;
                config.pricing = CompilePricingParams {
                    costs: meter::pricing_v1,
                    ink_header_cost: 2450,
                    memory_fill_ink: 800 / 8,
                    memory_copy_ink: 800 / 8,
                };
            }
            _ => panic!("no config exists for Stylus version {version}"),
        }

        config
    }

    #[cfg(feature = "native")]
    pub fn store(&self) -> Store {
        let mut compiler: Box<dyn wasmer::CompilerConfig> = match self.debug.cranelift {
            true => {
                let mut compiler = Cranelift::new();
                compiler.opt_level(CraneliftOptLevel::Speed);
                Box::new(compiler)
            }
            false => Box::new(Singlepass::new()),
        };
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();

        let meter = MiddlewareWrapper::new(Meter::new(&self.pricing));
        let dygas = MiddlewareWrapper::new(DynamicMeter::new(&self.pricing));
        let depth = MiddlewareWrapper::new(DepthChecker::new(self.bounds));
        let bound = MiddlewareWrapper::new(HeapBound::new(self.bounds));
        let start = MiddlewareWrapper::new(StartMover::default());

        // add the instrumentation in the order of application
        // note: this must be consistent with the prover
        compiler.push_middleware(Arc::new(meter));
        compiler.push_middleware(Arc::new(dygas));
        compiler.push_middleware(Arc::new(depth));
        compiler.push_middleware(Arc::new(bound));
        compiler.push_middleware(Arc::new(start));

        if self.debug.count_ops {
            let counter = Counter::new();
            compiler.push_middleware(Arc::new(MiddlewareWrapper::new(counter)));
        }

        Store::new(compiler)
    }

    #[cfg(feature = "native")]
    pub fn engine(&self) -> Engine {
        self.store().engine().clone()
    }
}
