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

    /// Publisher health check. Errors when the node is not configured
    /// as a transaction publisher (the executor-only mode here).
    #[method(name = "checkPublisherHealth")]
    fn check_publisher_health(&self) -> RpcResult<()>;

    /// Returns block info for the given block number.
    #[method(name = "getBlockInfo")]
    async fn get_block_info(&self, block_num: u64) -> RpcResult<ArbBlockInfo>;

    /// Raw block metadata for blocks in `[fromBlock, toBlock]`. The
    /// `rawMetadata` field is empty since no sidecar data is tracked.
    #[method(name = "getRawBlockMetadata")]
    async fn get_raw_block_metadata(
        &self,
        from_block: BlockNumberOrTag,
        to_block: BlockNumberOrTag,
    ) -> RpcResult<Vec<NumberAndBlockMetadata>>;

    /// Stylus host-I/O trace for a previously-executed transaction.
    /// Empty if no Stylus contracts were invoked.
    #[method(name = "traceStylusHostio")]
    async fn trace_stylus_hostio(&self, tx_hash: B256) -> RpcResult<Vec<HostioTraceInfo>>;

    /// Currently-set validated block hash. Zero hash when unset.
    #[method(name = "getValidatedBlock")]
    fn get_validated_block(&self) -> RpcResult<B256>;

    /// L1 confirmations for the L2 block at `block_num`. Returns 0
    /// without an L1 reader.
    #[method(name = "getL1Confirmations")]
    async fn get_l1_confirmations(&self, block_num: u64) -> RpcResult<u64>;

    /// L1 batch number containing the L2 block at `block_num`. Errors
    /// without an L1 batch index.
    #[method(name = "findBatchContainingBlock")]
    async fn find_batch_containing_block(&self, block_num: u64) -> RpcResult<u64>;
}

pub struct ArbApiHandler<Provider> {
    provider: Provider,
    maintenance_status: Arc<parking_lot::RwLock<ArbMaintenanceStatus>>,
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
        Err(jsonrpsee::types::ErrorObject::owned(
            jsonrpsee::types::error::INTERNAL_ERROR_CODE,
            "publishing transactions not supported by this endpoint",
            None::<()>,
        ))
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

    async fn get_l1_confirmations(&self, _block_num: u64) -> RpcResult<u64> {
        Ok(0)
    }

    async fn find_batch_containing_block(&self, block_num: u64) -> RpcResult<u64> {
        Err(jsonrpsee::types::ErrorObject::owned(
            jsonrpsee::types::error::INTERNAL_ERROR_CODE,
            format!("no batch index available for block {block_num}"),
            None::<()>,
        ))
    }

    async fn get_raw_block_metadata(
        &self,
        from_block: BlockNumberOrTag,
        to_block: BlockNumberOrTag,
    ) -> RpcResult<Vec<NumberAndBlockMetadata>> {
        let internal = |e: alloy_rpc_types_eth::ConversionError| {
            jsonrpsee::types::ErrorObject::owned(
                jsonrpsee::types::error::INVALID_PARAMS_CODE,
                e.to_string(),
                None::<()>,
            )
        };
        let provider_err = |e: reth_provider::ProviderError| {
            jsonrpsee::types::ErrorObject::owned(
                jsonrpsee::types::error::INTERNAL_ERROR_CODE,
                e.to_string(),
                None::<()>,
            )
        };
        let best = self.provider.best_block_number().map_err(provider_err)?;
        let resolve = |bn: BlockNumberOrTag| -> Result<u64, _> {
            match bn {
                BlockNumberOrTag::Number(n) => Ok(n),
                BlockNumberOrTag::Earliest => Ok(0),
                BlockNumberOrTag::Latest
                | BlockNumberOrTag::Safe
                | BlockNumberOrTag::Finalized
                | BlockNumberOrTag::Pending => Ok(best),
            }
            .map_err(internal)
        };
        let from = resolve(from_block)?;
        let to = resolve(to_block)?;
        if from > to {
            return Err(jsonrpsee::types::ErrorObject::owned(
                jsonrpsee::types::error::INVALID_PARAMS_CODE,
                format!("from_block ({from}) > to_block ({to})"),
                None::<()>,
            ));
        }
        const MAX_RANGE: u64 = 5_000;
        if to.saturating_sub(from) + 1 > MAX_RANGE {
            return Err(jsonrpsee::types::ErrorObject::owned(
                jsonrpsee::types::error::INVALID_PARAMS_CODE,
                format!("range exceeds {MAX_RANGE} blocks"),
                None::<()>,
            ));
        }
        // No sidecar metadata is tracked — return an empty list so the
        // wire shape matches a node with no metadata stored.
        let _ = (from, to, best);
        Ok(Vec::new())
    }
}
