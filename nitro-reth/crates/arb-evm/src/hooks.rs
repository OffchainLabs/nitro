use alloy_primitives::{Address, B256, U256};

use arb_primitives::tx_types::ArbTxType;

/// Context passed to ArbOS hooks at the start of transaction execution.
#[derive(Debug, Clone)]
pub struct StartTxContext {
    pub sender: Address,
    pub to: Option<Address>,
    pub nonce: u64,
    pub gas_limit: u64,
    pub gas_price: U256,
    pub value: U256,
    pub data: Vec<u8>,
    pub tx_type: ArbTxType,
    pub is_gas_estimation: bool,
}

/// Context passed to the gas charging hook.
#[derive(Debug, Clone)]
pub struct GasChargingContext {
    pub sender: Address,
    pub poster_address: Address,
    pub gas_limit: u64,
    pub intrinsic_gas: u64,
    pub gas_price: U256,
    pub base_fee: U256,
    pub tx_type: ArbTxType,
}

/// Result from gas charging.
#[derive(Debug, Clone, Default)]
pub struct GasChargingResult {
    pub poster_cost: U256,
    pub poster_gas: u64,
    pub compute_hold_gas: u64,
}

/// Context passed to the end-of-transaction hook.
#[derive(Debug, Clone)]
pub struct EndTxContext {
    pub sender: Address,
    pub gas_left: u64,
    pub gas_used: u64,
    pub gas_price: U256,
    pub base_fee: U256,
    pub tx_type: ArbTxType,
    pub success: bool,
    pub refund_to: Address,
}

/// Hooks for ArbOS-specific transaction processing.
///
/// These hooks integrate ArbOS state management into reth's block execution.
/// In Go Nitro, this corresponds to `TxProcessor`'s `StartTxHook`,
/// `GasChargingHook`, and `EndTxHook`.
pub trait ArbOsHooks {
    type Error: core::fmt::Debug;

    /// Called before each transaction. Sets up gas accounting,
    /// processes deposits, and initializes retryable state.
    fn start_tx(&mut self, ctx: &StartTxContext) -> Result<(), Self::Error>;

    /// Called after intrinsic gas calculation. Charges poster costs
    /// and manages L1 pricing.
    fn gas_charging(&mut self, ctx: &GasChargingContext) -> Result<GasChargingResult, Self::Error>;

    /// Called after transaction execution. Handles gas refunds,
    /// poster fee distribution, and state cleanup.
    fn end_tx(&mut self, ctx: &EndTxContext) -> Result<(), Self::Error>;

    /// Returns the amount of gas that cannot be refunded.
    fn nonrefundable_gas(&self) -> u64;

    /// Returns the amount of gas held for compute.
    fn held_gas(&self) -> u64;

    /// Returns scheduled internal transactions generated during execution.
    fn scheduled_txs(&mut self) -> Vec<Vec<u8>>;

    /// Returns the cached L1 block number.
    fn l1_block_number(&self) -> Result<u64, Self::Error>;

    /// Returns an L1 block hash.
    fn l1_block_hash(&self, block_number: u64) -> Result<B256, Self::Error>;
}

/// No-op implementation for testing.
pub struct NoopArbOsHooks;

impl ArbOsHooks for NoopArbOsHooks {
    type Error = ();

    fn start_tx(&mut self, _ctx: &StartTxContext) -> Result<(), ()> {
        Ok(())
    }

    fn gas_charging(&mut self, _ctx: &GasChargingContext) -> Result<GasChargingResult, ()> {
        Ok(GasChargingResult::default())
    }

    fn end_tx(&mut self, _ctx: &EndTxContext) -> Result<(), ()> {
        Ok(())
    }

    fn nonrefundable_gas(&self) -> u64 {
        0
    }

    fn held_gas(&self) -> u64 {
        0
    }

    fn scheduled_txs(&mut self) -> Vec<Vec<u8>> {
        vec![]
    }

    fn l1_block_number(&self) -> Result<u64, ()> {
        Ok(0)
    }

    fn l1_block_hash(&self, _block_number: u64) -> Result<B256, ()> {
        Ok(B256::ZERO)
    }
}
