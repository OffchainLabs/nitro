//! Block producer trait for the `nitroexecution` RPC handler.
//!
//! Defines the interface for producing blocks from L1 incoming messages.
//! The concrete implementation lives in `arb-node` where it has access
//! to the full node infrastructure (database, EVM config, state).

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

/// Input for block production from an L1 incoming message.
#[derive(Debug, Clone)]
pub struct BlockProductionInput {
    /// Message kind (L1MessageType_*).
    pub kind: u8,
    /// Message sender (poster address).
    pub sender: alloy_primitives::Address,
    /// L1 block number.
    pub l1_block_number: u64,
    /// L1 timestamp.
    pub l1_timestamp: u64,
    /// L1 request ID (for delayed messages).
    pub request_id: Option<B256>,
    /// L1 base fee.
    pub l1_base_fee: Option<alloy_primitives::U256>,
    /// L2 message payload (base64-decoded).
    pub l2_msg: Vec<u8>,
    /// Delayed messages read count.
    pub delayed_messages_read: u64,
    /// Legacy batch gas cost.
    pub batch_gas_cost: Option<u64>,
    /// Batch data stats (for newer batch posting reports).
    pub batch_data_stats: Option<(u64, u64)>,
}

/// Trait for producing blocks from L1 messages.
///
/// Implemented by the node infrastructure where full database and EVM
/// access is available.
#[async_trait::async_trait]
pub trait BlockProducer: Send + Sync + 'static {
    /// Cache the Init message params for later use during block 1 execution.
    ///
    /// The Init message (Kind=11) does NOT produce a block. Its params are
    /// applied during the first real block's pre-execution so that the
    /// state root for block 1 includes both Init and execution changes.
    fn cache_init_message(&self, l2_msg: &[u8]) -> Result<(), BlockProducerError>;

    /// Produce a block from the given L1 incoming message.
    ///
    /// The implementation should:
    /// 1. Parse the L1 message into transactions
    /// 2. Open the state at the current head
    /// 3. Execute transactions using the ArbOS pipeline
    /// 4. Compute the state root
    /// 5. Persist the block and state changes
    /// 6. Return the block hash and send root
    async fn produce_block(
        &self,
        msg_idx: u64,
        input: BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError>;

    /// Reset the canonical chain head to the given block number.
    ///
    /// Used by `nitroexecution_reorg` to roll back state before replaying
    /// divergent messages. The concrete implementation should truncate
    /// blocks above `target_block_number` from the canonical chain,
    /// making that block the new head. Receipts and transactions above
    /// the target should be removed.
    ///
    /// Default implementation returns an "unsupported" error. Node
    /// implementations that support reorg should override this.
    async fn reset_to_block(&self, _target_block_number: u64) -> Result<(), BlockProducerError> {
        Err(BlockProducerError::Unexpected(
            "reset_to_block not supported by this producer".into(),
        ))
    }

    /// Mark finality metadata (safe / finalized / validated block hashes)
    /// on the canonical chain.
    ///
    /// Nitro's consensus layer calls `setFinalityData` periodically to
    /// propagate finality information derived from L1 confirmations.
    /// The execution client should store these markers so that RPC
    /// queries like `eth_getBlockByNumber("finalized")` return the
    /// correct block.
    ///
    /// Default impl is a no-op; node implementations override if they
    /// support finality tracking.
    fn set_finality(
        &self,
        _safe: Option<B256>,
        _finalized: Option<B256>,
        _validated: Option<B256>,
    ) -> Result<(), BlockProducerError> {
        Ok(())
    }
}
