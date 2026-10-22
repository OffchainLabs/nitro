use alloy_primitives::U256;

/// Arbitrum consensus engine configuration.
#[derive(Debug, Clone)]
pub struct ArbEngine {
    pub is_sequencer: bool,
}

impl ArbEngine {
    pub fn new(is_sequencer: bool) -> Self {
        Self { is_sequencer }
    }
}

/// Difficulty for Arbitrum blocks is always 1.
pub const ARB_BLOCK_DIFFICULTY: U256 = U256::from_limbs([1, 0, 0, 0]);
