//! Block-level bookkeeping for block production, mirroring Go's `blockBuildState`
//! (`arbos/block_processor.go`).

use arb_chainspec::arbos_version::{ARBOS_VERSION_50, ARBOS_VERSION_FIX_REDEEM_GAS};

/// Standard Ethereum transaction gas; the compute-charging floor.
pub const TX_GAS: u64 = 21_000;

/// Per-block compute budget for block gas rate limiting. Caps the compute portion of gas: total
/// minus the L1 poster component. Distinct from the header gas limit.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ComputeBudget {
    gas_left: u64,
    user_txs_processed: u64,
}

impl ComputeBudget {
    pub fn new(per_block_gas_limit: u64) -> Self {
        Self {
            gas_left: per_block_gas_limit,
            user_txs_processed: 0,
        }
    }

    pub fn gas_left(&self) -> u64 {
        self.gas_left
    }

    /// User transactions processed so far (internal, deposit, submit-retryable and retry txs are
    /// not user txs).
    pub fn user_txs_processed(&self) -> u64 {
        self.user_txs_processed
    }

    /// Whether the block is out of compute gas for user transactions.
    pub fn exhausted_for_user_tx(&self) -> bool {
        self.gas_left < TX_GAS
    }

    /// For ArbOS < 50: rejects a user tx whose compute gas (gas limit minus poster gas, floored at
    /// [`TX_GAS`]) exceeds the remaining budget — except the block's first user tx.
    /// ArbOS >= 50 clamps per-tx gas in the gas charging hook instead.
    pub fn rejects_compute_gas(
        &self,
        arbos_version: u64,
        tx_gas_limit: u64,
        poster_gas: u64,
    ) -> bool {
        if arbos_version >= ARBOS_VERSION_50 || self.user_txs_processed == 0 {
            return false;
        }
        let compute_gas = tx_gas_limit.saturating_sub(poster_gas).max(TX_GAS);
        compute_gas > self.gas_left
    }

    /// Charges a committed transaction and counts it; returns the compute gas charged.
    pub fn charge(
        &mut self,
        arbos_version: u64,
        gas_used: u64,
        data_gas: u64,
        scheduled_retry_gas: &[u64],
        is_user_tx: bool,
    ) -> u64 {
        let adjusted = adjust_for_scheduled_retries(arbos_version, gas_used, scheduled_retry_gas);
        let compute_used = compute_used(adjusted, data_gas);
        self.gas_left = self.gas_left.saturating_sub(compute_used);
        if is_user_tx {
            self.user_txs_processed += 1;
        }
        compute_used
    }

    /// Charges a failed transaction ([`TX_GAS`]) and counts it.
    pub fn charge_failed_tx(&mut self, is_user_tx: bool) {
        self.gas_left = self.gas_left.saturating_sub(TX_GAS);
        if is_user_tx {
            self.user_txs_processed += 1;
        }
    }
}

/// From ArbOS >= FixRedeemGas, gas reserved by scheduled retry txs is subtracted from a tx's gas
/// used — it is charged when the retry itself executes.
fn adjust_for_scheduled_retries(
    arbos_version: u64,
    gas_used: u64,
    scheduled_retry_gas: &[u64],
) -> u64 {
    if arbos_version < ARBOS_VERSION_FIX_REDEEM_GAS {
        return gas_used;
    }
    scheduled_retry_gas
        .iter()
        .fold(gas_used, |gas, retry| gas.saturating_sub(*retry))
}

/// The compute portion of a tx's gas: `gas_used - data_gas`, floored at [`TX_GAS`].
fn compute_used(gas_used: u64, data_gas: u64) -> u64 {
    gas_used.saturating_sub(data_gas).max(TX_GAS)
}
