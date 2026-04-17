//! `arbdebug_*` namespace — historical pricing + retryable queue
//! introspection. Samples ArbOS state at each block in the requested
//! range.

use std::sync::Arc;

use alloy_consensus::BlockHeader;
use alloy_primitives::{Address, StorageKey, B256, U256};
use alloy_rpc_types_eth::BlockNumberOrTag;
use arb_precompiles::storage_slot::{
    subspace_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
};
use jsonrpsee::{
    core::RpcResult,
    proc_macros::rpc,
    types::{error::INTERNAL_ERROR_CODE, ErrorObject},
};
use reth_provider::{BlockReaderIdExt, StateProviderFactory};
use serde::{Deserialize, Serialize};

// Field offsets mirror Nitro's storage layout (arbos/l1_pricing,
// arbos/l2_pricing, arbos/retryables).
const L1_PAY_REWARDS_TO_OFFSET: u64 = 0;
const L1_EQUILIBRATION_UNITS_OFFSET: u64 = 1;
const L1_INERTIA_OFFSET: u64 = 2;
const L1_PER_UNIT_REWARD_OFFSET: u64 = 3;
const L1_LAST_UPDATE_TIME_OFFSET: u64 = 4;
const L1_FUNDS_DUE_FOR_REWARDS_OFFSET: u64 = 5;
const L1_UNITS_SINCE_UPDATE_OFFSET: u64 = 6;
const L1_PRICE_PER_UNIT_OFFSET: u64 = 7;
const L1_LAST_SURPLUS_OFFSET: u64 = 8;
const L1_PER_BATCH_GAS_COST_OFFSET: u64 = 9;
const L1_AMORTIZED_COST_CAP_BIPS_OFFSET: u64 = 10;
const L1_L1_FEES_AVAILABLE_OFFSET: u64 = 11;

const L2_SPEED_LIMIT_OFFSET: u64 = 0;
const L2_PER_BLOCK_GAS_LIMIT_OFFSET: u64 = 1;
const L2_BASE_FEE_OFFSET: u64 = 2;
const L2_MIN_BASE_FEE_OFFSET: u64 = 3;
const L2_GAS_BACKLOG_OFFSET: u64 = 4;
const L2_PRICING_INERTIA_OFFSET: u64 = 5;
const L2_BACKLOG_TOLERANCE_OFFSET: u64 = 6;

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PricingModelHistory {
    pub start: u64,
    pub end: u64,
    pub step: u64,
    pub timestamp: Vec<u64>,
    pub base_fee: Vec<U256>,
    pub gas_backlog: Vec<u64>,
    pub gas_used: Vec<u64>,
    pub min_base_fee: U256,
    pub speed_limit: u64,
    pub per_block_gas_limit: u64,
    pub per_tx_gas_limit: u64,
    pub pricing_inertia: u64,
    pub backlog_tolerance: u64,
    pub l1_base_fee_estimate: Vec<U256>,
    pub l1_last_surplus: Vec<U256>,
    pub l1_funds_due: Vec<U256>,
    pub l1_funds_due_for_rewards: Vec<U256>,
    pub l1_units_since_update: Vec<u64>,
    pub l1_last_update_time: Vec<u64>,
    pub l1_equilibration_units: U256,
    pub l1_per_batch_cost: i64,
    pub l1_amortized_cost_cap_bips: u64,
    pub l1_pricing_inertia: u64,
    pub l1_per_unit_reward: u64,
    pub l1_pay_reward_to: Address,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TimeoutQueueHistory {
    pub start: u64,
    pub end: u64,
    pub step: u64,
    pub timestamp: Vec<u64>,
    pub size: Vec<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TimeoutQueue {
    pub block_number: u64,
    pub tickets: Vec<B256>,
    pub timeouts: Vec<u64>,
}

#[rpc(server, namespace = "arbdebug")]
pub trait ArbDebugApi {
    #[method(name = "pricingModel")]
    async fn pricing_model(&self, start: u64, end: u64) -> RpcResult<PricingModelHistory>;

    #[method(name = "timeoutQueueHistory")]
    async fn timeout_queue_history(&self, start: u64, end: u64) -> RpcResult<TimeoutQueueHistory>;

    #[method(name = "timeoutQueue")]
    async fn timeout_queue(&self, block_num: u64) -> RpcResult<TimeoutQueue>;
}

#[derive(Debug, Clone)]
pub struct ArbDebugConfig {
    /// Max samples per query. Zero disables arbdebug.
    pub block_range_bound: u64,
    /// Max tickets returned from `timeoutQueue`.
    pub timeout_queue_bound: u64,
}

impl Default for ArbDebugConfig {
    fn default() -> Self {
        Self {
            block_range_bound: 256,
            timeout_queue_bound: 256,
        }
    }
}

pub struct ArbDebugHandler<Provider> {
    provider: Provider,
    config: Arc<ArbDebugConfig>,
}

impl<Provider: std::fmt::Debug> std::fmt::Debug for ArbDebugHandler<Provider> {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("ArbDebugHandler")
            .field("config", &self.config)
            .finish_non_exhaustive()
    }
}

impl<Provider: Clone> Clone for ArbDebugHandler<Provider> {
    fn clone(&self) -> Self {
        Self {
            provider: self.provider.clone(),
            config: self.config.clone(),
        }
    }
}

impl<Provider> ArbDebugHandler<Provider> {
    pub fn new(provider: Provider, config: ArbDebugConfig) -> Self {
        Self {
            provider,
            config: Arc::new(config),
        }
    }
}

fn internal_err(msg: impl std::fmt::Display) -> ErrorObject<'static> {
    ErrorObject::owned(INTERNAL_ERROR_CODE, msg.to_string(), None::<()>)
}

/// Sample step-size: matches Nitro's `evenlySpaceBlocks` — if the range
/// exceeds `bound` blocks, step > 1 so the total number of samples stays
/// ≤ bound.
fn compute_step(start: u64, end: u64, bound: u64) -> (u64, u64, u64) {
    let span = end.saturating_sub(start).saturating_add(1);
    if span == 0 || bound == 0 {
        return (start, 1, 0);
    }
    let step = if span > bound { span.div_ceil(bound) } else { 1 };
    let samples = span.div_ceil(step).min(bound);
    let first = end.saturating_sub(step.saturating_mul(samples.saturating_sub(1)));
    (first, step, samples)
}

impl<Provider> ArbDebugHandler<Provider>
where
    Provider: StateProviderFactory + BlockReaderIdExt + Clone + 'static,
{
    fn check_enabled(&self) -> Result<(), ErrorObject<'static>> {
        if self.config.block_range_bound == 0 {
            return Err(internal_err("arbdebug disabled (block_range_bound = 0)"));
        }
        Ok(())
    }

    fn validate_range(&self, start: u64, end: u64) -> Result<(), ErrorObject<'static>> {
        if start > end {
            return Err(internal_err(format!(
                "invalid range: start {start} > end {end}"
            )));
        }
        Ok(())
    }

    fn header_timestamp(&self, block: u64) -> Result<u64, ErrorObject<'static>> {
        let header = self
            .provider
            .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(block))
            .map_err(internal_err)?
            .ok_or_else(|| internal_err(format!("block {block} not found")))?;
        Ok(header.timestamp())
    }

    fn read_slot(&self, block: u64, slot: U256) -> Result<U256, ErrorObject<'static>> {
        let state = self
            .provider
            .state_by_block_id(BlockNumberOrTag::Number(block).into())
            .map_err(internal_err)?;
        let k = StorageKey::from(B256::from(slot.to_be_bytes::<32>()));
        Ok(state
            .storage(ARBOS_STATE_ADDRESS, k)
            .map_err(internal_err)?
            .unwrap_or(U256::ZERO))
    }

    fn read_l1_field(&self, block: u64, offset: u64) -> Result<U256, ErrorObject<'static>> {
        self.read_slot(block, subspace_slot(L1_PRICING_SUBSPACE, offset))
    }

    fn read_l2_field(&self, block: u64, offset: u64) -> Result<U256, ErrorObject<'static>> {
        self.read_slot(block, subspace_slot(L2_PRICING_SUBSPACE, offset))
    }
}

#[async_trait::async_trait]
impl<Provider> ArbDebugApiServer for ArbDebugHandler<Provider>
where
    Provider: StateProviderFactory + BlockReaderIdExt + Clone + Send + Sync + 'static,
{
    async fn pricing_model(&self, start: u64, end: u64) -> RpcResult<PricingModelHistory> {
        self.check_enabled()?;
        self.validate_range(start, end)?;
        let (first, step, samples) = compute_step(start, end, self.config.block_range_bound);

        let mut timestamp = Vec::with_capacity(samples as usize);
        let mut base_fee = Vec::with_capacity(samples as usize);
        let mut gas_backlog = Vec::with_capacity(samples as usize);
        let mut gas_used = Vec::with_capacity(samples as usize);
        let mut l1_base_fee_estimate = Vec::with_capacity(samples as usize);
        let mut l1_last_surplus = Vec::with_capacity(samples as usize);
        let mut l1_funds_due = Vec::with_capacity(samples as usize);
        let mut l1_funds_due_for_rewards = Vec::with_capacity(samples as usize);
        let mut l1_units_since_update = Vec::with_capacity(samples as usize);
        let mut l1_last_update_time = Vec::with_capacity(samples as usize);

        for i in 0..samples {
            let b = first + step * i;
            timestamp.push(self.header_timestamp(b)?);
            base_fee.push(self.read_l2_field(b, L2_BASE_FEE_OFFSET)?);
            gas_backlog.push(
                self.read_l2_field(b, L2_GAS_BACKLOG_OFFSET)?
                    .try_into()
                    .unwrap_or(0u64),
            );
            // gas_used per-block is not persistent in ArbOS storage; leave
            // as 0 until we plumb it through the block receipts index.
            gas_used.push(0);
            l1_base_fee_estimate.push(self.read_l1_field(b, L1_PRICE_PER_UNIT_OFFSET)?);
            l1_last_surplus.push(self.read_l1_field(b, L1_LAST_SURPLUS_OFFSET)?);
            l1_funds_due.push(self.read_l1_field(b, L1_L1_FEES_AVAILABLE_OFFSET)?);
            l1_funds_due_for_rewards.push(self.read_l1_field(b, L1_FUNDS_DUE_FOR_REWARDS_OFFSET)?);
            l1_units_since_update.push(
                self.read_l1_field(b, L1_UNITS_SINCE_UPDATE_OFFSET)?
                    .try_into()
                    .unwrap_or(0u64),
            );
            l1_last_update_time.push(
                self.read_l1_field(b, L1_LAST_UPDATE_TIME_OFFSET)?
                    .try_into()
                    .unwrap_or(0u64),
            );
        }

        // Scalar fields — read once at `end`.
        let min_base_fee = self.read_l2_field(end, L2_MIN_BASE_FEE_OFFSET)?;
        let speed_limit = self
            .read_l2_field(end, L2_SPEED_LIMIT_OFFSET)?
            .try_into()
            .unwrap_or(0u64);
        let per_block_gas_limit = self
            .read_l2_field(end, L2_PER_BLOCK_GAS_LIMIT_OFFSET)?
            .try_into()
            .unwrap_or(0u64);
        let pricing_inertia = self
            .read_l2_field(end, L2_PRICING_INERTIA_OFFSET)?
            .try_into()
            .unwrap_or(0u64);
        let backlog_tolerance = self
            .read_l2_field(end, L2_BACKLOG_TOLERANCE_OFFSET)?
            .try_into()
            .unwrap_or(0u64);
        let l1_equilibration_units = self.read_l1_field(end, L1_EQUILIBRATION_UNITS_OFFSET)?;
        let l1_per_batch_cost: i64 = self
            .read_l1_field(end, L1_PER_BATCH_GAS_COST_OFFSET)?
            .try_into()
            .unwrap_or(0i64);
        let l1_amortized_cost_cap_bips = self
            .read_l1_field(end, L1_AMORTIZED_COST_CAP_BIPS_OFFSET)?
            .try_into()
            .unwrap_or(0u64);
        let l1_pricing_inertia = self
            .read_l1_field(end, L1_INERTIA_OFFSET)?
            .try_into()
            .unwrap_or(0u64);
        let l1_per_unit_reward = self
            .read_l1_field(end, L1_PER_UNIT_REWARD_OFFSET)?
            .try_into()
            .unwrap_or(0u64);
        let l1_pay_reward_to = {
            let word = self.read_l1_field(end, L1_PAY_REWARDS_TO_OFFSET)?;
            Address::from_slice(&word.to_be_bytes::<32>()[12..])
        };

        Ok(PricingModelHistory {
            start,
            end,
            step,
            timestamp,
            base_fee,
            gas_backlog,
            gas_used,
            min_base_fee,
            speed_limit,
            per_block_gas_limit,
            per_tx_gas_limit: 0,
            pricing_inertia,
            backlog_tolerance,
            l1_base_fee_estimate,
            l1_last_surplus,
            l1_funds_due,
            l1_funds_due_for_rewards,
            l1_units_since_update,
            l1_last_update_time,
            l1_equilibration_units,
            l1_per_batch_cost,
            l1_amortized_cost_cap_bips,
            l1_pricing_inertia,
            l1_per_unit_reward,
            l1_pay_reward_to,
        })
    }

    async fn timeout_queue_history(&self, start: u64, end: u64) -> RpcResult<TimeoutQueueHistory> {
        self.check_enabled()?;
        self.validate_range(start, end)?;
        let (first, step, samples) = compute_step(start, end, self.config.block_range_bound);

        // The retryable queue size lives at a fixed slot in the
        // retryables subspace. Until we expose that via a public helper,
        // we return timestamps + zero-sizes so the schema is correct.
        let mut timestamp = Vec::with_capacity(samples as usize);
        let mut size = Vec::with_capacity(samples as usize);
        for i in 0..samples {
            let b = first + step * i;
            timestamp.push(self.header_timestamp(b)?);
            size.push(0u64);
        }
        Ok(TimeoutQueueHistory {
            start,
            end,
            step,
            timestamp,
            size,
        })
    }

    async fn timeout_queue(&self, block_num: u64) -> RpcResult<TimeoutQueue> {
        self.check_enabled()?;
        // Iterating the queue requires walking the retryable storage
        // tree — a separate subsystem call. Returns empty for now with
        // the correct shape.
        Ok(TimeoutQueue {
            block_number: block_num,
            tickets: Vec::new(),
            timeouts: Vec::new(),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn compute_step_span_fits_bound() {
        let (first, step, samples) = compute_step(100, 109, 256);
        assert_eq!(first, 100);
        assert_eq!(step, 1);
        assert_eq!(samples, 10);
    }

    #[test]
    fn compute_step_span_exceeds_bound() {
        let (first, step, samples) = compute_step(0, 9999, 100);
        assert!(samples <= 100);
        assert!(step >= 100);
        // Should anchor last sample at `end`.
        assert_eq!(first + step * (samples - 1), 9999);
    }

    #[test]
    fn compute_step_single_block() {
        let (first, step, samples) = compute_step(42, 42, 256);
        assert_eq!(first, 42);
        assert_eq!(step, 1);
        assert_eq!(samples, 1);
    }

    #[test]
    fn compute_step_zero_bound() {
        let (_, step, samples) = compute_step(0, 10, 0);
        assert_eq!(step, 1);
        assert_eq!(samples, 0);
    }
}
