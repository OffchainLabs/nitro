use alloy_primitives::Address;

/// The scenario in which tracing is occurring.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TracingScenario {
    /// Tracing a top-level transaction.
    Transaction,
    /// Tracing a precompile call within a transaction.
    Precompile,
    /// Tracing an ArbOS internal operation.
    ArbOS,
}

/// Tracing context passed through ArbOS for EVM debugging/tracing support.
#[derive(Debug, Clone)]
pub struct TracingInfo {
    pub scenario: TracingScenario,
    pub from: Address,
    pub to: Address,
    pub depth: usize,
}

impl TracingInfo {
    pub fn new(from: Address, to: Address, scenario: TracingScenario) -> Self {
        Self {
            scenario,
            from,
            to,
            depth: 0,
        }
    }
}
