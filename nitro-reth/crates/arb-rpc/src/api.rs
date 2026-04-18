//! Arbitrum EthApi wrapper with L1 gas estimation.
//!
//! Wraps reth's [`EthApiInner`] to override gas estimation
//! with L1 posting cost awareness.

use std::{sync::Arc, time::Duration};

use alloy_primitives::{Address, StorageKey, B256, U256};
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

impl<N, Rpc> ArbEthApi<N, Rpc>
where
    N: RpcNodeCore<Provider: StateProviderFactory>,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError, Evm = N::Evm>,
    RpcTxReq<<Rpc as RpcConvert>::Network>: AsRef<alloy_rpc_types_eth::TransactionRequest>
        + AsMut<alloy_rpc_types_eth::TransactionRequest>
        + Clone
        + Default,
{
    /// Gas estimate for `NodeInterface.estimateRetryableTicket` —
    /// `submit_intrinsic + auto_redeem_gas`.
    async fn estimate_retryable_ticket_gas(
        &self,
        input: &alloy_primitives::Bytes,
        at: BlockId,
        state_override: Option<StateOverride>,
    ) -> Result<U256, EthApiError>
    where
        RpcTxReq<<Rpc as RpcConvert>::Network>: From<alloy_rpc_types_eth::TransactionRequest>,
    {
        use alloy_primitives::{Bytes, TxKind};
        use alloy_rpc_types_eth::TransactionRequest;

        // ABI decode: selector(4) + 7 heads(32 each) + bytes tail.
        // sender, deposit, to, l2CallValue, excessFeeRefundAddr,
        // callValueRefundAddr, <bytes data offset>.
        const HEAD_LEN: usize = 4 + 32 * 7;
        if input.len() < HEAD_LEN {
            return Err(EthApiError::InvalidParams(
                "estimateRetryableTicket: calldata too short".into(),
            ));
        }
        let sender = Address::from_slice(&input[4 + 12..4 + 32]);
        let _deposit = U256::from_be_slice(&input[36..68]);
        let to_word = &input[68..100];
        let to = Address::from_slice(&to_word[12..32]);
        let l2_call_value = U256::from_be_slice(&input[100..132]);
        let _excess_fee_refund = Address::from_slice(&input[132 + 12..132 + 32]);
        let _call_value_refund = Address::from_slice(&input[164 + 12..164 + 32]);
        let data_offset: usize =
            U256::from_be_slice(&input[196..228])
                .try_into()
                .map_err(|_| {
                    EthApiError::InvalidParams(
                        "estimateRetryableTicket: invalid data offset".into(),
                    )
                })?;
        let abi_body = &input[4..];
        let data: Bytes = if data_offset + 32 <= abi_body.len() {
            let len: usize = U256::from_be_slice(&abi_body[data_offset..data_offset + 32])
                .try_into()
                .map_err(|_| {
                    EthApiError::InvalidParams(
                        "estimateRetryableTicket: data length too large".into(),
                    )
                })?;
            if data_offset + 32 + len > abi_body.len() {
                return Err(EthApiError::InvalidParams(
                    "estimateRetryableTicket: data out of bounds".into(),
                ));
            }
            Bytes::copy_from_slice(&abi_body[data_offset + 32..data_offset + 32 + len])
        } else {
            Bytes::new()
        };

        // `to == zero` means the retryable is a contract-create — map
        // that to TxKind::Create for the estimate request.
        let kind = if to == Address::ZERO {
            TxKind::Create
        } else {
            TxKind::Call(to)
        };
        let data_ref = data.clone();
        let equivalent = TransactionRequest {
            from: Some(sender),
            to: Some(kind),
            value: Some(l2_call_value),
            input: data.into(),
            ..Default::default()
        };
        let equivalent_req: RpcTxReq<<Rpc as RpcConvert>::Network> = equivalent.into();

        // Binary-search the auto-redeem gas via the standard eth
        // estimation machinery. The equivalent call has the exact
        // same state transitions as what the auto-redeem runs, so
        // its gas result is the auto-redeem's gas 1:1.
        let redeem_gas =
            EstimateCall::estimate_gas_at(self, equivalent_req, at, state_override).await?;

        // Submit-retryable intrinsic gas matches the default
        // IntrinsicGas for ArbitrumSubmitRetryableTx: 21,000 tx base +
        // EIP-2028 calldata (16 × non-zero + 4 × zero). ArbOS's
        // state-transition overhead for retryable creation, escrow,
        // and auto-redeem scheduling is charged inside the auto-redeem
        // itself and is therefore already captured by `redeem_gas`.
        let (zeros, non_zeros) =
            data_ref.iter().fold(
                (0u64, 0u64),
                |(z, nz), &b| if b == 0 { (z + 1, nz) } else { (z, nz + 1) },
            );
        let calldata_gas = zeros
            .saturating_mul(4)
            .saturating_add(non_zeros.saturating_mul(16));
        let submit_intrinsic = 21_000u64.saturating_add(calldata_gas);

        Ok(redeem_gas.saturating_add(U256::from(submit_intrinsic)))
    }

    /// `eth_call` of `NodeInterface.estimateRetryableTicket(...)` —
    /// simulates the auto-redeem inner call (`sender → to` with
    /// `l2CallValue` and `data`) and returns its raw bytes.
    async fn simulate_retryable_ticket_call(
        &self,
        input: &alloy_primitives::Bytes,
        at: BlockId,
        overrides: alloy_rpc_types_eth::state::EvmOverrides,
    ) -> Result<alloy_primitives::Bytes, EthApiError>
    where
        RpcTxReq<<Rpc as RpcConvert>::Network>: From<alloy_rpc_types_eth::TransactionRequest>,
    {
        use alloy_primitives::{Bytes, TxKind};
        use alloy_rpc_types_eth::TransactionRequest;

        const HEAD_LEN: usize = 4 + 32 * 7;
        if input.len() < HEAD_LEN {
            return Err(EthApiError::InvalidParams(
                "estimateRetryableTicket: calldata too short".into(),
            ));
        }
        let sender = Address::from_slice(&input[4 + 12..4 + 32]);
        let to_word = &input[68..100];
        let to = Address::from_slice(&to_word[12..32]);
        let l2_call_value = U256::from_be_slice(&input[100..132]);
        let data_offset: usize =
            U256::from_be_slice(&input[196..228])
                .try_into()
                .map_err(|_| {
                    EthApiError::InvalidParams(
                        "estimateRetryableTicket: invalid data offset".into(),
                    )
                })?;
        let abi_body = &input[4..];
        let data: Bytes = if data_offset + 32 <= abi_body.len() {
            let len: usize = U256::from_be_slice(&abi_body[data_offset..data_offset + 32])
                .try_into()
                .map_err(|_| {
                    EthApiError::InvalidParams(
                        "estimateRetryableTicket: data length too large".into(),
                    )
                })?;
            if data_offset + 32 + len > abi_body.len() {
                return Err(EthApiError::InvalidParams(
                    "estimateRetryableTicket: data out of bounds".into(),
                ));
            }
            Bytes::copy_from_slice(&abi_body[data_offset + 32..data_offset + 32 + len])
        } else {
            Bytes::new()
        };

        let kind = if to == Address::ZERO {
            TxKind::Create
        } else {
            TxKind::Call(to)
        };
        let equivalent = TransactionRequest {
            from: Some(sender),
            to: Some(kind),
            value: Some(l2_call_value),
            input: data.into(),
            ..Default::default()
        };
        let equivalent_req: RpcTxReq<<Rpc as RpcConvert>::Network> = equivalent.into();

        let _permit = self.acquire_owned_blocking_io().await;
        let res = self.transact_call_at(equivalent_req, at, overrides).await?;
        <EthApiError as reth_rpc_eth_types::error::api::FromEvmError<N::Evm>>::ensure_success(
            res.result,
        )
    }

    /// Handle `eth_call` dispatch of
    /// `NodeInterfaceDebug.getRetryable(bytes32 ticketId)` — reads the
    /// retryable record from storage and returns the 7-tuple
    /// `(timeout, from, to, value, beneficiary, tries, data)`.
    async fn get_retryable_abi(
        &self,
        input: &alloy_primitives::Bytes,
        at: BlockId,
    ) -> Result<alloy_primitives::Bytes, EthApiError> {
        use arb_precompiles::storage_slot::{derive_subspace_key, map_slot, ROOT_STORAGE_KEY};
        use arbos::retryables::{
            BENEFICIARY_OFFSET, CALLDATA_KEY, CALLVALUE_OFFSET, FROM_OFFSET, NUM_TRIES_OFFSET,
            TIMEOUT_OFFSET, TO_OFFSET,
        };

        if input.len() < 4 + 32 {
            return Err(EthApiError::InvalidParams(
                "getRetryable: expected bytes32 ticket".into(),
            ));
        }
        let ticket = B256::from_slice(&input[4..36]);

        let state = self
            .inner
            .provider()
            .state_by_block_id(at)
            .map_err(|e| EthApiError::Internal(e.into()))?;
        let load = |slot: U256| -> Result<U256, EthApiError> {
            let k = StorageKey::from(B256::from(slot.to_be_bytes::<32>()));
            Ok(state
                .storage(ARBOS_STATE_ADDRESS, k)
                .map_err(|e| EthApiError::Internal(e.into()))?
                .unwrap_or(U256::ZERO))
        };

        let retryables_key = derive_subspace_key(
            ROOT_STORAGE_KEY,
            arb_precompiles::storage_slot::RETRYABLES_SUBSPACE,
        );
        let r_key = derive_subspace_key(retryables_key.as_slice(), ticket.as_slice());

        let timeout: u64 = load(map_slot(r_key.as_slice(), TIMEOUT_OFFSET))?
            .try_into()
            .unwrap_or(0);
        if timeout == 0 {
            return Err(EthApiError::InvalidParams(format!(
                "no retryable with id 0x{ticket:x}"
            )));
        }
        let from_word = load(map_slot(r_key.as_slice(), FROM_OFFSET))?;
        let from = Address::from_slice(&from_word.to_be_bytes::<32>()[12..]);
        let to_word = load(map_slot(r_key.as_slice(), TO_OFFSET))?;
        let to_bytes: [u8; 32] = to_word.to_be_bytes();
        // StorageBackedAddressOrNil uses all-ones in the high 12 bytes
        // to encode Nil; actual encoding varies, so just take low 20
        // bytes and let callers treat zero as nil.
        let to = Address::from_slice(&to_bytes[12..]);
        let value = load(map_slot(r_key.as_slice(), CALLVALUE_OFFSET))?;
        let beneficiary_word = load(map_slot(r_key.as_slice(), BENEFICIARY_OFFSET))?;
        let beneficiary = Address::from_slice(&beneficiary_word.to_be_bytes::<32>()[12..]);
        let tries: u64 = load(map_slot(r_key.as_slice(), NUM_TRIES_OFFSET))?
            .try_into()
            .unwrap_or(0);

        // Calldata lives under its own subspace with StorageBackedBytes
        // layout: slot 0 = size, slot 1+ = chunks. We read the size,
        // then each 32-byte chunk, and truncate.
        let cd_key = derive_subspace_key(r_key.as_slice(), CALLDATA_KEY);
        let size: usize = load(map_slot(cd_key.as_slice(), 0))?
            .try_into()
            .unwrap_or(0);
        let chunks = size.div_ceil(32);
        let mut data = Vec::with_capacity(size);
        for i in 0..chunks {
            let chunk = load(map_slot(cd_key.as_slice(), 1 + i as u64))?;
            data.extend_from_slice(&chunk.to_be_bytes::<32>());
        }
        data.truncate(size);

        // ABI-encode the 7-tuple:
        //   head (7 × 32):
        //     timeout, from, to, value, beneficiary, tries, data_offset
        //   tail: data_len, data_bytes (padded to 32)
        let mut out = vec![0u8; 7 * 32];
        U256::from(timeout)
            .to_be_bytes::<32>()
            .iter()
            .enumerate()
            .for_each(|(i, b)| out[i] = *b);
        out[32 + 12..32 + 32].copy_from_slice(from.as_slice());
        out[64 + 12..64 + 32].copy_from_slice(to.as_slice());
        out[96..128].copy_from_slice(&value.to_be_bytes::<32>());
        out[128 + 12..128 + 32].copy_from_slice(beneficiary.as_slice());
        U256::from(tries)
            .to_be_bytes::<32>()
            .iter()
            .enumerate()
            .for_each(|(i, b)| out[160 + i] = *b);
        // data offset = 0xe0 (7 × 32).
        U256::from(7u64 * 32)
            .to_be_bytes::<32>()
            .iter()
            .enumerate()
            .for_each(|(i, b)| out[192 + i] = *b);
        // Tail.
        let padded_len = size.div_ceil(32) * 32;
        let mut tail = vec![0u8; 32 + padded_len];
        U256::from(size as u64)
            .to_be_bytes::<32>()
            .iter()
            .enumerate()
            .for_each(|(i, b)| tail[i] = *b);
        tail[32..32 + size].copy_from_slice(&data);
        out.extend_from_slice(&tail);
        Ok(alloy_primitives::Bytes::from(out))
    }

    /// Handle `eth_call` dispatch of
    /// `NodeInterface.constructOutboxProof(size, leaf)`. Scans ArbSys
    /// (0x64) L2ToL1Tx / SendMerkleUpdate event logs over the chain up
    /// to `at` to resolve every node hash the proof walk needs, then
    /// feeds the map to `outbox_proof::finalize_proof`.
    async fn construct_outbox_proof(
        &self,
        input: &alloy_primitives::Bytes,
        at: BlockId,
    ) -> Result<alloy_primitives::Bytes, EthApiError>
    where
        N: RpcNodeCore<
            Provider: reth_provider::BlockReaderIdExt + reth_storage_api::ReceiptProvider,
        >,
    {
        use std::collections::HashMap;

        use alloy_consensus::TxReceipt;
        use arb_precompiles::arbsys::{
            l2_to_l1_tx_topic, send_merkle_update_topic, ARBSYS_ADDRESS,
        };
        use reth_provider::{BlockNumReader, ReceiptProvider};

        use crate::outbox_proof::{encode_outbox_proof, finalize_proof, plan_proof, LevelAndLeaf};

        if input.len() < 4 + 64 {
            return Err(EthApiError::InvalidParams(
                "constructOutboxProof: expected (uint64 size, uint64 leaf)".into(),
            ));
        }
        let size: u64 = U256::from_be_slice(&input[4..36])
            .try_into()
            .unwrap_or(u64::MAX);
        let leaf: u64 = U256::from_be_slice(&input[36..68])
            .try_into()
            .unwrap_or(u64::MAX);

        let plan = plan_proof(size, leaf).ok_or_else(|| {
            EthApiError::InvalidParams(format!("constructOutboxProof: leaf {leaf} ≥ size {size}"))
        })?;

        // Resolve `at` to a concrete block number upper-bound. If
        // `latest` or missing, use the chain tip.
        let provider = self.inner.provider();
        let tip = provider
            .best_block_number()
            .map_err(|e| EthApiError::Internal(e.into()))?;
        let upper = match at {
            BlockId::Number(alloy_rpc_types_eth::BlockNumberOrTag::Number(n)) => n.min(tip),
            _ => tip,
        };

        // Scan receipts over [0..=upper] for ArbSys merkle + L2ToL1Tx
        // logs. Topic layout for both events: topic[3] = position (a
        // LevelAndLeaf packed as uint256), topic[1..3] carry the hash
        // depending on which event variant.
        let merkle_topic = send_merkle_update_topic();
        let l2tol1_topic = l2_to_l1_tx_topic();

        // Position → hash map. Keyed by the 32-byte position bytes.
        let mut positions: HashMap<[u8; 32], B256> = HashMap::new();

        let receipts_per_block = provider
            .receipts_by_block_range(0..=upper)
            .map_err(|e| EthApiError::Internal(e.into()))?;

        for block_receipts in receipts_per_block {
            for receipt in block_receipts {
                for log in receipt.logs() {
                    if log.address != ARBSYS_ADDRESS {
                        continue;
                    }
                    let topics = log.data.topics();
                    if topics.len() < 4 {
                        continue;
                    }
                    let kind = topics[0];
                    let is_merkle = kind == merkle_topic;
                    let is_l2tol1 = kind == l2tol1_topic;
                    if !is_merkle && !is_l2tol1 {
                        continue;
                    }
                    // position encoded in topic[3]; hash in topic[2]
                    // for both events (hash is an indexed arg).
                    let pos: [u8; 32] = topics[3].0;
                    let hash: B256 = topics[2];
                    positions.insert(pos, hash);
                }
            }
        }

        let lookup = |p: LevelAndLeaf| -> Option<B256> {
            let topic = p.as_topic();
            positions.get(&topic.0).copied()
        };

        let (send, root, proof) = finalize_proof(&plan, leaf, lookup)
            .map_err(|e| EthApiError::InvalidParams(format!("constructOutboxProof: {e}")))?;

        Ok(encode_outbox_proof(send, root, &proof))
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
    /// `eth_gasPrice` returns just the latest base fee — there is no
    /// priority-fee market on this chain.
    fn gas_price(&self) -> impl std::future::Future<Output = Result<U256, Self::Error>> + Send
    where
        Self: reth_rpc_eth_api::helpers::LoadBlock,
    {
        use alloy_consensus::BlockHeader;
        use reth_storage_api::{BlockNumReader, HeaderProvider};
        async move {
            let best = self
                .provider()
                .best_block_number()
                .map_err(|e| EthApiError::Internal(e.into()))?;
            let header_opt = HeaderProvider::sealed_header(self.provider(), best)
                .map_err(|e| EthApiError::Internal(e.into()))?;
            let base_fee = match header_opt {
                Some(sealed) => sealed.header().base_fee_per_gas().unwrap_or_default(),
                None => 0,
            };
            Ok(U256::from(base_fee))
        }
    }

    #[allow(clippy::manual_async_fn)]
    fn suggested_priority_fee(
        &self,
    ) -> impl std::future::Future<Output = Result<U256, Self::Error>> + Send
    where
        Self: 'static,
    {
        async move { Ok(U256::ZERO) }
    }
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
    N: RpcNodeCore<Primitives = arb_primitives::ArbPrimitives>,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError>,
    Self::Error: reth_rpc_eth_types::error::FromEthApiError,
{
    /// Override to use `convert_receipts_with_block` so every single-tx
    /// receipt fetch (e.g. `eth_getTransactionReceipt`) includes the
    /// Arbitrum `l1BlockNumber` field sourced from the block's mix_hash.
    ///
    /// Reth's default impl uses `convert_receipts` (no-block path), which
    /// our `ArbReceiptConverter` populates with `l1_block_number = None`.
    /// That breaks Arbitrum spec (bridges, indexers, explorers all expect
    /// `l1BlockNumber` on every receipt).
    fn build_transaction_receipt(
        &self,
        tx: reth_storage_api::ProviderTx<Self::Provider>,
        meta: alloy_consensus::transaction::TransactionMeta,
        receipt: reth_storage_api::ProviderReceipt<Self::Provider>,
    ) -> impl std::future::Future<
        Output = Result<reth_rpc_eth_api::RpcReceipt<Self::NetworkTypes>, Self::Error>,
    > + Send {
        use alloy_consensus::TxReceipt;
        use reth_primitives_traits::SignerRecoverable;
        use reth_rpc_convert::transaction::ConvertReceiptInput;
        use reth_rpc_eth_api::RpcNodeCoreExt;
        use reth_rpc_eth_types::{
            error::FromEthApiError, utils::calculate_gas_used_and_next_log_index, EthApiError,
        };
        async move {
            let hash = meta.block_hash;
            let all_receipts = self
                .cache()
                .get_receipts(hash)
                .await
                .map_err(<Self::Error as FromEthApiError>::from_eth_err)?
                .ok_or_else(|| {
                    <Self::Error as FromEthApiError>::from_eth_err(EthApiError::HeaderNotFound(
                        hash.into(),
                    ))
                })?;

            let (gas_used, next_log_index) =
                calculate_gas_used_and_next_log_index(meta.index, &all_receipts);

            let block = self
                .cache()
                .get_recovered_block(hash)
                .await
                .map_err(<Self::Error as FromEthApiError>::from_eth_err)?;

            let tx_recovered = tx
                .try_into_recovered_unchecked()
                .map_err(<Self::Error as FromEthApiError>::from_eth_err)?;

            let input = ConvertReceiptInput {
                tx: tx_recovered.as_recovered_ref(),
                gas_used: receipt.cumulative_gas_used() - gas_used,
                receipt,
                next_log_index,
                meta,
            };

            let result = match block {
                Some(sealed_block_with_senders) => self.converter().convert_receipts_with_block(
                    vec![input],
                    sealed_block_with_senders.sealed_block(),
                )?,
                None => self.converter().convert_receipts(vec![input])?,
            };
            Ok(result.into_iter().next().expect("one receipt in, one out"))
        }
    }
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
    N: RpcNodeCore<
        Provider: StateProviderFactory + reth_provider::BlockReaderIdExt + Clone,
        Primitives = arb_primitives::ArbPrimitives,
    >,
    EthApiError: FromEvmError<N::Evm>,
    Rpc: RpcConvert<Primitives = N::Primitives, Error = EthApiError, Evm = N::Evm>,
    RpcTxReq<<Rpc as RpcConvert>::Network>: AsRef<alloy_rpc_types_eth::TransactionRequest>
        + AsMut<alloy_rpc_types_eth::TransactionRequest>
        + Clone
        + Default
        + From<alloy_rpc_types_eth::TransactionRequest>,
{
    /// Override gas estimation to add L1 posting costs.
    ///
    /// Also intercepts `estimateRetryableTicket` calls to the
    /// NodeInterface (0xc8): client calls
    /// `eth_estimateGas({to:0xc8, data: estimateRetryableTicket(...)})`
    /// and expects back the gas for the retryable submission. We parse
    /// the ABI args, build an equivalent transaction request targeting
    /// the retry_to with retry_value + retry_data, run the standard
    /// estimation on that, and add the submit-retryable overhead.
    #[allow(clippy::manual_async_fn)]
    fn estimate_gas_at(
        &self,
        request: RpcTxReq<<Self::RpcConvert as RpcConvert>::Network>,
        at: BlockId,
        state_override: Option<StateOverride>,
    ) -> impl std::future::Future<Output = Result<U256, Self::Error>> + Send {
        async move {
            use crate::nodeinterface_rpc::NODE_INTERFACE_ADDRESS;
            use alloy_primitives::TxKind;

            let inner = request.as_ref();
            let target: Option<Address> = match inner.to {
                Some(TxKind::Call(addr)) => Some(addr),
                _ => None,
            };
            let input_bytes: Option<alloy_primitives::Bytes> = inner.input.input().cloned();

            // Intercept estimateRetryableTicket on NodeInterface (0xc8).
            //
            // ABI: estimateRetryableTicket(
            //   address sender, uint256 deposit, address to,
            //   uint256 l2CallValue, address excessFeeRefundAddress,
            //   address callValueRefundAddress, bytes data)
            //
            // selector: 0xc3dc5879
            if target == Some(NODE_INTERFACE_ADDRESS) {
                if let Some(ref buf) = input_bytes {
                    if buf.len() >= 4 && buf[..4] == [0xc3, 0xdc, 0x58, 0x79] {
                        return self
                            .estimate_retryable_ticket_gas(buf, at, state_override)
                            .await;
                    }
                }
            }

            // Extract calldata length before request is consumed by the binary search.
            let calldata_len = input_bytes.as_ref().map(|b| b.len()).unwrap_or(0);

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

    /// Intercept `eth_call` to the NodeInterface (0xc8) virtual contract
    /// for methods that need chain history or nested EVM calls. Methods
    /// that can be resolved at the precompile layer (with zero / empty
    /// fallbacks) are delegated to the default EVM path.
    #[allow(clippy::manual_async_fn)]
    fn call(
        &self,
        request: RpcTxReq<<Self::RpcConvert as RpcConvert>::Network>,
        block_number: Option<BlockId>,
        overrides: alloy_rpc_types_eth::state::EvmOverrides,
    ) -> impl std::future::Future<Output = Result<alloy_primitives::Bytes, Self::Error>> + Send
    {
        async move {
            use crate::nodeinterface_rpc::{
                encode_gas_estimate_components, encode_l2_block_range, gas_estimate_data_len,
                NODE_INTERFACE_ADDRESS, SEL_GAS_ESTIMATE_COMPONENTS, SEL_GAS_ESTIMATE_L1_COMPONENT,
                SEL_L2_BLOCK_RANGE_FOR_L1,
            };
            use alloy_primitives::{Address, TxKind};

            // Only intercept calls targeting the NodeInterface or
            // NodeInterfaceDebug addresses.
            let target: Option<Address> = match request.as_ref().to {
                Some(TxKind::Call(addr)) => Some(addr),
                _ => None,
            };
            let is_ni = target == Some(NODE_INTERFACE_ADDRESS);
            let is_ni_debug = target == Some(arb_precompiles::NODE_INTERFACE_DEBUG_ADDRESS);
            if !is_ni && !is_ni_debug {
                let _permit = self.acquire_owned_blocking_io().await;
                let res = self
                    .transact_call_at(request, block_number.unwrap_or_default(), overrides)
                    .await?;
                return <Self::Error as reth_rpc_eth_types::error::api::FromEvmError<N::Evm>>::ensure_success(res.result);
            }

            // NodeInterfaceDebug (0xc9) has one method: getRetryable(bytes32).
            if is_ni_debug {
                let at = block_number.unwrap_or_default();
                let data: alloy_primitives::Bytes =
                    request.as_ref().input.input().cloned().unwrap_or_default();
                return self.get_retryable_abi(&data, at).await;
            }

            // Parse selector.
            let input_bytes = request.as_ref().input.input().cloned().unwrap_or_default();
            if input_bytes.len() < 4 {
                // Fall back to EVM (which will revert with our precompile).
                let _permit = self.acquire_owned_blocking_io().await;
                let res = self
                    .transact_call_at(request, block_number.unwrap_or_default(), overrides)
                    .await?;
                return <Self::Error as reth_rpc_eth_types::error::api::FromEvmError<N::Evm>>::ensure_success(res.result);
            }
            let selector: [u8; 4] = [
                input_bytes[0],
                input_bytes[1],
                input_bytes[2],
                input_bytes[3],
            ];
            let at = block_number.unwrap_or_default();

            match selector {
                SEL_GAS_ESTIMATE_COMPONENTS | SEL_GAS_ESTIMATE_L1_COMPONENT => {
                    // Run a real eth_estimateGas binary search for the total.
                    // The tx data we want to estimate is the `bytes` param
                    // embedded in the call; we don't have direct access to
                    // it here, so we use the inner tx request calldata as a
                    // proxy for estimation. The L1 component is derived
                    // from the embedded `bytes` param length.
                    let data_len = gas_estimate_data_len(&input_bytes);

                    // Read L1 price and L2 basefee from ArbOS state.
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

                    // Compute gas-for-L1 from ArbOS pricing.
                    let gas_for_l1 = if basefee.is_zero() || l1_price.is_zero() {
                        0u64
                    } else {
                        let l1_fee = l1_price
                            .saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS))
                            .saturating_mul(U256::from(data_len));
                        // Only gasEstimateComponents applies padding.
                        let padded = if selector == SEL_GAS_ESTIMATE_COMPONENTS {
                            l1_fee.saturating_mul(U256::from(GAS_ESTIMATION_L1_PRICE_PADDING))
                                / U256::from(10_000u64)
                        } else {
                            l1_fee
                        };
                        (padded / basefee).try_into().unwrap_or(u64::MAX)
                    };

                    if selector == SEL_GAS_ESTIMATE_L1_COMPONENT {
                        // Returns (uint64, uint256, uint256).
                        let mut out = vec![0u8; 96];
                        out[24..32].copy_from_slice(&gas_for_l1.to_be_bytes());
                        out[32..64].copy_from_slice(&basefee.to_be_bytes::<32>());
                        out[64..96].copy_from_slice(&l1_price.to_be_bytes::<32>());
                        return Ok(alloy_primitives::Bytes::from(out));
                    }

                    // For gasEstimateComponents, compute the full estimate by
                    // issuing an inner estimate_gas_at with the embedded call.
                    // We approximate: the full estimate is the caller's
                    // current request's compute-gas estimate.
                    let compute_gas =
                        EstimateCall::estimate_gas_at(self, request, at, overrides.state).await?;
                    let total: u64 = compute_gas
                        .saturating_add(U256::from(gas_for_l1))
                        .try_into()
                        .unwrap_or(u64::MAX);

                    Ok(encode_gas_estimate_components(
                        total, gas_for_l1, basefee, l1_price,
                    ))
                }

                SEL_L2_BLOCK_RANGE_FOR_L1 => {
                    use reth_provider::{BlockNumReader, BlockReaderIdExt};

                    if input_bytes.len() < 4 + 32 {
                        return Err(EthApiError::InvalidParams(
                            "l2BlockRangeForL1: missing uint64 arg".into(),
                        ));
                    }
                    let target_l1: u64 = U256::from_be_slice(&input_bytes[4..36])
                        .try_into()
                        .unwrap_or(u64::MAX);

                    let provider = self.inner.provider().clone();
                    let best = provider
                        .best_block_number()
                        .map_err(|e| EthApiError::Internal(e.into()))?;

                    let mix_hash_of = move |n: u64| -> Option<B256> {
                        use alloy_consensus::BlockHeader;
                        provider
                            .sealed_header_by_number_or_tag(
                                alloy_rpc_types_eth::BlockNumberOrTag::Number(n),
                            )
                            .ok()
                            .flatten()
                            .and_then(|h| h.header().mix_hash())
                    };

                    match crate::nodeinterface_rpc::find_l2_block_range(
                        target_l1,
                        best,
                        mix_hash_of,
                    ) {
                        Some((first, last)) => Ok(encode_l2_block_range(first, last)),
                        None => Err(EthApiError::InvalidParams(format!(
                            "l2BlockRangeForL1: no L2 blocks found for L1 block {target_l1}"
                        ))),
                    }
                }

                // estimateRetryableTicket via eth_call.
                [0xc3, 0xdc, 0x58, 0x79] => {
                    self.simulate_retryable_ticket_call(&input_bytes, at, overrides)
                        .await
                }

                // constructOutboxProof(uint64 size, uint64 leaf): scan
                // ArbSys SendMerkleUpdate / L2ToL1Tx events, build a
                // position → hash map, run the outbox-proof algorithm.
                [0x42, 0x69, 0x63, 0x50] => self.construct_outbox_proof(&input_bytes, at).await,

                _ => {
                    // Delegate to EVM (precompile returns zero / reverts).
                    let _permit = self.acquire_owned_blocking_io().await;
                    let res = self.transact_call_at(request, at, overrides).await?;
                    <Self::Error as reth_rpc_eth_types::error::api::FromEvmError<N::Evm>>::ensure_success(res.result)
                }
            }
        }
    }
}
