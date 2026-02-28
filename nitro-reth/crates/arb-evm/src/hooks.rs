use alloy_primitives::{Address, U256};

use arb_primitives::multigas::MultiGas;
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
    /// Pre-computed poster cost in ETH (price_per_unit * brotli_units).
    /// Computed by the block executor using L1PricingState with brotli compression.
    pub poster_cost: U256,
    /// Pre-computed calldata units for L1 pricing state tracking.
    pub calldata_units: u64,
}

/// Result from gas charging.
#[derive(Debug, Clone, Default)]
pub struct GasChargingResult {
    pub poster_cost: U256,
    pub poster_gas: u64,
    pub compute_hold_gas: u64,
    /// Calldata units to add to L1 pricing state's units_since_update.
    pub calldata_units: u64,
    /// Multi-dimensional gas consumed during gas charging (L1 calldata component).
    pub multi_gas: MultiGas,
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

    /// Whether the priority fee tip should be dropped (not sent to coinbase).
    fn drop_tip(&self) -> bool;

    /// The effective gas price for the GASPRICE opcode.
    fn gas_price_op(&self, gas_price: U256, base_fee: U256) -> U256;

    /// Whether the message is non-mutating (eth_call).
    fn msg_is_non_mutating(&self) -> bool;

    /// Whether EIP-7623 calldata pricing increase is enabled.
    fn is_calldata_pricing_increase_enabled(&self) -> bool;
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

    fn drop_tip(&self) -> bool {
        false
    }

    fn gas_price_op(&self, gas_price: U256, _base_fee: U256) -> U256 {
        gas_price
    }

    fn msg_is_non_mutating(&self) -> bool {
        false
    }

    fn is_calldata_pricing_increase_enabled(&self) -> bool {
        true
    }
}
