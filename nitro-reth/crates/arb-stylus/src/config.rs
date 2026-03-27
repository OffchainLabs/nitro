use crate::ink::Ink;

use super::ink::Gas;

/// Runtime configuration for a Stylus program execution.
#[derive(Clone, Copy, Debug)]
pub struct StylusConfig {
    /// Version the program was compiled against.
    pub version: u16,
    /// Maximum stack depth in words.
    pub max_depth: u32,
    /// Pricing parameters for ink/gas conversion.
    pub pricing: PricingParams,
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

impl StylusConfig {
    pub const fn new(version: u16, max_depth: u32, ink_price: u32) -> Self {
        Self {
            version,
            max_depth,
            pricing: PricingParams::new(ink_price),
        }
    }
}

/// Pricing parameters for ink/gas conversion.
#[derive(Clone, Copy, Debug)]
pub struct PricingParams {
    /// The price of ink, measured in bips of an EVM gas.
    pub ink_price: u32,
}

impl Default for PricingParams {
    fn default() -> Self {
        Self { ink_price: 1 }
    }
}

impl PricingParams {
    pub const fn new(ink_price: u32) -> Self {
        Self { ink_price }
    }

    /// Convert EVM gas to ink.
    pub fn gas_to_ink(&self, gas: Gas) -> Ink {
        Ink(gas.0.saturating_mul(self.ink_price as u64))
    }

    /// Convert ink to EVM gas.
    pub fn ink_to_gas(&self, ink: Ink) -> Gas {
        Gas(ink.0 / self.ink_price as u64)
    }
}

/// Compile-time configuration for WASM module compilation.
#[derive(Clone, Debug, Default)]
pub struct CompileConfig {
    /// Version of the compiler to use.
    pub version: u16,
    /// Pricing parameters for metering.
    pub pricing: CompilePricingParams,
    /// Memory bounds.
    pub bounds: CompileMemoryParams,
    /// Debug parameters.
    pub debug: CompileDebugParams,
}

/// Memory bounds for WASM compilation.
#[derive(Clone, Copy, Debug)]
pub struct CompileMemoryParams {
    /// Maximum number of WASM pages a program may start with.
    pub heap_bound: u32,
    /// Maximum size of a stack frame in words.
    pub max_frame_size: u32,
    /// Maximum overlapping value lifetimes in a frame.
    pub max_frame_contention: u16,
}

impl Default for CompileMemoryParams {
    fn default() -> Self {
        Self {
            heap_bound: u32::MAX / 65536, // Pages(u32::MAX / WASM_PAGE_SIZE)
            max_frame_size: u32::MAX,
            max_frame_contention: u16::MAX,
        }
    }
}

/// Pricing parameters for WASM compilation.
#[derive(Clone, Debug, Default)]
pub struct CompilePricingParams {
    /// Cost of checking the amount of ink left.
    pub ink_header_cost: u64,
    /// Per-byte MemoryFill cost.
    pub memory_fill_ink: u64,
    /// Per-byte MemoryCopy cost.
    pub memory_copy_ink: u64,
}

/// Debug parameters for WASM compilation.
#[derive(Clone, Debug, Default)]
pub struct CompileDebugParams {
    /// Allow debug functions (console.log, etc.).
    pub debug_funcs: bool,
    /// Retain debug info in compiled modules.
    pub debug_info: bool,
    /// Add instrumentation to count opcode executions.
    pub count_ops: bool,
}

impl CompileConfig {
    /// Create a versioned compile config.
    pub fn version(version: u16, debug_chain: bool) -> Self {
        let mut config = Self {
            version,
            debug: CompileDebugParams {
                debug_funcs: debug_chain,
                debug_info: debug_chain,
                ..Default::default()
            },
            ..Default::default()
        };

        match version {
            0 => {}
            1 | 2 => {
                config.bounds.heap_bound = 128; // 128 pages = 8 MB
                config.bounds.max_frame_size = 10 * 1024;
                config.bounds.max_frame_contention = 4096;
                config.pricing = CompilePricingParams {
                    ink_header_cost: 2450,
                    memory_fill_ink: 800 / 8,
                    memory_copy_ink: 800 / 8,
                };
            }
            _ => unreachable!("unsupported Stylus version {version}"),
        }

        config
    }
}
