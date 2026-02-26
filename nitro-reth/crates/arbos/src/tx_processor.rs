use alloy_primitives::{Address, U256};

/// ArbOS system address (0x00000000000000000000000000000000000a4b05).
pub const ARBOS_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[16] = 0x0a;
    bytes[17] = 0x4b;
    bytes[18] = 0x05;
    Address::new(bytes)
};

/// Padding applied to L1 gas price estimates for safety margin.
pub const GAS_ESTIMATION_L1_PRICE_PADDING: U256 = U256::from_limbs([11, 0, 0, 0]); // 11/10 = 10% padding

/// Per-transaction state for processing Arbitrum transactions.
///
/// In reth, this maps to the block executor's per-transaction state.
/// The Go implementation uses geth hooks (StartTxHook, EndTxHook, etc.)
/// but in reth we integrate this into the BlockExecutor implementation.
#[derive(Debug)]
pub struct TxProcessorState {
    /// The current L1 block number (for ARBBLOCKNUM opcode).
    pub l1_block_number: u64,
    /// The current L1 block hash (for ARBLOCKHASH opcode).
    pub l1_block_hash: [u8; 32],
    /// Gas held aside for L1 calldata costs.
    pub poster_gas: u64,
    /// The poster's fee contribution.
    pub poster_fee: U256,
    /// The network fee (burned/sent to fee address).
    pub network_fee: U256,
    /// The infrastructure fee.
    pub infra_fee: U256,
    /// Whether the message is non-mutating.
    pub msg_is_non_mutating: bool,
    /// Scheduled transactions (e.g., retryable auto-redeems).
    pub scheduled_txs: Vec<Vec<u8>>,
    /// Calldata units for L1 pricing.
    pub calldata_units: u64,
}

impl Default for TxProcessorState {
    fn default() -> Self {
        Self {
            l1_block_number: 0,
            l1_block_hash: [0u8; 32],
            poster_gas: 0,
            poster_fee: U256::ZERO,
            network_fee: U256::ZERO,
            infra_fee: U256::ZERO,
            msg_is_non_mutating: false,
            scheduled_txs: Vec::new(),
            calldata_units: 0,
        }
    }
}

impl TxProcessorState {
    pub fn new() -> Self {
        Self::default()
    }
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
