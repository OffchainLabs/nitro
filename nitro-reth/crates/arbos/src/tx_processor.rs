use alloy_primitives::{Address, B256, U256};
use std::collections::HashMap;

use arb_chainspec::arbos_version as arb_ver;
use crate::l1_pricing;
use crate::retryables;

/// ArbOS system address (0x00000000000000000000000000000000000a4b05).
pub const ARBOS_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[17] = 0x0a;
    bytes[18] = 0x4b;
    bytes[19] = 0x05;
    Address::new(bytes)
};

/// Padding applied to L1 gas price estimates for safety margin (110% = 11000 bips).
pub const GAS_ESTIMATION_L1_PRICE_PADDING_BIPS: u64 = 11000;

/// Per-transaction state for processing Arbitrum transactions.
///
/// Created and freed for every L2 transaction. Tracks ArbOS state
/// that influences transaction processing. In reth, this is used by
/// the block executor's per-transaction logic.
#[derive(Debug)]
pub struct TxProcessor {
    /// The poster's fee contribution (L1 calldata cost expressed in ETH).
    pub poster_fee: U256,
    /// Gas reserved for L1 posting costs.
    pub poster_gas: u64,
    /// Gas temporarily held to prevent compute from exceeding the gas limit.
    pub compute_hold_gas: u64,
    /// Whether this tx was submitted through the delayed inbox.
    pub delayed_inbox: bool,
    /// The top-level tx type byte, set in StartTxHook.
    pub top_tx_type: Option<u8>,
    /// The current retryable ticket being redeemed (if any).
    pub current_retryable: Option<B256>,
    /// The refund-to address for retryable redeems.
    pub current_refund_to: Option<Address>,
    /// Scheduled transactions (e.g., retryable auto-redeems).
    pub scheduled_txs: Vec<Vec<u8>>,
    /// Count of open Stylus program contexts per contract address.
    /// Used to detect reentrance.
    pub programs_depth: HashMap<Address, usize>,
}

impl Default for TxProcessor {
    fn default() -> Self {
        Self {
            poster_fee: U256::ZERO,
            poster_gas: 0,
            compute_hold_gas: 0,
            delayed_inbox: false,
            top_tx_type: None,
            current_retryable: None,
            current_refund_to: None,
            scheduled_txs: Vec::new(),
            programs_depth: HashMap::new(),
        }
    }
}

impl TxProcessor {
    /// Create a new TxProcessor. The `delayed_inbox` flag indicates whether the
    /// coinbase differs from the batch poster address.
    pub fn new(coinbase: Address) -> Self {
        Self {
            delayed_inbox: coinbase != l1_pricing::BATCH_POSTER_ADDRESS,
            ..Self::default()
        }
    }

    /// Gas that should not be refundable (the poster's L1 cost component).
    pub fn nonrefundable_gas(&self) -> u64 {
        self.poster_gas
    }

    /// Gas held back to limit computation; must be refunded after computation completes.
    pub fn held_gas(&self) -> u64 {
        self.compute_hold_gas
    }

    /// Whether the tip should be dropped (version-gated behavior).
    pub fn drop_tip(&self, arbos_version: u64) -> bool {
        arbos_version != 9 || self.delayed_inbox
    }

    /// Get the effective gas price paid.
    pub fn get_paid_gas_price(&self, arbos_version: u64, base_fee: U256, gas_price: U256) -> U256 {
        if arbos_version != 9 {
            base_fee
        } else {
            gas_price
        }
    }

    /// The GASPRICE opcode return value.
    pub fn gas_price_op(&self, arbos_version: u64, base_fee: U256, gas_price: U256) -> U256 {
        if arbos_version >= 3 {
            self.get_paid_gas_price(arbos_version, base_fee, gas_price)
        } else {
            gas_price
        }
    }

    /// Fill receipt info with the poster gas used for L1.
    pub fn fill_receipt_gas_used_for_l1(&self) -> u64 {
        self.poster_gas
    }

    // -----------------------------------------------------------------
    // Stylus / WASM Execution
    // -----------------------------------------------------------------

    /// Record entering a Stylus program context for a contract address.
    pub fn push_program(&mut self, addr: Address) {
        *self.programs_depth.entry(addr).or_insert(0) += 1;
    }

    /// Record leaving a Stylus program context for a contract address.
    pub fn pop_program(&mut self, addr: Address) {
        if let Some(count) = self.programs_depth.get_mut(&addr) {
            *count = count.saturating_sub(1);
            if *count == 0 {
                self.programs_depth.remove(&addr);
            }
        }
    }

    /// Whether the given address has a reentrant Stylus call.
    pub fn is_reentrant(&self, addr: &Address) -> bool {
        self.programs_depth.get(addr).copied().unwrap_or(0) > 1
    }

    // -----------------------------------------------------------------
    // Reverted Tx Hook
    // -----------------------------------------------------------------

    /// Check for pre-recorded reverted or filtered transactions.
    ///
    /// Returns an action describing how the caller should handle this tx
    /// before normal execution. The caller should:
    /// - `None`: proceed with normal execution
    /// - `PreRecordedRevert`: increment sender nonce, deduct `gas_to_consume`
    ///   from gas remaining, and return execution-reverted error
    /// - `FilteredTx`: increment sender nonce, consume ALL remaining gas,
    ///   and return filtered-tx error
    pub fn reverted_tx_hook(
        &self,
        tx_hash: Option<B256>,
        pre_recorded_gas: Option<u64>,
        is_filtered: bool,
    ) -> RevertedTxAction {
        let Some(_hash) = tx_hash else {
            return RevertedTxAction::None;
        };

        if let Some(l2_gas_used) = pre_recorded_gas {
            let adjusted_gas = l2_gas_used.saturating_sub(TX_GAS);
            return RevertedTxAction::PreRecordedRevert {
                gas_to_consume: adjusted_gas,
            };
        }

        if is_filtered {
            return RevertedTxAction::FilteredTx;
        }

        RevertedTxAction::None
    }

    // -----------------------------------------------------------------
    // Start Tx Hook helpers
    // -----------------------------------------------------------------

    /// Set the top-level transaction type for this tx.
    pub fn set_tx_type(&mut self, tx_type: u8) {
        self.top_tx_type = Some(tx_type);
    }

    /// Set up state for processing a retry transaction.
    ///
    /// The caller should:
    /// 1. Verify the retryable exists (via `RetryableState::open_retryable`)
    /// 2. Transfer call value from escrow to `from`
    /// 3. Mint prepaid gas (`base_fee * gas`) to `from`
    /// 4. Continue to gas charging and EVM execution
    pub fn prepare_retry_tx(&mut self, ticket_id: B256, refund_to: Address) {
        self.current_retryable = Some(ticket_id);
        self.current_refund_to = Some(refund_to);
    }

    // -----------------------------------------------------------------
    // Gas Charging Hook
    // -----------------------------------------------------------------

    /// Compute poster gas and held compute gas.
    ///
    /// Charges poster data cost from the remaining gas and holds excess
    /// compute gas to enforce per-block/per-tx limits. After calling,
    /// `poster_gas`, `poster_fee`, and `compute_hold_gas` are set.
    pub fn gas_charging_hook(
        &mut self,
        gas_remaining: &mut u64,
        intrinsic_gas: u64,
        params: &GasChargingParams,
    ) -> Result<(), GasChargingError> {
        let mut gas_needed = 0u64;

        if !params.base_fee.is_zero() && !params.skip_l1_charging {
            self.poster_gas = compute_poster_gas(
                params.poster_cost,
                params.base_fee,
                params.is_gas_estimation,
                params.min_base_fee,
            );
            self.poster_fee = params.base_fee.saturating_mul(U256::from(self.poster_gas));
            gas_needed = self.poster_gas;
        }

        if *gas_remaining < gas_needed {
            return Err(GasChargingError::IntrinsicGasTooLow);
        }
        *gas_remaining -= gas_needed;

        // Hold excess compute gas to enforce per-block/per-tx limits.
        if !params.is_eth_call {
            let max = if params.arbos_version < arb_ver::ARBOS_VERSION_50 {
                params.per_block_gas_limit
            } else {
                // ArbOS 50+ uses per-tx limit, reduced by already-charged intrinsic gas.
                params.per_tx_gas_limit.saturating_sub(intrinsic_gas)
            };

            if *gas_remaining > max {
                self.compute_hold_gas = *gas_remaining - max;
                *gas_remaining = max;
            }
        }

        Ok(())
    }

    // -----------------------------------------------------------------
    // End Tx Hook (normal transactions)
    // -----------------------------------------------------------------

    /// Compute fee distribution for a normal (non-retryable) transaction.
    ///
    /// Returns the amounts to mint to each fee account and the gas to
    /// add to the backlog. The caller executes the balance operations.
    pub fn compute_end_tx_fee_distribution(
        &self,
        params: &EndTxNormalParams,
    ) -> EndTxFeeDistribution {
        let gas_used = params.gas_used;
        let base_fee = params.base_fee;

        let total_cost = base_fee.saturating_mul(U256::from(gas_used));
        let mut compute_cost = total_cost.saturating_sub(self.poster_fee);
        let mut poster_fee = self.poster_fee;

        if total_cost < self.poster_fee {
            tracing::error!(
                gas_used,
                ?base_fee,
                poster_fee = ?self.poster_fee,
                "total cost < poster cost"
            );
            poster_fee = U256::ZERO;
            compute_cost = total_cost;
        }

        let mut infra_fee_amount = U256::ZERO;

        if params.arbos_version > 4
            && params.infra_fee_account != Address::ZERO
        {
            let infra_fee = params.min_base_fee.min(base_fee);
            let compute_gas = gas_used.saturating_sub(self.poster_gas);
            infra_fee_amount = infra_fee.saturating_mul(U256::from(compute_gas));
            compute_cost = compute_cost.saturating_sub(infra_fee_amount);
        }

        let poster_fee_destination = if params.arbos_version < 2 {
            params.coinbase
        } else {
            l1_pricing::L1_PRICER_FUNDS_POOL_ADDRESS
        };

        let l1_fees_to_add = if params.arbos_version >= arb_ver::ARBOS_VERSION_10 {
            poster_fee
        } else {
            U256::ZERO
        };

        let compute_gas_for_backlog = if !params.gas_price.is_zero() {
            if gas_used > self.poster_gas {
                gas_used - self.poster_gas
            } else {
                tracing::error!(
                    gas_used,
                    poster_gas = self.poster_gas,
                    "gas used < poster gas"
                );
                gas_used
            }
        } else {
            0
        };

        EndTxFeeDistribution {
            infra_fee_account: params.infra_fee_account,
            infra_fee_amount,
            network_fee_account: params.network_fee_account,
            network_fee_amount: compute_cost,
            poster_fee_destination,
            poster_fee_amount: poster_fee,
            l1_fees_to_add,
            compute_gas_for_backlog,
        }
    }

    // -----------------------------------------------------------------
    // End Tx Hook (retryable transactions)
    // -----------------------------------------------------------------

    /// Process end-of-tx for a retryable redemption.
    ///
    /// Handles undoing geth's gas refund, distributing refunds between
    /// the refund-to address and the sender, and determining whether
    /// to delete the retryable or return value to escrow.
    pub fn end_tx_retryable<F>(
        &self,
        params: &EndTxRetryableParams,
        mut burn_fn: impl FnMut(Address, U256),
        mut transfer_fn: F,
    ) -> EndTxRetryableResult
    where
        F: FnMut(Address, Address, U256) -> Result<(), ()>,
    {
        let effective_base_fee = params.effective_base_fee;
        let gas_left = params.gas_left;
        let gas_used = params.gas_used;

        // Undo geth's gas refund to From.
        let gas_refund_amount = effective_base_fee.saturating_mul(U256::from(gas_left));
        burn_fn(params.from, gas_refund_amount);

        let single_gas_cost = effective_base_fee.saturating_mul(U256::from(gas_used));

        let mut max_refund = params.max_refund;

        if params.success {
            // Refund submission fee from network account.
            refund_with_pool(
                params.network_fee_account,
                params.submission_fee_refund,
                &mut max_refund,
                params.refund_to,
                params.from,
                &mut transfer_fn,
            );
        } else {
            // Submission fee taken but not refunded on failure.
            take_funds(&mut max_refund, params.submission_fee_refund);
        }

        // Gas cost conceptually taken from the L1 deposit pool.
        take_funds(&mut max_refund, single_gas_cost);

        // Refund unused gas.
        let mut network_refund = gas_refund_amount;

        if params.arbos_version >= arb_ver::ARBOS_VERSION_11
            && params.infra_fee_account != Address::ZERO
        {
            let infra_fee = params.min_base_fee.min(effective_base_fee);
            let infra_refund_amount = infra_fee.saturating_mul(U256::from(gas_left));
            let infra_refund = take_funds(&mut network_refund, infra_refund_amount);
            refund_with_pool(
                params.infra_fee_account,
                infra_refund,
                &mut max_refund,
                params.refund_to,
                params.from,
                &mut transfer_fn,
            );
        }

        refund_with_pool(
            params.network_fee_account,
            network_refund,
            &mut max_refund,
            params.refund_to,
            params.from,
            &mut transfer_fn,
        );

        // Multi-dimensional gas refund: if multi-gas cost < single-gas cost,
        // refund the difference. Only when effective_base_fee == block_base_fee
        // (skip during retryable gas estimation).
        if let Some(multi_cost) = params.multi_dimensional_cost {
            let should_refund = single_gas_cost > multi_cost
                && effective_base_fee == params.block_base_fee;
            if should_refund {
                let refund_amount = single_gas_cost.saturating_sub(multi_cost);
                refund_with_pool(
                    params.network_fee_account,
                    refund_amount,
                    &mut max_refund,
                    params.refund_to,
                    params.from,
                    &mut transfer_fn,
                );
            }
        }

        let escrow = retryables::retryable_escrow_address(params.ticket_id);

        EndTxRetryableResult {
            compute_gas_for_backlog: gas_used,
            should_delete_retryable: params.success,
            should_return_value_to_escrow: !params.success,
            escrow_address: escrow,
        }
    }
}

// =====================================================================
// Parameter and result types
// =====================================================================

/// Parameters for the gas charging hook.
#[derive(Debug, Clone)]
pub struct GasChargingParams {
    /// The current block base fee.
    pub base_fee: U256,
    /// The computed poster data cost for this tx.
    pub poster_cost: U256,
    /// Whether this is gas estimation (eth_estimateGas).
    pub is_gas_estimation: bool,
    /// Whether this is an eth_call (non-mutating).
    pub is_eth_call: bool,
    /// Whether to skip L1 charging.
    pub skip_l1_charging: bool,
    /// The minimum L2 base fee.
    pub min_base_fee: U256,
    /// The per-block gas limit from L2 pricing.
    pub per_block_gas_limit: u64,
    /// The per-tx gas limit from L2 pricing (ArbOS v50+).
    pub per_tx_gas_limit: u64,
    /// Current ArbOS version.
    pub arbos_version: u64,
}

/// Error from gas charging.
#[derive(Debug, Clone, thiserror::Error)]
pub enum GasChargingError {
    #[error("intrinsic gas too low")]
    IntrinsicGasTooLow,
}

/// Parameters for end-tx fee distribution (normal transactions).
#[derive(Debug, Clone)]
pub struct EndTxNormalParams {
    pub gas_used: u64,
    pub gas_price: U256,
    pub base_fee: U256,
    pub coinbase: Address,
    pub network_fee_account: Address,
    pub infra_fee_account: Address,
    pub min_base_fee: U256,
    pub arbos_version: u64,
}

/// Fee distribution result from end-tx hook (normal transactions).
///
/// The caller mints `infra_fee_amount` to `infra_fee_account`,
/// `network_fee_amount` to `network_fee_account`, and
/// `poster_fee_amount` to `poster_fee_destination`. Then adds
/// `l1_fees_to_add` to L1 fees available and grows the gas backlog
/// by `compute_gas_for_backlog`.
#[derive(Debug, Clone, Default)]
pub struct EndTxFeeDistribution {
    pub infra_fee_account: Address,
    pub infra_fee_amount: U256,
    pub network_fee_account: Address,
    pub network_fee_amount: U256,
    pub poster_fee_destination: Address,
    pub poster_fee_amount: U256,
    pub l1_fees_to_add: U256,
    pub compute_gas_for_backlog: u64,
}

/// Parameters for end-tx hook (retryable transactions).
#[derive(Debug, Clone)]
pub struct EndTxRetryableParams {
    pub gas_left: u64,
    pub gas_used: u64,
    pub effective_base_fee: U256,
    pub from: Address,
    pub refund_to: Address,
    pub max_refund: U256,
    pub submission_fee_refund: U256,
    pub ticket_id: B256,
    pub value: U256,
    pub success: bool,
    pub network_fee_account: Address,
    pub infra_fee_account: Address,
    pub min_base_fee: U256,
    pub arbos_version: u64,
    /// Multi-dimensional cost if ArbOS >= v60 (None otherwise).
    /// When set and less than single-gas cost, the difference is refunded.
    pub multi_dimensional_cost: Option<U256>,
    /// Block base fee for comparing with effective_base_fee.
    /// Multi-gas refund is skipped if effective_base_fee != block_base_fee
    /// (retryable estimation case).
    pub block_base_fee: U256,
}

/// Result from end-tx retryable hook.
///
/// The caller should:
/// - Grow gas backlog by `compute_gas_for_backlog`
/// - If `should_delete_retryable`: delete the retryable ticket
/// - If `should_return_value_to_escrow`: transfer value from `from`
///   back to `escrow_address`
#[derive(Debug, Clone)]
pub struct EndTxRetryableResult {
    pub compute_gas_for_backlog: u64,
    pub should_delete_retryable: bool,
    pub should_return_value_to_escrow: bool,
    pub escrow_address: Address,
}

/// Action to take for a reverted/filtered transaction.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum RevertedTxAction {
    /// No special handling; proceed with normal execution.
    None,
    /// Pre-recorded revert: increment nonce, consume specific gas, return revert.
    PreRecordedRevert {
        gas_to_consume: u64,
    },
    /// Filtered transaction: increment nonce, consume all remaining gas.
    FilteredTx,
}

/// Parameters for computing submit retryable fees.
#[derive(Debug, Clone)]
pub struct SubmitRetryableParams {
    pub ticket_id: B256,
    pub deposit_value: U256,
    pub retry_value: U256,
    pub gas_fee_cap: U256,
    pub gas: u64,
    pub max_submission_fee: U256,
    pub retry_data_len: usize,
    pub l1_base_fee: U256,
    pub effective_base_fee: U256,
    pub current_time: u64,
    /// From address balance after deposit minting.
    pub balance_after_mint: U256,
    pub infra_fee_account: Address,
    pub min_base_fee: U256,
    pub arbos_version: u64,
}

/// Computed fee distribution for a submit retryable transaction.
///
/// The caller should execute the following operations in order:
/// 1. Mint `deposit_value` to `from`
/// 2. Transfer `submission_fee` from `from` to network fee account
/// 3. Transfer `submission_fee_refund` from `from` to fee refund address
/// 4. Transfer `retry_value` from `from` to `escrow`
/// 5. Create retryable ticket with `timeout`
/// 6. If `can_pay_for_gas`:
///    - Transfer `infra_cost` from `from` to infra fee account
///    - Transfer `network_cost` from `from` to network fee account
///    - Transfer `gas_price_refund` from `from` to fee refund address
///    - Schedule auto-redeem with `available_refund` as max refund
/// 7. If not `can_pay_for_gas`: refund `gas_cost_refund` to fee refund address
#[derive(Debug, Clone, Default)]
pub struct SubmitRetryableFees {
    /// The actual submission fee.
    pub submission_fee: U256,
    /// Excess submission fee to refund.
    pub submission_fee_refund: U256,
    /// Escrow address for the retryable's call value.
    pub escrow: Address,
    /// Retryable ticket timeout.
    pub timeout: u64,
    /// Whether the user can pay for gas.
    pub can_pay_for_gas: bool,
    /// Total gas cost (effective_base_fee * gas).
    pub gas_cost: U256,
    /// Infra fee portion of gas cost (ArbOS v11+).
    pub infra_cost: U256,
    /// Network fee portion (gas_cost - infra_cost).
    pub network_cost: U256,
    /// Gas price refund ((gas_fee_cap - effective_base_fee) * gas).
    pub gas_price_refund: U256,
    /// If user can't pay for gas, this amount should be refunded.
    pub gas_cost_refund: U256,
    /// Remaining L1 deposit available for auto-redeem max refund.
    pub available_refund: U256,
    /// Withheld submission fee (for error path refunds).
    pub withheld_submission_fee: U256,
    /// Error if validation fails.
    pub error: Option<String>,
}

/// Standard Ethereum base transaction gas.
pub const TX_GAS: u64 = 21_000;

// =====================================================================
// Helper functions
// =====================================================================

/// Attempts to subtract up to `take` from `pool` without going negative.
/// Returns the amount actually subtracted.
pub fn take_funds(pool: &mut U256, take: U256) -> U256 {
    if *pool < take {
        let old = *pool;
        *pool = U256::ZERO;
        old
    } else {
        *pool -= take;
        take
    }
}

/// Compute poster gas given a poster cost and base fee,
/// with optional gas estimation padding.
pub fn compute_poster_gas(
    poster_cost: U256,
    base_fee: U256,
    is_gas_estimation: bool,
    min_gas_price: U256,
) -> u64 {
    if base_fee.is_zero() {
        return 0;
    }

    let adjusted_base_fee = if is_gas_estimation {
        // Assume congestion: use 7/8 of base fee
        let adjusted = base_fee * U256::from(7) / U256::from(8);
        if adjusted < min_gas_price {
            min_gas_price
        } else {
            adjusted
        }
    } else {
        base_fee
    };

    let padded_cost = if is_gas_estimation {
        poster_cost * U256::from(GAS_ESTIMATION_L1_PRICE_PADDING_BIPS) / U256::from(10000)
    } else {
        poster_cost
    };

    if adjusted_base_fee.is_zero() {
        return 0;
    }

    let gas = padded_cost / adjusted_base_fee;
    gas.try_into().unwrap_or(u64::MAX)
}

/// Calculates the poster gas cost for a transaction's calldata.
///
/// Returns (poster_gas, calldata_units) where:
/// - poster_gas: Gas that should be reserved for L1 posting costs
/// - calldata_units: The raw calldata units before price conversion
pub fn get_poster_gas(
    tx_data: &[u8],
    l1_base_fee: U256,
    l2_base_fee: U256,
    _arbos_version: u64,
) -> (u64, u64) {
    if l2_base_fee.is_zero() || l1_base_fee.is_zero() {
        return (0, 0);
    }

    let calldata_units = tx_data_non_zero_count(tx_data) * 16
        + tx_data_zero_count(tx_data) * 4;

    let l1_cost = U256::from(calldata_units) * l1_base_fee;
    let poster_gas = l1_cost / l2_base_fee;
    let poster_gas_u64: u64 = poster_gas.try_into().unwrap_or(u64::MAX);

    (poster_gas_u64, calldata_units as u64)
}

/// Refund with L1 deposit pool cap.
///
/// Takes up to `amount` from `max_refund` and transfers that to `refund_to`.
/// Any excess (amount beyond the L1 deposit) goes to `from`.
fn refund_with_pool<F>(
    refund_from: Address,
    amount: U256,
    max_refund: &mut U256,
    refund_to: Address,
    from: Address,
    transfer_fn: &mut F,
) where
    F: FnMut(Address, Address, U256) -> Result<(), ()>,
{
    let to_refund_addr = take_funds(max_refund, amount);
    if to_refund_addr > U256::ZERO {
        let _ = transfer_fn(refund_from, refund_to, to_refund_addr);
    }
    let remainder = amount.saturating_sub(to_refund_addr);
    if remainder > U256::ZERO {
        let _ = transfer_fn(refund_from, from, remainder);
    }
}

/// Compute the gas payment split between infra and network fee accounts.
///
/// Returns (infra_cost, network_cost) where gas_cost = infra_cost + network_cost.
pub fn compute_retryable_gas_split(
    gas: u64,
    effective_base_fee: U256,
    infra_fee_account: Address,
    min_base_fee: U256,
    arbos_version: u64,
) -> (U256, U256) {
    let gas_cost = effective_base_fee.saturating_mul(U256::from(gas));
    let mut network_cost = gas_cost;
    let mut infra_cost = U256::ZERO;

    if arbos_version >= arb_ver::ARBOS_VERSION_11
        && infra_fee_account != Address::ZERO
    {
        let infra_fee = min_base_fee.min(effective_base_fee);
        infra_cost = infra_fee.saturating_mul(U256::from(gas));
        infra_cost = take_funds(&mut network_cost, infra_cost);
    }

    (infra_cost, network_cost)
}

/// Compute fees for a submit retryable transaction.
///
/// This performs the pure fee computation without executing any balance
/// operations. The caller should execute the operations described in
/// the `SubmitRetryableFees` documentation.
pub fn compute_submit_retryable_fees(params: &SubmitRetryableParams) -> SubmitRetryableFees {
    let submission_fee = retryables::retryable_submission_fee(
        params.retry_data_len,
        params.l1_base_fee,
    );

    let escrow = retryables::retryable_escrow_address(params.ticket_id);
    let timeout = params.current_time + retryables::RETRYABLE_LIFETIME_SECONDS;

    // Check balance covers max submission fee.
    if params.balance_after_mint < params.max_submission_fee {
        return SubmitRetryableFees {
            submission_fee,
            escrow,
            timeout,
            error: Some(format!(
                "insufficient funds for max submission fee: have {} want {}",
                params.balance_after_mint, params.max_submission_fee,
            )),
            ..Default::default()
        };
    }

    // Check max submission fee covers actual fee.
    if params.max_submission_fee < submission_fee {
        return SubmitRetryableFees {
            submission_fee,
            escrow,
            timeout,
            error: Some(format!(
                "max submission fee {} is less than actual {}",
                params.max_submission_fee, submission_fee,
            )),
            ..Default::default()
        };
    }

    let submission_fee_refund = params.max_submission_fee.saturating_sub(submission_fee);

    // Track available refund from L1 deposit.
    let mut available_refund = params.deposit_value;
    take_funds(&mut available_refund, params.retry_value);
    let withheld_submission_fee = take_funds(&mut available_refund, submission_fee);
    take_funds(&mut available_refund, submission_fee_refund);

    // Check if user can pay for gas.
    let max_gas_cost = params.gas_fee_cap.saturating_mul(U256::from(params.gas));
    let fee_cap_too_low = params.gas_fee_cap < params.effective_base_fee;

    // Balance after all deductions so far.
    let balance_after_deductions = params.balance_after_mint
        .saturating_sub(submission_fee)
        .saturating_sub(submission_fee_refund)
        .saturating_sub(params.retry_value);

    let can_pay_for_gas = !fee_cap_too_low
        && params.gas >= TX_GAS
        && balance_after_deductions >= max_gas_cost;

    // Compute gas cost split.
    let (infra_cost, network_cost) = compute_retryable_gas_split(
        params.gas,
        params.effective_base_fee,
        params.infra_fee_account,
        params.min_base_fee,
        params.arbos_version,
    );
    let gas_cost = params.effective_base_fee.saturating_mul(U256::from(params.gas));

    // Gas cost refund if user can't pay.
    let gas_cost_refund = if !can_pay_for_gas {
        take_funds(&mut available_refund, max_gas_cost)
    } else {
        U256::ZERO
    };

    // Gas price refund (difference between fee cap and effective base fee).
    let gas_price_refund = if params.gas_fee_cap > params.effective_base_fee {
        (params.gas_fee_cap - params.effective_base_fee)
            .saturating_mul(U256::from(params.gas))
    } else {
        U256::ZERO
    };

    if can_pay_for_gas {
        // Track gas cost and gas price refund through available_refund.
        let withheld_gas_funds = take_funds(&mut available_refund, gas_cost);
        let _gas_price_taken = take_funds(&mut available_refund, gas_price_refund);
        // Add back withheld amounts for the auto-redeem's max refund.
        available_refund = available_refund
            .saturating_add(withheld_gas_funds)
            .saturating_add(withheld_submission_fee);
    }

    SubmitRetryableFees {
        submission_fee,
        submission_fee_refund,
        escrow,
        timeout,
        can_pay_for_gas,
        gas_cost,
        infra_cost,
        network_cost,
        gas_price_refund,
        gas_cost_refund,
        available_refund,
        withheld_submission_fee,
        error: None,
    }
}

fn tx_data_non_zero_count(data: &[u8]) -> usize {
    data.iter().filter(|&&b| b != 0).count()
}

fn tx_data_zero_count(data: &[u8]) -> usize {
    data.iter().filter(|&&b| b == 0).count()
}
