//! `meldataprovider` JSON-RPC namespace: the read/query/control surface a MEL
//! provider (this node) serves to a Nitro node, so a Nitro node can consume MEL
//! extraction results over RPC instead of running its own in-process extractor.
//!
//! The wire contract mirrors nitro (`arbnode/mel/rpc_runner`,
//! namespace const `mel.RPCNamespace = "meldataprovider"`). Any change to a method
//! name, argument, return shape, or error string must land in BOTH repos.

use alloy_primitives::B256;
use arb_mel_types::{BatchMetadata, DelayedInboxMessage, MelState, MessageSyncProgress};
use arbos::types::L1IncomingMessage;
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use serde::{Deserialize, Serialize};

use crate::nitro_execution::RpcL1IncomingMessage;

pub type MelProviderResult<T> = Result<T, MelProviderError>;

#[derive(Debug, thiserror::Error)]
pub enum MelProviderError {
    /// A requested item (state, head) was absent.
    #[error("not found: {0}")]
    NotFound(String),
    /// A delayed message index at/beyond the head `DelayedMessagesSeen` count.
    /// The string mirrors nitro's `MessageExtractor.GetDelayedMessage`.
    #[error(
        "DelayedInboxMessage not available for index: {index} greater than head MEL state DelayedMessagesSeen count: {count}"
    )]
    DelayedMessageOutOfBounds { index: u64, count: u64 },
    /// A batch seqNum at/beyond the head batch count. The string mirrors nitro's
    /// `MessageExtractor.GetBatchMetadata`; "not found" is load-bearing (BOLD's
    /// state provider substring-matches it to detect chain-catching-up).
    #[error(
        "batchMetadata not found for seqNum: {seq_num} greater than head MEL state batch count: {count}"
    )]
    BatchMetadataNotFound { seq_num: u64, count: u64 },
    /// `FindMessageOriginMELState` found no batch containing the message.
    /// The string is nitro's exact error (`MessageExtractor.FindMessageOriginMELState`).
    #[error("batch containing message not found")]
    MessageOriginNotFound,
    /// A finalized delayed message's `before_inbox_acc` did not match the caller's
    /// expected accumulator. The string is the exact Nitro sentinel (client substring-matches it).
    #[error("delayed message accumulator mismatch")]
    AccumulatorMismatch,
    /// `FindParentChainBlockContainingDelayed` is intentionally unimplemented by MEL.
    /// The string is the exact Nitro sentinel (client substring-matches it).
    #[error(
        "FindParentChainBlockContainingDelayed is not implemented by MEL as batch gas cost data is already filled in during extraction"
    )]
    FindDelayedNotImplemented,
    /// Anything the concrete backing failed at (DB read, L1 fetch, extraction).
    #[error("mel backing error: {0}")]
    Backing(String),
}

#[derive(Debug, Clone, Copy)]
pub enum L1BlockTag {
    Safe,
    Finalized,
}

/// Result of `finalized_delayed_message_at_position`. `not_yet_finalized`
/// is carried in-band (not an error) so `parent_chain_block_number` survives the
/// round-trip, mirroring nitro's `FinalizedDelayedResult`.
pub struct MelFinalizedDelayed {
    pub message: Option<L1IncomingMessage>,
    pub after_inbox_acc: B256,
    pub parent_chain_block_number: u64,
    pub not_yet_finalized: bool,
}

/// Wire form of a delayed inbox message: PascalCase outer (nitro's tagless
/// `mel.DelayedInboxMessage`), with `message` keeping its own camelCase tags.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcDelayedInboxMessage {
    pub block_hash: B256,
    pub before_inbox_acc: B256,
    pub message: RpcL1IncomingMessage,
    pub parent_chain_block_number: u64,
}

impl From<&DelayedInboxMessage> for RpcDelayedInboxMessage {
    fn from(d: &DelayedInboxMessage) -> Self {
        Self {
            block_hash: d.block_hash,
            before_inbox_acc: d.before_inbox_acc,
            message: RpcL1IncomingMessage::from(&d.message),
            parent_chain_block_number: d.parent_chain_block_number,
        }
    }
}

/// `mel.SequencerMessageResult`. `data` is base64-encoded bytes.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RpcSequencerMessageResult {
    pub data: String,
    pub block_hash: B256,
}

/// `mel.FindInboxBatchResult`
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RpcFindInboxBatchResult {
    pub seq_num: u64,
    pub found: bool,
}

/// `mel.FinalizedDelayedResult`. `not_yet_finalized` is carried
/// in-band so `parent_chain_block_number` survives (matches nitro).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct RpcFinalizedDelayedResult {
    pub message: Option<RpcL1IncomingMessage>,
    pub after_inbox_acc: B256,
    pub parent_chain_block_number: u64,
    pub not_yet_finalized: bool,
}

/// The read/query/control seam a MEL provider backing implements.
///
/// Primitives are implemented per backing; the default methods carry the shared,
/// nitro-faithful derivations (bounds checks, safe/finalized clamp, etc.).
#[async_trait::async_trait]
pub trait MelProvider: Send + Sync + 'static {
    // Required primitives: implemented by each concrete backing.
    async fn head_state(&self) -> MelProviderResult<MelState>;
    async fn state(&self, parent_chain_block_number: u64) -> MelProviderResult<Option<MelState>>;
    async fn raw_delayed_message(&self, index: u64) -> MelProviderResult<DelayedInboxMessage>;
    async fn raw_batch_metadata(&self, seq_num: u64) -> MelProviderResult<BatchMetadata>;
    async fn resolve_l1_block(&self, tag: L1BlockTag) -> MelProviderResult<u64>;
    async fn sequencer_message_bytes_for_parent_block(
        &self,
        seq_num: u64,
        parent_chain_block: u64,
    ) -> MelProviderResult<(Vec<u8>, B256)>;
    async fn find_message_origin_mel_state(&self, pos: u64) -> MelProviderResult<Option<MelState>>;
    async fn caught_up(&self) -> MelProviderResult<bool>;
    async fn reorg_to(&self, parent_chain_block_number: u64) -> MelProviderResult<()>;

    // Derived queries: default implementations shared by every backing.
    async fn get_msg_count(&self) -> MelProviderResult<u64> {
        Ok(self.head_state().await?.msg_count)
    }

    async fn get_delayed_count(&self) -> MelProviderResult<u64> {
        Ok(self.head_state().await?.delayed_messages_seen)
    }

    async fn get_batch_count(&self) -> MelProviderResult<u64> {
        Ok(self.head_state().await?.batch_count)
    }

    async fn get_delayed_message(&self, index: u64) -> MelProviderResult<DelayedInboxMessage> {
        let seen = self.head_state().await?.delayed_messages_seen;
        if index >= seen {
            return Err(MelProviderError::DelayedMessageOutOfBounds { index, count: seen });
        }
        self.raw_delayed_message(index).await
    }

    async fn get_delayed_message_bytes(&self, seq_num: u64) -> MelProviderResult<Vec<u8>> {
        Ok(self.get_delayed_message(seq_num).await?.message.serialize())
    }

    async fn get_delayed_acc(&self, seq_num: u64) -> MelProviderResult<B256> {
        Ok(self.get_delayed_message(seq_num).await?.after_inbox_acc())
    }

    async fn get_batch_metadata(&self, seq_num: u64) -> MelProviderResult<BatchMetadata> {
        let count = self.head_state().await?.batch_count;
        if seq_num >= count {
            return Err(MelProviderError::BatchMetadataNotFound { seq_num, count });
        }
        self.raw_batch_metadata(seq_num).await
    }

    async fn get_batch_acc(&self, seq_num: u64) -> MelProviderResult<B256> {
        Ok(self.get_batch_metadata(seq_num).await?.accumulator)
    }

    async fn get_batch_message_count(&self, seq_num: u64) -> MelProviderResult<u64> {
        Ok(self.get_batch_metadata(seq_num).await?.message_count)
    }

    async fn get_batch_parent_chain_block(&self, seq_num: u64) -> MelProviderResult<u64> {
        Ok(self.get_batch_metadata(seq_num).await?.parent_chain_block)
    }

    async fn find_parent_chain_block_containing_delayed(
        &self,
        _index: u64,
    ) -> MelProviderResult<u64> {
        Err(MelProviderError::FindDelayedNotImplemented)
    }

    async fn supports_pushing_finality_data(&self) -> MelProviderResult<bool> {
        Ok(true)
    }

    async fn get_sequencer_message_bytes(
        &self,
        seq_num: u64,
    ) -> MelProviderResult<(Vec<u8>, B256)> {
        let meta = self.get_batch_metadata(seq_num).await?;
        self.sequencer_message_bytes_for_parent_block(seq_num, meta.parent_chain_block)
            .await
    }

    /// safe/finalized -> clamp `min(head_block, tag_block)` -> state
    /// (nitro's `MessageExtractor.getStateByRPCBlockNum`).
    async fn state_at_tag(&self, tag: L1BlockTag) -> MelProviderResult<MelState> {
        let blk = self.resolve_l1_block(tag).await?;
        let head_block = self.head_state().await?.parent_chain_block_number;
        let target = head_block.min(blk);
        self.state(target)
            .await?
            .ok_or_else(|| MelProviderError::NotFound(format!("state at block {target}")))
    }

    async fn get_safe_msg_count(&self) -> MelProviderResult<u64> {
        Ok(self.state_at_tag(L1BlockTag::Safe).await?.msg_count)
    }

    async fn get_finalized_msg_count(&self) -> MelProviderResult<u64> {
        Ok(self.state_at_tag(L1BlockTag::Finalized).await?.msg_count)
    }

    /// Find the sequencer batch whose message range contains message `pos`.
    ///
    /// Binary search over per-batch cumulative message counts. Returns `Some(seq)`
    /// for the containing batch, or `None` when `pos` is at or beyond the last
    /// batch's message count (i.e. not yet posted in any batch). Ported from
    /// nitro's `FindInboxBatchContainingMessage`.
    async fn find_inbox_batch_containing_message(
        &self,
        pos: u64,
    ) -> MelProviderResult<Option<u64>> {
        let batch_count = self.get_batch_count().await?;
        if batch_count == 0 {
            return Ok(None);
        }
        let mut low = 0u64;
        let mut high = batch_count - 1;
        if self.get_batch_message_count(high).await? <= pos {
            return Ok(None);
        }
        // Invariants each iteration: high >= low, msgCount(low-1) <= pos < msgCount(high),
        // so the target batch is always in [low, high].
        loop {
            let mid = low.midpoint(high);
            let count = self.get_batch_message_count(mid).await?;
            if count < pos {
                low = mid + 1;
            } else if count == pos {
                return Ok(Some(mid + 1));
            } else if count == pos + 1 || mid == low {
                return Ok(Some(mid));
            } else {
                high = mid;
            }
            if high == low {
                return Ok(Some(high));
            }
        }
    }

    /// Sync progress: batches seen/processed and messages produced.
    ///
    /// Without an on-chain sequencer-batch counter wired in, `batch_seen` equals
    /// `batch_processed` (the head state's `batch_count`); a backing with a counter
    /// can override this to report batches posted but not yet processed. Mirrors
    /// nitro's `GetSyncProgress` fallback path.
    async fn get_sync_progress(&self) -> MelProviderResult<MessageSyncProgress> {
        // TODO(NIT-5431) allow override for sequencer "seen" values, if needed
        let head = self.head_state().await?;
        Ok(MessageSyncProgress {
            batch_seen: head.batch_count,
            batch_processed: head.batch_count,
            msg_count: head.msg_count,
        })
    }

    /// The delayed message at `pos`, if it is finalized as of `finalized_block`.
    ///
    /// `not_yet_finalized` is returned in-band (with `parent_chain_block_number`
    /// populated, `message` `None`) rather than as an error when MEL has not
    /// processed `finalized_block` yet, or when `pos` is at/beyond the finalized
    /// delayed count, so the delayed sequencer's short-circuit still gets the block
    /// number. When `last_delayed_acc` is non-zero it must equal the message's
    /// `before_inbox_acc`, else [`MelProviderError::AccumulatorMismatch`]. Ported from
    /// nitro's `FinalizedDelayedMessageAtPosition`.
    async fn finalized_delayed_message_at_position(
        &self,
        finalized_block: u64,
        last_delayed_acc: B256,
        pos: u64,
    ) -> MelProviderResult<MelFinalizedDelayed> {
        let msg = self.get_delayed_message(pos).await?;

        // A missing state at `finalized_block` means MEL has not processed that
        // block yet: treat as not-yet-finalized (nitro's rawdb-not-found path).
        let finalized_delayed_count = match self.state(finalized_block).await? {
            Some(state) => state.delayed_messages_seen,
            None => {
                return Ok(MelFinalizedDelayed {
                    message: None,
                    after_inbox_acc: B256::ZERO,
                    parent_chain_block_number: msg.parent_chain_block_number,
                    not_yet_finalized: true,
                });
            }
        };

        if pos >= finalized_delayed_count {
            return Ok(MelFinalizedDelayed {
                message: None,
                after_inbox_acc: B256::ZERO,
                parent_chain_block_number: msg.parent_chain_block_number,
                not_yet_finalized: true,
            });
        }

        if last_delayed_acc != B256::ZERO && msg.before_inbox_acc != last_delayed_acc {
            return Err(MelProviderError::AccumulatorMismatch);
        }

        Ok(MelFinalizedDelayed {
            after_inbox_acc: msg.after_inbox_acc(),
            parent_chain_block_number: msg.parent_chain_block_number,
            message: Some(msg.message),
            not_yet_finalized: false,
        })
    }

    /// TODO(NIT-5119): needs accumulator preimage recording in arb-mel. Stub until then.
    async fn get_preimages_for_validation(
        &self,
        _last_validated_parent_chain_block: u64,
        _validate_msg_extraction_till: u64,
    ) -> MelProviderResult<()> {
        Err(MelProviderError::Backing(
            "getPreimagesForValidation not yet supported".into(),
        ))
    }
}

/// The `meldataprovider` JSON-RPC surface a Nitro node calls on this provider.
/// Method names/args/returns mirror nitro.
#[rpc(server, namespace = "meldataprovider")]
pub trait MelApi {
    // Batch queries
    #[method(name = "getBatchCount")]
    async fn get_batch_count(&self) -> RpcResult<u64>;
    #[method(name = "getBatchMessageCount")]
    async fn get_batch_message_count(&self, seq_num: u64) -> RpcResult<u64>;
    #[method(name = "getBatchMetadata")]
    async fn get_batch_metadata(&self, seq_num: u64) -> RpcResult<BatchMetadata>;
    #[method(name = "getBatchAcc")]
    async fn get_batch_acc(&self, seq_num: u64) -> RpcResult<B256>;
    #[method(name = "getBatchParentChainBlock")]
    async fn get_batch_parent_chain_block(&self, seq_num: u64) -> RpcResult<u64>;
    #[method(name = "findInboxBatchContainingMessage")]
    async fn find_inbox_batch_containing_message(
        &self,
        pos: u64,
    ) -> RpcResult<RpcFindInboxBatchResult>;

    // Delayed-message queries
    #[method(name = "getDelayedCount")]
    async fn get_delayed_count(&self) -> RpcResult<u64>;
    #[method(name = "getDelayedMessage")]
    async fn get_delayed_message(&self, index: u64) -> RpcResult<RpcDelayedInboxMessage>;
    #[method(name = "getDelayedMessageBytes")]
    async fn get_delayed_message_bytes(&self, seq_num: u64) -> RpcResult<String>; // base64
    #[method(name = "getDelayedAcc")]
    async fn get_delayed_acc(&self, seq_num: u64) -> RpcResult<B256>;
    #[method(name = "findParentChainBlockContainingDelayed")]
    async fn find_parent_chain_block_containing_delayed(&self, index: u64) -> RpcResult<u64>;

    // Sequencer-message + finality
    #[method(name = "getSequencerMessageBytes")]
    async fn get_sequencer_message_bytes(
        &self,
        seq_num: u64,
    ) -> RpcResult<RpcSequencerMessageResult>;
    #[method(name = "getSequencerMessageBytesForParentBlock")]
    async fn get_sequencer_message_bytes_for_parent_block(
        &self,
        seq_num: u64,
        parent_chain_block: u64,
    ) -> RpcResult<RpcSequencerMessageResult>;
    #[method(name = "finalizedDelayedMessageAtPosition")]
    async fn finalized_delayed_message_at_position(
        &self,
        finalized_block: u64,
        last_delayed_accumulator: B256,
        requested_position: u64,
    ) -> RpcResult<RpcFinalizedDelayedResult>;
    #[method(name = "getMsgCount")]
    async fn get_msg_count(&self) -> RpcResult<u64>;
    #[method(name = "getSafeMsgCount")]
    async fn get_safe_msg_count(&self) -> RpcResult<u64>;
    #[method(name = "getFinalizedMsgCount")]
    async fn get_finalized_msg_count(&self) -> RpcResult<u64>;
    #[method(name = "getSyncProgress")]
    async fn get_sync_progress(&self) -> RpcResult<MessageSyncProgress>;
    #[method(name = "supportsPushingFinalityData")]
    async fn supports_pushing_finality_data(&self) -> RpcResult<bool>;

    // MEL state queries
    #[method(name = "getState")]
    async fn get_state(&self, parent_chain_block_number: u64) -> RpcResult<MelState>;
    #[method(name = "getHeadState")]
    async fn get_head_state(&self) -> RpcResult<MelState>;
    #[method(name = "findMessageOriginMELState")]
    async fn find_message_origin_mel_state(&self, pos: u64) -> RpcResult<MelState>;

    // Lifecycle / control
    #[method(name = "caughtUp")]
    async fn caught_up(&self) -> RpcResult<bool>;
    #[method(name = "reorgTo")]
    async fn reorg_to(&self, parent_chain_block_number: u64) -> RpcResult<()>;

    // TODO(NIT-5119): needs preimage recording; wire result type TBD
    // #[method(name = "getPreimagesForValidation")]
    // async fn get_preimages_for_validation(&self, last_validated_parent_chain_block: u64,
    // validate_msg_extraction_till: u64) -> RpcResult<RpcGetPreimagesForValidationResult>;
}
