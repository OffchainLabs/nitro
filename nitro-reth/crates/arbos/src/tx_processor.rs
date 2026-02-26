use alloy_primitives::{Address, B256, U256};
use std::collections::HashMap;

use crate::l1_pricing;

/// ArbOS system address (0x00000000000000000000000000000000000a4b05).
pub const ARBOS_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[16] = 0x0a;
    bytes[17] = 0x4b;
    bytes[18] = 0x05;
    Address::new(bytes)
};

/// L1 pricer funds pool address (where poster fees are sent).
pub const L1_PRICER_FUNDS_POOL_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[18] = 0x00;
    bytes[19] = 0x6c;
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
}

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

fn tx_data_non_zero_count(data: &[u8]) -> usize {
    data.iter().filter(|&&b| b != 0).count()
}

fn tx_data_zero_count(data: &[u8]) -> usize {
    data.iter().filter(|&&b| b == 0).count()
}
