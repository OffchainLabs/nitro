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
    /// Cache for L1 block number (for NUMBER opcode).
    pub cached_l1_block_number: Option<u64>,
    /// Cache for L1 block hashes (for BLOCKHASH opcode).
    pub cached_l1_block_hashes: HashMap<u64, B256>,
    /// Scheduled transactions (e.g., retryable auto-redeems).
    pub scheduled_txs: Vec<Vec<u8>>,
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
            cached_l1_block_number: None,
            cached_l1_block_hashes: HashMap::new(),
            scheduled_txs: Vec::new(),
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

fn tx_data_non_zero_count(data: &[u8]) -> usize {
    data.iter().filter(|&&b| b != 0).count()
}

fn tx_data_zero_count(data: &[u8]) -> usize {
    data.iter().filter(|&&b| b == 0).count()
}
