//! Arbitrum EthApi wrapper with L1 gas estimation.
//!
//! Wraps reth's [`EthApiInner`] to override gas estimation
//! with L1 posting cost awareness.

use std::{sync::Arc, time::Duration};

use alloy_primitives::{StorageKey, B256, U256};
use alloy_rpc_types_eth::{state::StateOverride, BlockId};
use reth_primitives_traits::{Recovered, WithEncoded};
use reth_rpc::eth::core::EthApiInner;
use reth_rpc_convert::{RpcConvert, RpcTxReq};
use reth_rpc_eth_api::{
    helpers::{
        estimate::EstimateCall, pending_block::PendingEnvBuilder, Call, EthApiSpec, EthBlocks,
        EthCall, EthFees, EthSigner, EthState, EthTransactions, LoadBlock, LoadFee,
        LoadPendingBlock, LoadReceipt, LoadState, LoadTransaction, SpawnBlocking, Trace,
    },
    EthApiTypes, FromEvmError, RpcNodeCore, RpcNodeCoreExt,
};
use reth_rpc_eth_types::{
    builder::config::PendingBlockKind, EthApiError, EthStateCache, FeeHistoryCache, GasPriceOracle,
    PendingBlock,
};
use reth_storage_api::{ProviderHeader, StateProviderFactory, TransactionsProvider};
use reth_tasks::{
    pool::{BlockingTaskGuard, BlockingTaskPool},
    Runtime,
};
use reth_transaction_pool::{
    AddedTransactionOutcome, PoolPooledTx, PoolTransaction, TransactionOrigin, TransactionPool,
};
use tracing::trace;

use arb_precompiles::storage_slot::{
    subspace_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
};

/// Type alias matching reth's `SignersForRpc`.
type SignersForRpc<Provider, Rpc> = parking_lot::RwLock<
    Vec<Box<dyn EthSigner<<Provider as TransactionsProvider>::Transaction, RpcTxReq<Rpc>>>>,
>;

/// L1 pricing field offset for price per unit.
const L1_PRICE_PER_UNIT: u64 = 7;

/// L2 pricing field offset for base fee.
const L2_BASE_FEE: u64 = 2;

/// Non-zero calldata gas cost per byte (EIP-2028).
const TX_DATA_NON_ZERO_GAS: u64 = 16;

/// Padding applied to L1 fee estimates (110% = 11000 bips).
const GAS_ESTIMATION_L1_PRICE_PADDING: u64 = 11000;

/// Arbitrum Eth API wrapping the standard reth EthApiInner.
///
/// This wrapper overrides gas estimation to add L1 posting costs.
pub struct ArbEthApi<N: RpcNodeCore, Rpc: RpcConvert> {
    inner: Arc<EthApiInner<N, Rpc>>,
}

impl<N: RpcNodeCore, Rpc: RpcConvert> Clone for ArbEthApi<N, Rpc> {
    fn clone(&self) -> Self {
        Self {
            inner: self.inner.clone(),
        }
    }
}

impl<N: RpcNodeCore, Rpc: RpcConvert> std::fmt::Debug for ArbEthApi<N, Rpc> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("ArbEthApi").finish_non_exhaustive()
    }
}

impl<N: RpcNodeCore, Rpc: RpcConvert> ArbEthApi<N, Rpc> {
    /// Create a new `ArbEthApi` wrapping the given inner.
    pub fn new(inner: EthApiInner<N, Rpc>) -> Self {
        Self {
            inner: Arc::new(inner),
        }
    }
}

impl<N, Rpc> ArbEthApi<N, Rpc>
where
    N: RpcNodeCore<Provider: StateProviderFactory>,
    Rpc: RpcConvert,
{
    /// Compute L1 posting gas for gas estimation.
    ///
    /// Reads L1 pricing state from ArbOS to estimate the gas needed to cover
    /// L1 data posting costs for the given calldata length.
    fn l1_posting_gas(&self, calldata_len: usize, at: BlockId) -> Result<u64, EthApiError> {
        if calldata_len == 0 {
            return Ok(0);
        }

        let state = self
            .inner
            .provider()
            .state_by_block_id(at)
            .map_err(|e| EthApiError::Internal(e.into()))?;

        let l1_price_slot = subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT);
        let l1_price = state
            .storage(
                ARBOS_STATE_ADDRESS,
                StorageKey::from(B256::from(l1_price_slot.to_be_bytes::<32>())),
            )
            .map_err(|e| EthApiError::Internal(e.into()))?
            .unwrap_or_default();

        let basefee_slot = subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE);
        let basefee = state
            .storage(
                ARBOS_STATE_ADDRESS,
                StorageKey::from(B256::from(basefee_slot.to_be_bytes::<32>())),
            )
            .map_err(|e| EthApiError::Internal(e.into()))?
            .unwrap_or_default();

        if l1_price.is_zero() || basefee.is_zero() {
            return Ok(0);
        }

        // L1 fee = l1_price * calldata_bytes * TX_DATA_NON_ZERO_GAS
        let l1_fee = l1_price
            .saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS))
            .saturating_mul(U256::from(calldata_len));

        // Apply 110% padding for L1 price volatility.
        let padded = l1_fee.saturating_mul(U256::from(GAS_ESTIMATION_L1_PRICE_PADDING))
            / U256::from(10000u64);

        // Use 7/8 of basefee as congestion discount for estimation.
        let adjusted_basefee = basefee.saturating_mul(U256::from(7)) / U256::from(8);
        let adjusted_basefee = if adjusted_basefee.is_zero() {
            U256::from(1)
        } else {
            adjusted_basefee
        };

        // Convert to gas units: posting_gas = padded_fee / adjusted_basefee
        let gas = padded / adjusted_basefee;
        Ok(gas.try_into().unwrap_or(u64::MAX))
    }
}

// ---- Trait delegations (matching reth's EthApi bounds exactly) ----

impl<N, Rpc> EthApiTypes for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Error = EthApiError>,
{
    type Error = EthApiError;
    type NetworkTypes = Rpc::Network;
    type RpcConvert = Rpc;

    fn converter(&self) -> &Self::RpcConvert {
        self.inner.converter()
    }
}

impl<N, Rpc> RpcNodeCore for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert,
{
    type Primitives = N::Primitives;
    type Provider = N::Provider;
    type Pool = N::Pool;
    type Evm = N::Evm;
    type Network = N::Network;

    #[inline]
    fn pool(&self) -> &Self::Pool {
        self.inner.pool()
    }

    #[inline]
    fn evm_config(&self) -> &Self::Evm {
        self.inner.evm_config()
    }

    #[inline]
    fn network(&self) -> &Self::Network {
        self.inner.network()
    }

    #[inline]
    fn provider(&self) -> &Self::Provider {
        self.inner.provider()
    }
}

impl<N, Rpc> RpcNodeCoreExt for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert,
{
    #[inline]
    fn cache(&self) -> &EthStateCache<N::Primitives> {
        self.inner.cache()
    }
}

impl<N, Rpc> EthApiSpec for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
    fn starting_block(&self) -> U256 {
        self.inner.starting_block()
    }
}

impl<N, Rpc> SpawnBlocking for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Error = EthApiError>,
{
    #[inline]
    fn io_task_spawner(&self) -> &Runtime {
        self.inner.task_spawner()
    }

    #[inline]
    fn tracing_task_pool(&self) -> &BlockingTaskPool {
        self.inner.blocking_task_pool()
    }

    #[inline]
    fn tracing_task_guard(&self) -> &BlockingTaskGuard {
        self.inner.blocking_task_guard()
    }

    #[inline]
    fn blocking_io_task_guard(&self) -> &Arc<tokio::sync::Semaphore> {
        self.inner.blocking_io_request_semaphore()
    }
}

impl<N, Rpc> LoadFee for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
    fn gas_oracle(&self) -> &GasPriceOracle<Self::Provider> {
        self.inner.gas_oracle()
    }

    fn fee_history_cache(&self) -> &FeeHistoryCache<ProviderHeader<N::Provider>> {
        self.inner.fee_history_cache()
    }
}

impl<N, Rpc> LoadState for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives>,
    Self: LoadPendingBlock,
{
}

impl<N, Rpc> EthState for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
    Self: LoadPendingBlock,
{
    fn max_proof_window(&self) -> u64 {
        self.inner.eth_proof_window()
    }
}

impl<N, Rpc> EthFees for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
}

impl<N, Rpc> Trace for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError, Evm = N::Evm>,
{
}

impl<N, Rpc> LoadPendingBlock for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
    fn pending_block(&self) -> &tokio::sync::Mutex<Option<PendingBlock<N::Primitives>>> {
        self.inner.pending_block()
    }

    fn pending_env_builder(&self) -> &dyn PendingEnvBuilder<N::Evm> {
        self.inner.pending_env_builder()
    }

    fn pending_block_kind(&self) -> PendingBlockKind {
        self.inner.pending_block_kind()
    }
}

impl<N, Rpc> LoadBlock for ArbEthApi<N, Rpc>
where
    Self: LoadPendingBlock,
    N: RpcNodeCore,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
}

impl<N, Rpc> LoadTransaction for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
}

impl<N, Rpc> EthBlocks for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
}

impl<N, Rpc> EthTransactions for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
    fn signers(&self) -> &SignersForRpc<Self::Provider, Self::NetworkTypes> {
        self.inner.signers()
    }

    fn send_raw_transaction_sync_timeout(&self) -> Duration {
        self.inner.send_raw_transaction_sync_timeout()
    }

    async fn send_transaction(
        &self,
        origin: TransactionOrigin,
        tx: WithEncoded<Recovered<PoolPooledTx<Self::Pool>>>,
    ) -> Result<B256, Self::Error> {
        let (_tx_bytes, recovered) = tx.split();
        let pool_transaction = <Self::Pool as TransactionPool>::Transaction::from_pooled(recovered);

        let AddedTransactionOutcome { hash, .. } = self
            .inner
            .add_pool_transaction(origin, pool_transaction)
            .await?;

        Ok(hash)
    }
}

impl<N, Rpc> LoadReceipt for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
{
}

// ---- Gas estimation override ----

impl<N, Rpc> Call for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError, Evm = N::Evm>,
{
    #[inline]
    fn call_gas_limit(&self) -> u64 {
        self.inner.gas_cap()
    }

    #[inline]
    fn max_simulate_blocks(&self) -> u64 {
        self.inner.max_simulate_blocks()
    }

    #[inline]
    fn evm_memory_limit(&self) -> u64 {
        self.inner.evm_memory_limit()
    }
}

impl<N, Rpc> EstimateCall for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError, Evm = N::Evm>,
{
    // Uses default binary search. L1 posting gas is added in EthCall below.
}

impl<N, Rpc> EthCall for ArbEthApi<N, Rpc>
where
    N: RpcNodeCore<Provider: StateProviderFactory>,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError, Evm = N::Evm>,
{
    /// Override gas estimation to add L1 posting costs.
    #[allow(clippy::manual_async_fn)]
    fn estimate_gas_at(
        &self,
        request: RpcTxReq<<Self::RpcConvert as RpcConvert>::Network>,
        at: BlockId,
        state_override: Option<StateOverride>,
    ) -> impl std::future::Future<Output = Result<U256, Self::Error>> + Send {
        async move {
            // Extract calldata length before request is consumed by the binary search.
            let calldata_len = request.as_ref().input.input().map(|b| b.len()).unwrap_or(0);

            // Run the standard binary search to find compute gas.
            let compute_gas =
                EstimateCall::estimate_gas_at(self, request, at, state_override).await?;

            // Add L1 posting gas.
            let l1_gas = self.l1_posting_gas(calldata_len, at)?;

            if l1_gas > 0 {
                trace!(target: "rpc::eth::estimate", %compute_gas, l1_gas, "Adding L1 posting gas to estimate");
            }

            Ok(compute_gas.saturating_add(U256::from(l1_gas)))
        }
    }
}
