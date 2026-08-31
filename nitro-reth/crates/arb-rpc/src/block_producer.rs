//! Block producer trait for the `nitroexecution` RPC handler.
//!
//! Defines the interface for producing blocks from L1 incoming messages.
//! The concrete implementation lives in `arb-node` where it has access
//! to the full node infrastructure (database, EVM config, state).

use std::sync::Arc;

use alloy_primitives::B256;
pub use arb_evm::block_producer::{BlockProducerError, ProducedBlock};
use arbos::types::MessageWithMetadata;

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
        input: MessageWithMetadata,
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
    /// The consensus layer calls `setFinalityData` periodically to
    /// propagate finality info derived from L1 confirmations. The
    /// execution client stores these markers so RPC queries like
    /// `eth_getBlockByNumber("finalized")` return the correct block.
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

    /// Attach an external validated-block watcher. The producer should
    /// write the current validated marker into the provided shared
    /// slot on every `set_finality` call so RPC handlers (which hold
    /// the same handle) can surface the value.
    ///
    /// No-op by default; concrete producers override.
    fn attach_validated_watcher(&self, _watcher: Arc<parking_lot::RwLock<B256>>) {}
}
