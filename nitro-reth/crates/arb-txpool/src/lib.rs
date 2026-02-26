//! Arbitrum transaction pool.
//!
//! For Arbitrum L2 nodes, the transaction pool is typically a no-op
//! since the sequencer manages its own mempool. This crate provides
//! the pool builder and validator types needed by the node builder.

/// Arbitrum transaction validator.
///
/// Performs basic validation on incoming transactions. For sequencer
/// mode, more sophisticated ordering and filtering is applied.
#[derive(Debug, Clone)]
pub struct ArbTransactionValidator;

impl ArbTransactionValidator {
    /// Create a new validator.
    pub fn new() -> Self {
        Self
    }
}

impl Default for ArbTransactionValidator {
    fn default() -> Self {
        Self::new()
    }
}
