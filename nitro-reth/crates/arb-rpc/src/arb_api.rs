//! Arbitrum `arb_` RPC namespace.
//!
//! Implements the `arb_` JSON-RPC methods: maintenance status,
//! health checks, and block metadata queries.

use std::sync::Arc;

use alloy_consensus::BlockHeader;
use alloy_primitives::B256;
use alloy_rpc_types_eth::BlockNumberOrTag;
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use reth_provider::{BlockNumReader, BlockReaderIdExt, HeaderProvider};

use crate::{
    stylus_tracer::HostioTraceInfo, ArbBlockInfo, ArbMaintenanceStatus, NumberAndBlockMetadata,
};

/// Arbitrum `arb_` RPC namespace.
#[rpc(server, namespace = "arb")]
pub trait ArbApi {
    /// Returns the maintenance status of the node.
    #[method(name = "maintenanceStatus")]
    fn maintenance_status(&self) -> RpcResult<ArbMaintenanceStatus>;

    /// Checks publisher health. Returns an error if unhealthy.
    #[method(name = "checkPublisherHealth")]
    fn check_publisher_health(&self) -> RpcResult<()>;

    /// Returns block info for the given block number.
    #[method(name = "getBlockInfo")]
    async fn get_block_info(&self, block_num: u64) -> RpcResult<ArbBlockInfo>;

    /// Returns the raw block metadata for blocks in `[fromBlock, toBlock]`.
    ///
    /// Used by Nitro consensus layer for bulk metadata sync. arbreth does
    /// not persist separate block metadata; each entry's `rawMetadata` is
    /// empty, matching the Nitro schema for blocks with no sidecar data.
    #[method(name = "getRawBlockMetadata")]
    async fn get_raw_block_metadata(
        &self,
        from_block: u64,
        to_block: u64,
    ) -> RpcResult<Vec<NumberAndBlockMetadata>>;

    /// Returns the Stylus host-I/O trace for a previously-executed
    /// transaction. Empty if the tx didn't invoke any Stylus contracts
    /// or hasn't been re-traced yet.
    ///
    /// Pairs with `debug_traceTransaction` — clients call both and
    /// merge the EVM opcode trace with the Stylus host calls.
    #[method(name = "traceStylusHostio")]
    async fn trace_stylus_hostio(&self, tx_hash: B256) -> RpcResult<Vec<HostioTraceInfo>>;

    /// Returns the currently-set validated block hash, as propagated
    /// via `nitroexecution_setFinalityData`. Returns the zero hash
    /// when no validated marker is set.
    #[method(name = "getValidatedBlock")]
    fn get_validated_block(&self) -> RpcResult<B256>;
}

/// Implementation of the `arb_` RPC namespace.
pub struct ArbApiHandler<Provider> {
    provider: Provider,
    /// Current maintenance mode status.
    maintenance_status: Arc<parking_lot::RwLock<ArbMaintenanceStatus>>,
    /// Current validated block hash set by `nitroexecution_setFinalityData`.
    validated_block: Arc<parking_lot::RwLock<B256>>,
}

impl<Provider> ArbApiHandler<Provider> {
    /// Create a new `ArbApiHandler`.
    pub fn new(provider: Provider) -> Self {
        Self {
            provider,
            maintenance_status: Arc::new(parking_lot::RwLock::new(ArbMaintenanceStatus::default())),
            validated_block: Arc::new(parking_lot::RwLock::new(B256::ZERO)),
        }
    }

    /// Returns a reference to the maintenance status lock for external updates.
    pub fn maintenance_status_handle(&self) -> Arc<parking_lot::RwLock<ArbMaintenanceStatus>> {
        self.maintenance_status.clone()
    }

    /// Returns a shared handle the block producer uses to push
    /// validated-block updates from `setFinalityData`.
    pub fn validated_block_handle(&self) -> Arc<parking_lot::RwLock<B256>> {
        self.validated_block.clone()
    }
}

#[async_trait::async_trait]
impl<Provider> ArbApiServer for ArbApiHandler<Provider>
where
    Provider: BlockNumReader + BlockReaderIdExt + HeaderProvider + 'static,
{
    fn maintenance_status(&self) -> RpcResult<ArbMaintenanceStatus> {
        Ok(self.maintenance_status.read().clone())
    }

    fn check_publisher_health(&self) -> RpcResult<()> {
        // Sequencer health is always OK for a full node.
        Ok(())
    }

    async fn get_block_info(&self, block_num: u64) -> RpcResult<ArbBlockInfo> {
        let header = self
            .provider
            .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(block_num))
            .map_err(|e| {
                jsonrpsee::types::ErrorObject::owned(
                    jsonrpsee::types::error::INTERNAL_ERROR_CODE,
                    e.to_string(),
                    None::<()>,
                )
            })?
            .ok_or_else(|| {
                jsonrpsee::types::ErrorObject::owned(
                    jsonrpsee::types::error::INVALID_PARAMS_CODE,
                    format!("block {block_num} not found"),
                    None::<()>,
                )
            })?;

        let mix = header.mix_hash().unwrap_or_default();
        let extra = header.extra_data();

        let send_count = u64::from_be_bytes(mix.0[0..8].try_into().unwrap_or_default());
        let l1_block_number = u64::from_be_bytes(mix.0[8..16].try_into().unwrap_or_default());
        let arbos_format_version = u64::from_be_bytes(mix.0[16..24].try_into().unwrap_or_default());

        let send_root = if extra.len() >= 32 {
            B256::from_slice(&extra[..32])
        } else {
            B256::ZERO
        };

        Ok(ArbBlockInfo {
            l1_block_number,
            arbos_format_version,
            send_count,
            send_root,
        })
    }

    async fn trace_stylus_hostio(
        &self,
        tx_hash: B256,
    ) -> RpcResult<Vec<crate::stylus_tracer::HostioTraceInfo>> {
        Ok(crate::stylus_tracer::take_cached_trace(tx_hash))
    }

    fn get_validated_block(&self) -> RpcResult<B256> {
        Ok(*self.validated_block.read())
    }

    async fn get_raw_block_metadata(
        &self,
        from_block: u64,
        to_block: u64,
    ) -> RpcResult<Vec<NumberAndBlockMetadata>> {
        if from_block > to_block {
            return Err(jsonrpsee::types::ErrorObject::owned(
                jsonrpsee::types::error::INVALID_PARAMS_CODE,
                format!("from_block ({from_block}) > to_block ({to_block})"),
                None::<()>,
            ));
        }
        // Cap the range to protect the node from huge scans. Matches a
        // reasonable upper bound while still serving typical sync queries.
        const MAX_RANGE: u64 = 5_000;
        if to_block.saturating_sub(from_block) + 1 > MAX_RANGE {
            return Err(jsonrpsee::types::ErrorObject::owned(
                jsonrpsee::types::error::INVALID_PARAMS_CODE,
                format!("range exceeds {MAX_RANGE} blocks"),
                None::<()>,
            ));
        }
        let best = self.provider.best_block_number().map_err(|e| {
            jsonrpsee::types::ErrorObject::owned(
                jsonrpsee::types::error::INTERNAL_ERROR_CODE,
                e.to_string(),
                None::<()>,
            )
        })?;
        let clamped_to = to_block.min(best);
        let mut out =
            Vec::with_capacity((clamped_to.saturating_sub(from_block) + 1).min(MAX_RANGE) as usize);
        for block_number in from_block..=clamped_to {
            // Block must exist — verify via header lookup; else skip.
            let maybe = self
                .provider
                .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(block_number))
                .map_err(|e| {
                    jsonrpsee::types::ErrorObject::owned(
                        jsonrpsee::types::error::INTERNAL_ERROR_CODE,
                        e.to_string(),
                        None::<()>,
                    )
                })?;
            if maybe.is_some() {
                out.push(NumberAndBlockMetadata {
                    block_number,
                    raw_metadata: alloy_primitives::Bytes::new(),
                });
            }
        }
        Ok(out)
    }
}
