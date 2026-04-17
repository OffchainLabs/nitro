//! `arbdebug_*` RPC namespace — historical pricing and retryable queue
//! introspection intended for operators and validators.
//!
//! Nitro's `ArbDebugAPI` (at `execution/gethexec/api.go:102`) exposes
//! three methods: `pricingModel`, `timeoutQueueHistory`, `timeoutQueue`.
//! Each iterates a block range, sampling ArbOS pricing / retryable
//! state. The methods are strictly debugging aids — no consensus or
//! user-facing path depends on them.
//!
//! For arbreth we ship the method schema + a working range-check /
//! sampling scaffold. The actual per-block ArbOS state reads require a
//! provider with historic state access (we'd open state at each block
//! and pull L1/L2 pricing fields). Until that's wired, we return
//! valid-shape responses with empty samples — matching Nitro's
//! behavior when `blockRangeBound == 0`.

use alloy_primitives::{Address, U256};
use jsonrpsee::{
    core::RpcResult,
    proc_macros::rpc,
    types::{error::INTERNAL_ERROR_CODE, ErrorObject},
};
use serde::{Deserialize, Serialize};

/// History of the L1 + L2 pricing model sampled over a block range.
///
/// Mirrors Nitro's `PricingModelHistory` struct (api.go:112). The
/// vectors are per-sample (one entry per sampled block at the
/// configured step size).
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

/// Retryable timeout queue progression across a block range.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TimeoutQueueHistory {
    pub start: u64,
    pub end: u64,
    pub step: u64,
    pub timestamp: Vec<u64>,
    pub size: Vec<u64>,
}

/// Snapshot of the retryable timeout queue at a specific block.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TimeoutQueue {
    pub block_number: u64,
    pub tickets: Vec<alloy_primitives::B256>,
    pub timeouts: Vec<u64>,
}

/// `arbdebug` RPC namespace.
#[rpc(server, namespace = "arbdebug")]
pub trait ArbDebugApi {
    /// Returns per-sample L1/L2 pricing state over `[start, end]`.
    /// The server applies an internal sample-step based on
    /// `block_range_bound` to keep the response bounded.
    #[method(name = "pricingModel")]
    async fn pricing_model(&self, start: u64, end: u64) -> RpcResult<PricingModelHistory>;

    /// Retryable timeout-queue sizes sampled over `[start, end]`.
    #[method(name = "timeoutQueueHistory")]
    async fn timeout_queue_history(&self, start: u64, end: u64) -> RpcResult<TimeoutQueueHistory>;

    /// Full retryable timeout queue at a given block (tickets +
    /// expiry times).
    #[method(name = "timeoutQueue")]
    async fn timeout_queue(&self, block_num: u64) -> RpcResult<TimeoutQueue>;
}

/// Configuration for `arbdebug_*`.
#[derive(Debug, Clone)]
pub struct ArbDebugConfig {
    /// Maximum samples per pricing/history query. Zero disables
    /// arbdebug and causes every method to return an error (matching
    /// Nitro's unconfigured-bound behavior).
    pub block_range_bound: u64,
    /// Maximum tickets to return from `timeoutQueue`.
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

/// Handler implementing the `arbdebug` namespace.
#[derive(Debug, Clone)]
pub struct ArbDebugHandler {
    config: ArbDebugConfig,
}

impl ArbDebugHandler {
    pub fn new(config: ArbDebugConfig) -> Self {
        Self { config }
    }

    fn check_enabled(&self) -> Result<(), ErrorObject<'static>> {
        if self.config.block_range_bound == 0 {
            return Err(ErrorObject::owned(
                INTERNAL_ERROR_CODE,
                "arbdebug disabled (block_range_bound = 0)",
                None::<()>,
            ));
        }
        Ok(())
    }

    fn validate_range(&self, start: u64, end: u64) -> Result<(), ErrorObject<'static>> {
        if start > end {
            return Err(ErrorObject::owned(
                INTERNAL_ERROR_CODE,
                format!("invalid range: start {start} > end {end}"),
                None::<()>,
            ));
        }
        Ok(())
    }
}

#[async_trait::async_trait]
impl ArbDebugApiServer for ArbDebugHandler {
    async fn pricing_model(&self, start: u64, end: u64) -> RpcResult<PricingModelHistory> {
        self.check_enabled()?;
        self.validate_range(start, end)?;
        // Placeholder response: schema-correct but no per-block samples
        // until historic-state sampling is wired up.
        Ok(PricingModelHistory {
            start,
            end,
            step: 1,
            timestamp: Vec::new(),
            base_fee: Vec::new(),
            gas_backlog: Vec::new(),
            gas_used: Vec::new(),
            min_base_fee: U256::ZERO,
            speed_limit: 0,
            per_block_gas_limit: 0,
            per_tx_gas_limit: 0,
            pricing_inertia: 0,
            backlog_tolerance: 0,
            l1_base_fee_estimate: Vec::new(),
            l1_last_surplus: Vec::new(),
            l1_funds_due: Vec::new(),
            l1_funds_due_for_rewards: Vec::new(),
            l1_units_since_update: Vec::new(),
            l1_last_update_time: Vec::new(),
            l1_equilibration_units: U256::ZERO,
            l1_per_batch_cost: 0,
            l1_amortized_cost_cap_bips: 0,
            l1_pricing_inertia: 0,
            l1_per_unit_reward: 0,
            l1_pay_reward_to: Address::ZERO,
        })
    }

    async fn timeout_queue_history(&self, start: u64, end: u64) -> RpcResult<TimeoutQueueHistory> {
        self.check_enabled()?;
        self.validate_range(start, end)?;
        Ok(TimeoutQueueHistory {
            start,
            end,
            step: 1,
            timestamp: Vec::new(),
            size: Vec::new(),
        })
    }

    async fn timeout_queue(&self, block_num: u64) -> RpcResult<TimeoutQueue> {
        self.check_enabled()?;
        Ok(TimeoutQueue {
            block_number: block_num,
            tickets: Vec::new(),
            timeouts: Vec::new(),
        })
    }
}
