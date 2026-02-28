//! Implementation of the `nitroexecution` RPC handler.
//!
//! Receives messages from the Nitro consensus layer, produces blocks,
//! and maintains the mapping between message indices and block numbers.

use std::sync::Arc;

use alloy_consensus::BlockHeader;
use alloy_primitives::B256;
use alloy_rpc_types_eth::BlockNumberOrTag;
use jsonrpsee::core::RpcResult;
use parking_lot::RwLock;
use reth_provider::{BlockNumReader, BlockReaderIdExt, HeaderProvider};
use tracing::{debug, info, warn};

use crate::nitro_execution::{
    NitroExecutionApiServer, RpcConsensusSyncData, RpcFinalityData, RpcMaintenanceStatus,
    RpcMessageResult, RpcMessageWithMetadata, RpcMessageWithMetadataAndBlockInfo,
};

/// Genesis block number for the chain (0 for Arbitrum Sepolia).
const GENESIS_BLOCK_NUM: u64 = 0;

/// State shared between the RPC handler and the node.
#[derive(Debug)]
pub struct NitroExecutionState {
    /// Whether the node is synced with consensus.
    pub synced: bool,
    /// Maximum message count from consensus.
    pub max_message_count: u64,
}

impl Default for NitroExecutionState {
    fn default() -> Self {
        Self {
            synced: false,
            max_message_count: 0,
        }
    }
}

/// Handler for the `nitroexecution` RPC namespace.
///
/// This is a read-only handler that responds to queries from Nitro consensus.
/// Block production is not yet implemented - this handler allows the node to
/// start up and respond to status queries while we develop the full
/// execution engine.
pub struct NitroExecutionHandler<Provider> {
    provider: Provider,
    state: Arc<RwLock<NitroExecutionState>>,
}

impl<Provider> NitroExecutionHandler<Provider> {
    /// Create a new handler.
    pub fn new(provider: Provider) -> Self {
        Self {
            provider,
            state: Arc::new(RwLock::new(NitroExecutionState::default())),
        }
    }

    /// Convert a message index to a block number.
    /// Matches Go: `MessageIndexToBlockNumber(msgIdx) = msgIdx + genesisBlockNum`
    fn message_index_to_block_number(msg_idx: u64) -> u64 {
        GENESIS_BLOCK_NUM + msg_idx
    }

    /// Convert a block number to a message index.
    /// Matches Go: `BlockNumberToMessageIndex(blockNum) = blockNum - genesisBlockNum`
    fn block_number_to_message_index(block_num: u64) -> Option<u64> {
        if block_num < GENESIS_BLOCK_NUM {
            return None;
        }
        Some(block_num - GENESIS_BLOCK_NUM)
    }
}

impl<Provider> NitroExecutionHandler<Provider>
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

#[async_trait::async_trait]
impl<Provider> NitroExecutionApiServer for NitroExecutionHandler<Provider>
where
    Provider: BlockNumReader + BlockReaderIdExt + HeaderProvider + 'static,
{
    async fn digest_message(
        &self,
        msg_idx: u64,
        _message: RpcMessageWithMetadata,
        _message_for_prefetch: Option<RpcMessageWithMetadata>,
    ) -> RpcResult<RpcMessageResult> {
        let block_num = Self::message_index_to_block_number(msg_idx);
        info!(target: "nitroexecution", msg_idx, block_num, "digestMessage called");

        // Check if we already have this block (idempotent)
        if let Some(header) = self.get_header(block_num).map_err(internal_error)? {
            let send_root = Self::send_root_from_header(header.header());
            debug!(target: "nitroexecution", block_num, ?send_root, "Block already exists");
            return Ok(RpcMessageResult {
                block_hash: header.hash(),
                send_root,
            });
        }

        // Block production not yet implemented - return error
        Err(internal_error(format!(
            "Block production not yet implemented for block {block_num}"
        )))
    }

    async fn reorg(
        &self,
        msg_idx_of_first_msg_to_add: u64,
        _new_messages: Vec<RpcMessageWithMetadataAndBlockInfo>,
        _old_messages: Vec<RpcMessageWithMetadata>,
    ) -> RpcResult<Vec<RpcMessageResult>> {
        warn!(target: "nitroexecution", msg_idx_of_first_msg_to_add, "Reorg not yet implemented");
        Err(internal_error("Reorg not yet implemented"))
    }

    async fn head_message_index(&self) -> RpcResult<u64> {
        let best = self
            .provider
            .best_block_number()
            .map_err(|e| internal_error(e.to_string()))?;

        let msg_idx = Self::block_number_to_message_index(best).unwrap_or(0);
        debug!(target: "nitroexecution", best, msg_idx, "headMessageIndex");
        Ok(msg_idx)
    }

    async fn result_at_message_index(&self, msg_idx: u64) -> RpcResult<RpcMessageResult> {
        let block_num = Self::message_index_to_block_number(msg_idx);

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
        let block_num = Self::message_index_to_block_number(msg_idx);

        let header = self
            .get_header(block_num)
            .map_err(internal_error)?
            .ok_or_else(|| internal_error(format!("Block {block_num} not found")))?;

        let mix = header.header().mix_hash().unwrap_or_default();
        let arbos_version = u64::from_be_bytes(mix.0[16..24].try_into().unwrap_or_default());

        Ok(arbos_version)
    }
}
