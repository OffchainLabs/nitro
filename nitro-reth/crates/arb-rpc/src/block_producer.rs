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
}
