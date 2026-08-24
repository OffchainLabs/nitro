//! Block production output and error types.

use alloy_primitives::B256;

/// Result of producing a block.
#[derive(Debug, Clone)]
pub struct ProducedBlock {
    /// Hash of the produced block.
    pub block_hash: B256,
    /// Send root from the block's extra_data.
    pub send_root: B256,
}

/// Error type for block production.
#[derive(Debug, thiserror::Error)]
pub enum BlockProducerError {
    #[error("state access: {0}")]
    StateAccess(String),
    #[error("execution: {0}")]
    Execution(String),
    #[error("storage: {0}")]
    Storage(String),
    #[error("parse: {0}")]
    Parse(String),
    #[error("unexpected: {0}")]
    Unexpected(String),
}
