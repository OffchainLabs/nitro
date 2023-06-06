// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::field_reassign_with_default)]

use arbutil::evm::EvmData;
use derivative::Derivative;
use fixed::types::U32F32;
use std::fmt::Debug;
use wasmer_types::{Pages, WASM_PAGE_SIZE};
use wasmparser::Operator;

#[cfg(feature = "native")]
use {
    super::{
        counter::Counter, depth::DepthChecker, dynamic::DynamicMeter, heap::HeapBound,
        meter::Meter, start::StartMover, MiddlewareWrapper,
    },
    std::sync::Arc,
    wasmer::{Cranelift, CraneliftOptLevel, Store},
    wasmer_compiler_singlepass::Singlepass,
};

#[derive(Clone, Copy, Debug)]
#[repr(C)]
pub struct StylusConfig {
    /// Version the program was compiled against
    pub version: u32,
    /// The maximum size of the stack, measured in words
    pub max_depth: u32,
    /// Pricing parameters supplied at runtime
    pub pricing: PricingParams,
}

#[derive(Clone, Copy, Debug)]
#[repr(C)]
pub struct PricingParams {
    /// The price of ink, measured in bips of an evm gas
    pub ink_price: u64,
    /// The amount of ink one pays to do a user_host call
    pub hostio_ink: u64,
    /// Memory pricing model
    pub memory_model: MemoryModel,
}

#[derive(Clone, Copy, Debug)]
#[repr(C)]
pub struct MemoryModel {
    /// Number of pages a tx gets for free
    pub free_pages: u16,
    /// Base cost of each additional wasm page
    pub page_gas: u32,
    /// Ramps up exponential memory costs
    pub page_ramp: u32,
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
        Self {
            ink_price: 1,
            hostio_ink: 0,
            memory_model: MemoryModel::default(),
        }
    }
}

impl Default for MemoryModel {
    fn default() -> Self {
        Self {
            free_pages: u16::MAX,
            page_gas: 0,
            page_ramp: 0,
        }
    }
}

impl StylusConfig {
    pub const fn new(
        version: u32,
        max_depth: u32,
        ink_price: u64,
        hostio_ink: u64,
        memory_model: MemoryModel,
    ) -> Self {
        let pricing = PricingParams::new(ink_price, hostio_ink, memory_model);
        Self {
            version,
            max_depth,
            pricing,
        }
    }
}

#[allow(clippy::inconsistent_digit_grouping)]
impl PricingParams {
    pub const fn new(ink_price: u64, hostio_ink: u64, memory_model: MemoryModel) -> Self {
        Self {
            ink_price,
            hostio_ink,
            memory_model,
        }
    }

    pub fn gas_to_ink(&self, gas: u64) -> u64 {
        gas.saturating_mul(100_00) / self.ink_price
    }

    pub fn ink_to_gas(&self, ink: u64) -> u64 {
        ink.saturating_mul(self.ink_price) / 100_00
    }
}

impl MemoryModel {
    pub const fn new(free_pages: u16, page_gas: u32, page_ramp: u32) -> Self {
        Self {
            free_pages,
            page_gas,
            page_ramp,
        }
    }

    /// Determines the gas cost of allocating `new` pages given `open` are active and `ever` have ever been.
    pub fn gas_cost(&self, new: u16, open: u16, ever: u16) -> u64 {
        let ramp = U32F32::from_bits(self.page_ramp.into()) + U32F32::lit("1");
        let size = ever.max(open.saturating_add(new));

        // free until expansion beyond the first few
        if size <= self.free_pages {
            return 0;
        }

        // exponentiates ramp by squaring
        let curve = |mut exponent| {
            let mut result = U32F32::from_num(1);
            let mut base = ramp;

            while exponent > 0 {
                if exponent & 1 == 1 {
                    result = result.saturating_mul(base);
                }
                exponent /= 2;
                if exponent > 0 {
                    base = base.saturating_mul(base);
                }
            }
            result.to_num::<u64>()
        };

        let linear = (new as u64).saturating_mul(self.page_gas.into());
        let expand = curve(size) - curve(ever);
        linear.saturating_add(expand)
    }

    pub fn start_cost(&self, evm_data: &EvmData) -> u64 {
        let start = evm_data.start_pages;
        self.gas_cost(start.need, start.open, start.ever)
    }
}

pub type OpCosts = fn(&Operator) -> u64;

#[derive(Clone, Debug, Default)]
pub struct CompileConfig {
    /// Version of the compiler to use
    pub version: u32,
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
}

#[derive(Clone, Derivative)]
#[derivative(Debug)]
pub struct CompilePricingParams {
    /// Associates opcodes to their ink costs
    #[derivative(Debug = "ignore")]
    pub costs: OpCosts,
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
            costs: |_| 0,
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
        }
    }
}

impl CompileConfig {
    pub fn version(version: u32, debug_chain: bool) -> Self {
        let mut config = Self::default();
        config.version = version;
        config.debug.debug_funcs = debug_chain;

        match version {
            0 => {}
            1 => {
                // TODO: settle on reasonable values for the v1 release
                config.bounds.heap_bound = Pages(128); // 8 mb
                config.bounds.max_frame_size = 1024 * 1024;
                config.pricing = CompilePricingParams {
                    costs: |_| 1,
                    memory_fill_ink: 1,
                    memory_copy_ink: 1,
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

        let meter = MiddlewareWrapper::new(Meter::new(self.pricing.costs));
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
}
