//! Handler for the `meldataprovider` RPC namespace.
//!
//! Adapts a [`MelProvider`] backing to the generated `MelApiServer` trait.

use std::sync::Arc;

use alloy_primitives::B256;
use arb_mel_types::{BatchMetadata, MelState, MessageSyncProgress};
use base64::Engine as _;
use jsonrpsee::core::RpcResult;

use crate::{
    mel::{
        MelApiServer, MelProvider, RpcDelayedInboxMessage, RpcFinalizedDelayedResult,
        RpcFindInboxBatchResult, RpcSequencerMessageResult,
    },
    nitro_execution::RpcL1IncomingMessage,
};

/// RPC handler serving `meldataprovider_*` over a type-erased [`MelProvider`] backing.
pub struct MelApiHandler {
    query: Arc<dyn MelProvider>,
}

impl MelApiHandler {
    /// Build a handler over the given backing.
    pub fn new(query: Arc<dyn MelProvider>) -> Self {
        Self { query }
    }
}

#[async_trait::async_trait]
impl MelApiServer for MelApiHandler {
    // Batch queries
    async fn get_batch_count(&self) -> RpcResult<u64> {
        Ok(self.query.get_batch_count().await?)
    }
    async fn get_batch_message_count(&self, seq_num: u64) -> RpcResult<u64> {
        Ok(self.query.get_batch_message_count(seq_num).await?)
    }
    async fn get_batch_metadata(&self, seq_num: u64) -> RpcResult<BatchMetadata> {
        Ok(self.query.get_batch_metadata(seq_num).await?)
    }
    async fn get_batch_acc(&self, seq_num: u64) -> RpcResult<B256> {
        Ok(self.query.get_batch_acc(seq_num).await?)
    }
    async fn get_batch_parent_chain_block(&self, seq_num: u64) -> RpcResult<u64> {
        Ok(self.query.get_batch_parent_chain_block(seq_num).await?)
    }
    async fn find_inbox_batch_containing_message(
        &self,
        pos: u64,
    ) -> RpcResult<RpcFindInboxBatchResult> {
        let found = self.query.find_inbox_batch_containing_message(pos).await?;
        Ok(RpcFindInboxBatchResult {
            seq_num: found.unwrap_or(0),
            found: found.is_some(),
        })
    }

    // Delayed-message queries
    async fn get_delayed_count(&self) -> RpcResult<u64> {
        Ok(self.query.get_delayed_count().await?)
    }
    async fn get_delayed_message(&self, index: u64) -> RpcResult<Option<RpcDelayedInboxMessage>> {
        let msg = self.query.get_delayed_message(index).await?;
        Ok(Some(RpcDelayedInboxMessage::from(&msg)))
    }
    async fn get_delayed_message_bytes(&self, seq_num: u64) -> RpcResult<String> {
        Ok(b64(&self.query.get_delayed_message_bytes(seq_num).await?))
    }
    async fn get_delayed_acc(&self, seq_num: u64) -> RpcResult<B256> {
        Ok(self.query.get_delayed_acc(seq_num).await?)
    }
    async fn find_parent_chain_block_containing_delayed(&self, index: u64) -> RpcResult<u64> {
        Ok(self
            .query
            .find_parent_chain_block_containing_delayed(index)
            .await?)
    }

    // Sequencer-message + finality
    async fn get_sequencer_message_bytes(
        &self,
        seq_num: u64,
    ) -> RpcResult<RpcSequencerMessageResult> {
        let (data, block_hash) = self.query.get_sequencer_message_bytes(seq_num).await?;
        Ok(RpcSequencerMessageResult {
            data: b64(&data),
            block_hash,
        })
    }
    async fn get_sequencer_message_bytes_for_parent_block(
        &self,
        seq_num: u64,
        parent_chain_block: u64,
    ) -> RpcResult<RpcSequencerMessageResult> {
        let (data, block_hash) = self
            .query
            .sequencer_message_bytes_for_parent_block(seq_num, parent_chain_block)
            .await?;
        Ok(RpcSequencerMessageResult {
            data: b64(&data),
            block_hash,
        })
    }
    async fn finalized_delayed_message_at_position(
        &self,
        finalized_block: u64,
        last_delayed_accumulator: B256,
        requested_position: u64,
    ) -> RpcResult<RpcFinalizedDelayedResult> {
        let r = self
            .query
            .finalized_delayed_message_at_position(
                finalized_block,
                last_delayed_accumulator,
                requested_position,
            )
            .await?;
        Ok(RpcFinalizedDelayedResult {
            message: r.message.as_ref().map(RpcL1IncomingMessage::from),
            after_inbox_acc: r.after_inbox_acc,
            parent_chain_block_number: r.parent_chain_block_number,
            not_yet_finalized: r.not_yet_finalized,
        })
    }
    async fn get_msg_count(&self) -> RpcResult<u64> {
        Ok(self.query.get_msg_count().await?)
    }
    async fn get_safe_msg_count(&self) -> RpcResult<u64> {
        Ok(self.query.get_safe_msg_count().await?)
    }
    async fn get_finalized_msg_count(&self) -> RpcResult<u64> {
        Ok(self.query.get_finalized_msg_count().await?)
    }
    async fn get_sync_progress(&self) -> RpcResult<MessageSyncProgress> {
        Ok(self.query.get_sync_progress().await?)
    }
    async fn supports_pushing_finality_data(&self) -> RpcResult<bool> {
        Ok(self.query.supports_pushing_finality_data().await?)
    }

    // MEL state queries
    async fn get_state(&self, parent_chain_block_number: u64) -> RpcResult<Option<MelState>> {
        Ok(self.query.state(parent_chain_block_number).await?)
    }
    async fn get_head_state(&self) -> RpcResult<MelState> {
        Ok(self.query.head_state().await?)
    }
    async fn find_message_origin_mel_state(&self, pos: u64) -> RpcResult<Option<MelState>> {
        Ok(self.query.find_message_origin_mel_state(pos).await?)
    }

    // Lifecycle / control
    async fn caught_up(&self) -> RpcResult<bool> {
        Ok(self.query.caught_up().await?)
    }
    async fn reorg_to(&self, parent_chain_block_number: u64) -> RpcResult<()> {
        Ok(self.query.reorg_to(parent_chain_block_number).await?)
    }
}

/// Standard-padded base64, matching Go's `encoding/json` `[]byte` marshaling.
fn b64(bytes: &[u8]) -> String {
    base64::engine::general_purpose::STANDARD.encode(bytes)
}
