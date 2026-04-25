//! Implementation of the `nitroexecution` RPC handler.
//!
//! Receives messages from the Nitro consensus layer, produces blocks,
//! and maintains the mapping between message indices and block numbers.

use std::sync::{Arc, OnceLock};

use alloy_consensus::BlockHeader;
use alloy_primitives::B256;
use alloy_rpc_types_eth::BlockNumberOrTag;
use base64::{
    alphabet,
    engine::{DecodePaddingMode, GeneralPurpose, GeneralPurposeConfig},
    Engine as _,
};
use jsonrpsee::core::RpcResult;
use parking_lot::RwLock;
use reth_provider::{BlockNumReader, BlockReaderIdExt, HeaderProvider};
use tracing::{debug, info};

use crate::{
    block_producer::{BlockProducer, BlockProductionInput},
    nitro_execution::{
        NitroExecutionApiServer, RpcConsensusSyncData, RpcFinalityData, RpcMaintenanceStatus,
        RpcMessageResult, RpcMessageWithMetadata, RpcMessageWithMetadataAndBlockInfo,
    },
};

/// State shared between the RPC handler and the node.
#[derive(Debug, Default)]
pub struct NitroExecutionState {
    /// Whether the node is synced with consensus.
    pub synced: bool,
    /// Maximum message count from consensus.
    pub max_message_count: u64,
}

/// Handler for the `nitroexecution` RPC namespace.
///
/// Receives L1 incoming messages from the consensus layer and produces blocks.
/// Delegates actual block production to the `BlockProducer` implementation.
pub struct NitroExecutionHandler<Provider, BP> {
    provider: Provider,
    block_producer: Arc<BP>,
    state: Arc<RwLock<NitroExecutionState>>,
    /// Genesis block number (0 for Arbitrum Sepolia, 22207817 for Arbitrum One).
    genesis_block_num: u64,
}

impl<Provider, BP> NitroExecutionHandler<Provider, BP> {
    /// Create a new handler with a block producer.
    pub fn new(provider: Provider, block_producer: Arc<BP>, genesis_block_num: u64) -> Self {
        Self {
            provider,
            block_producer,
            state: Arc::new(RwLock::new(NitroExecutionState::default())),
            genesis_block_num,
        }
    }

    /// Convert a message index to a block number.
    fn message_index_to_block_number(&self, msg_idx: u64) -> u64 {
        self.genesis_block_num + msg_idx
    }

    /// Convert a block number to a message index.
    fn block_number_to_message_index(&self, block_num: u64) -> Option<u64> {
        if block_num < self.genesis_block_num {
            return None;
        }
        Some(block_num - self.genesis_block_num)
    }
}

impl<Provider, BP> NitroExecutionHandler<Provider, BP>
where
    Provider: BlockReaderIdExt + HeaderProvider,
{
    /// Look up a sealed header by block number.
    fn get_header(
        &self,
        block_num: u64,
    ) -> Result<
        Option<reth_primitives_traits::SealedHeader<<Provider as HeaderProvider>::Header>>,
        String,
    > {
        self.provider
            .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(block_num))
            .map_err(|e| e.to_string())
    }

    /// Extract send root from a header's extra_data.
    fn send_root_from_header(header: &impl BlockHeader) -> B256 {
        let extra = header.extra_data();
        if extra.len() >= 32 {
            B256::from_slice(&extra[..32])
        } else {
            B256::ZERO
        }
    }
}

fn internal_error(msg: impl Into<String>) -> jsonrpsee::types::ErrorObjectOwned {
    jsonrpsee::types::ErrorObject::owned(
        jsonrpsee::types::error::INTERNAL_ERROR_CODE,
        msg.into(),
        None::<()>,
    )
}

/// Decode the l2_msg field from the RPC message.
///
/// JSON encoding always base64-encodes byte fields. The base64 output
/// can start with "0x" as valid base64 characters, so always decode as base64.
fn decode_l2_msg(l2_msg: &Option<String>) -> Result<Vec<u8>, String> {
    match l2_msg {
        Some(s) if !s.is_empty() => base64_decode(s).map_err(|e| format!("base64 decode: {e}")),
        _ => Ok(vec![]),
    }
}

const STANDARD_ALPHABET: &[u8; 64] =
    b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

fn base64_engine() -> &'static GeneralPurpose {
    static ENGINE: OnceLock<GeneralPurpose> = OnceLock::new();
    ENGINE.get_or_init(|| {
        let cfg = GeneralPurposeConfig::new()
            .with_decode_padding_mode(DecodePaddingMode::Indifferent)
            .with_decode_allow_trailing_bits(true);
        GeneralPurpose::new(&alphabet::STANDARD, cfg)
    })
}

fn base64_decode(input: &str) -> Result<Vec<u8>, String> {
    let stripped = input.trim_end_matches('=');
    // A length-1 mod 4 tail carries no meaningful bits; the orphan symbol
    // is still alphabet-validated, then discarded.
    let body_len = stripped.len() & !3;
    let tail = &stripped.as_bytes()[body_len..];
    let body = if tail.len() == 1 {
        let b = tail[0];
        if !STANDARD_ALPHABET.contains(&b) {
            return Err(format!("invalid base64 character: {}", b as char));
        }
        &stripped[..body_len]
    } else {
        stripped
    };
    base64_engine()
        .decode(body)
        .map_err(|e| format!("invalid base64: {e}"))
}

#[async_trait::async_trait]
impl<Provider, BP> NitroExecutionApiServer for NitroExecutionHandler<Provider, BP>
where
    Provider: BlockNumReader + BlockReaderIdExt + HeaderProvider + 'static,
    BP: BlockProducer,
{
    async fn digest_message(
        &self,
        msg_idx: u64,
        message: RpcMessageWithMetadata,
        _message_for_prefetch: Option<RpcMessageWithMetadata>,
    ) -> RpcResult<RpcMessageResult> {
        let block_num = self.message_index_to_block_number(msg_idx);
        let kind = message.message.header.kind;
        info!(target: "nitroexecution", msg_idx, block_num, kind, "digestMessage called");

        // Handle init message (Kind=11) — cache params, return genesis block.
        // The Init message does NOT produce a block. Its params are applied
        // during the first real block's execution.
        if kind == 11 {
            let l2_msg = decode_l2_msg(&message.message.l2_msg).map_err(internal_error)?;
            self.block_producer
                .cache_init_message(&l2_msg)
                .map_err(|e| internal_error(e.to_string()))?;

            // Return the genesis block info.
            let genesis_header = self
                .get_header(self.genesis_block_num)
                .map_err(internal_error)?
                .ok_or_else(|| internal_error("Genesis block not found for Init message"))?;
            let send_root = Self::send_root_from_header(genesis_header.header());
            info!(target: "nitroexecution", "Init message cached, returning genesis block");
            return Ok(RpcMessageResult {
                block_hash: genesis_header.hash(),
                send_root,
            });
        }

        // Check if we already have this block (idempotent).
        if let Some(header) = self.get_header(block_num).map_err(internal_error)? {
            let send_root = Self::send_root_from_header(header.header());
            debug!(target: "nitroexecution", block_num, ?send_root, "Block already exists");
            return Ok(RpcMessageResult {
                block_hash: header.hash(),
                send_root,
            });
        }

        // Decode the L2 message bytes
        let l2_msg = decode_l2_msg(&message.message.l2_msg).map_err(internal_error)?;

        // Build batch data stats if present
        let batch_data_stats = message
            .message
            .batch_data_tokens
            .as_ref()
            .map(|s| (s.length, s.nonzeros));

        // Build the block production input
        let input = BlockProductionInput {
            kind,
            sender: message.message.header.sender,
            l1_block_number: message.message.header.block_number,
            l1_timestamp: message.message.header.timestamp,
            request_id: message.message.header.request_id,
            l1_base_fee: message.message.header.base_fee_l1,
            l2_msg,
            delayed_messages_read: message.delayed_messages_read,
            batch_gas_cost: message.message.batch_gas_cost,
            batch_data_stats,
        };

        // Delegate to the block producer
        let result = self
            .block_producer
            .produce_block(msg_idx, input)
            .await
            .map_err(|e| internal_error(e.to_string()))?;

        Ok(RpcMessageResult {
            block_hash: result.block_hash,
            send_root: result.send_root,
        })
    }

    async fn reorg(
        &self,
        msg_idx_of_first_msg_to_add: u64,
        new_messages: Vec<RpcMessageWithMetadataAndBlockInfo>,
        _old_messages: Vec<RpcMessageWithMetadata>,
    ) -> RpcResult<Vec<RpcMessageResult>> {
        info!(
            target: "nitroexecution",
            msg_idx_of_first_msg_to_add,
            new_msgs = new_messages.len(),
            "reorg"
        );

        // Roll back to the last kept block = (first divergent msg) - 1.
        // Message i corresponds to block (genesis + i), so reset head to
        // genesis + (msg_idx_of_first_msg_to_add - 1). If the target is
        // before genesis, reset to genesis.
        let target_block = msg_idx_of_first_msg_to_add
            .saturating_sub(1)
            .saturating_add(self.genesis_block_num);

        self.block_producer
            .reset_to_block(target_block)
            .await
            .map_err(|e| internal_error(format!("reset_to_block: {e}")))?;

        // Replay new messages on top of the rolled-back head.
        let mut results = Vec::with_capacity(new_messages.len());
        for (i, wrapped) in new_messages.into_iter().enumerate() {
            let msg_idx = msg_idx_of_first_msg_to_add + i as u64;
            let meta = wrapped.message;
            let l2_msg = decode_l2_msg(&meta.message.l2_msg).map_err(internal_error)?;
            let batch_data_stats = meta
                .message
                .batch_data_tokens
                .as_ref()
                .map(|s| (s.length, s.nonzeros));
            let input = BlockProductionInput {
                kind: meta.message.header.kind,
                sender: meta.message.header.sender,
                l1_block_number: meta.message.header.block_number,
                l1_timestamp: meta.message.header.timestamp,
                request_id: meta.message.header.request_id,
                l1_base_fee: meta.message.header.base_fee_l1,
                l2_msg,
                delayed_messages_read: meta.delayed_messages_read,
                batch_gas_cost: meta.message.batch_gas_cost,
                batch_data_stats,
            };
            let produced = self
                .block_producer
                .produce_block(msg_idx, input)
                .await
                .map_err(|e| internal_error(format!("reorg replay msg {msg_idx}: {e}")))?;
            results.push(RpcMessageResult {
                block_hash: produced.block_hash,
                send_root: produced.send_root,
            });
        }
        Ok(results)
    }

    async fn head_message_index(&self) -> RpcResult<u64> {
        let best = self
            .provider
            .best_block_number()
            .map_err(|e| internal_error(e.to_string()))?;

        let msg_idx = self.block_number_to_message_index(best).unwrap_or(0);
        debug!(target: "nitroexecution", best, msg_idx, "headMessageIndex");
        Ok(msg_idx)
    }

    async fn result_at_message_index(&self, msg_idx: u64) -> RpcResult<RpcMessageResult> {
        let block_num = self.message_index_to_block_number(msg_idx);

        let header = self
            .get_header(block_num)
            .map_err(internal_error)?
            .ok_or_else(|| internal_error(format!("Block {block_num} not found")))?;

        let send_root = Self::send_root_from_header(header.header());

        Ok(RpcMessageResult {
            block_hash: header.hash(),
            send_root,
        })
    }

    fn set_finality_data(
        &self,
        safe: Option<RpcFinalityData>,
        finalized: Option<RpcFinalityData>,
        validated: Option<RpcFinalityData>,
    ) -> RpcResult<()> {
        debug!(target: "nitroexecution", ?safe, ?finalized, ?validated, "setFinalityData");
        self.block_producer
            .set_finality(
                safe.map(|f| f.block_hash),
                finalized.map(|f| f.block_hash),
                validated.map(|f| f.block_hash),
            )
            .map_err(|e| internal_error(format!("set_finality: {e}")))?;
        Ok(())
    }

    fn set_consensus_sync_data(&self, sync_data: RpcConsensusSyncData) -> RpcResult<()> {
        let mut state = self.state.write();
        state.synced = sync_data.synced;
        state.max_message_count = sync_data.max_message_count;
        debug!(target: "nitroexecution", synced = sync_data.synced, max = sync_data.max_message_count, "setConsensusSyncData");
        Ok(())
    }

    fn mark_feed_start(&self, to: u64) -> RpcResult<()> {
        debug!(target: "nitroexecution", to, "markFeedStart");
        Ok(())
    }

    async fn trigger_maintenance(&self) -> RpcResult<()> {
        Ok(())
    }

    async fn should_trigger_maintenance(&self) -> RpcResult<bool> {
        Ok(false)
    }

    async fn maintenance_status(&self) -> RpcResult<RpcMaintenanceStatus> {
        Ok(RpcMaintenanceStatus { is_running: false })
    }

    async fn arbos_version_for_message_index(&self, msg_idx: u64) -> RpcResult<u64> {
        let block_num = self.message_index_to_block_number(msg_idx);

        let header = self
            .get_header(block_num)
            .map_err(internal_error)?
            .ok_or_else(|| internal_error(format!("Block {block_num} not found")))?;

        let mix = header.header().mix_hash().unwrap_or_default();
        let arbos_version = u64::from_be_bytes(mix.0[16..24].try_into().unwrap_or_default());

        Ok(arbos_version)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use base64::engine::general_purpose::STANDARD as B64;

    #[test]
    fn decode_empty_option_is_ok() {
        assert_eq!(decode_l2_msg(&None).unwrap(), Vec::<u8>::new());
        assert_eq!(
            decode_l2_msg(&Some(String::new())).unwrap(),
            Vec::<u8>::new()
        );
    }

    #[test]
    fn decode_standard_padded() {
        let encoded = B64.encode(b"Hello, world!");
        let out = base64_decode(&encoded).unwrap();
        assert_eq!(out, b"Hello, world!");
    }

    #[test]
    fn decode_accepts_unpadded() {
        let encoded = B64.encode(b"Hello");
        let stripped = encoded.trim_end_matches('=').to_string();
        assert_eq!(base64_decode(&stripped).unwrap(), b"Hello");
    }

    #[test]
    fn decode_accepts_extra_padding() {
        assert_eq!(base64_decode("SGVsbG8==").unwrap(), b"Hello");
        assert_eq!(base64_decode("SGVsbG8====").unwrap(), b"Hello");
    }

    #[test]
    fn decode_rejects_invalid_character() {
        assert!(base64_decode("SG!X").is_err());
        assert!(base64_decode("a b").is_err());
        assert!(base64_decode("hello world").is_err());
    }

    #[test]
    fn decode_rejects_padding_in_body() {
        assert!(base64_decode("=SGVs").is_err());
        assert!(base64_decode("SGVs=bG8").is_err());
    }

    #[test]
    fn decode_large_payload_matches_roundtrip() {
        let bytes: Vec<u8> = (0..32 * 1024).map(|i| (i * 7 + 3) as u8).collect();
        let encoded = B64.encode(&bytes);
        assert_eq!(base64_decode(&encoded).unwrap(), bytes);
    }

    #[test]
    fn decode_preserves_lenient_padding_tail() {
        assert_eq!(base64_decode("S").unwrap(), Vec::<u8>::new());
        assert_eq!(base64_decode("SG").unwrap(), vec![b'H']);
        assert_eq!(base64_decode("SGV").unwrap(), vec![b'H', b'e']);
    }

    #[test]
    fn decode_length_one_tail_validates_alphabet() {
        assert!(base64_decode("ABCD!").is_err());
    }

    #[test]
    fn decode_all_alphabet_characters() {
        let out = base64_decode("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
            .unwrap();
        assert_eq!(out.len(), 48);
    }
}
