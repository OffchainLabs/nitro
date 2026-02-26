use alloy_primitives::{Address, B256, U256};

use arbos::tx_processor::{
    EndTxFeeDistribution, EndTxNormalParams, GasChargingError, GasChargingParams, TxProcessor,
    get_poster_gas,
};
use arbos::util::tx_type_has_poster_costs;

use crate::hooks::{
    ArbOsHooks, EndTxContext, GasChargingContext, GasChargingResult, StartTxContext,
};

/// Concrete ArbOS hooks implementation backed by `TxProcessor`.
///
/// Bridges the `ArbOsHooks` trait to the arbos crate's `TxProcessor` which
/// contains the core fee accounting logic.
#[derive(Debug)]
pub struct DefaultArbOsHooks {
    /// Per-transaction processor state.
    pub tx_proc: TxProcessor,
    /// Current ArbOS version.
    pub arbos_version: u64,
    /// Network fee account from ArbOS state.
    pub network_fee_account: Address,
    /// Infrastructure fee account from ArbOS state.
    pub infra_fee_account: Address,
    /// Minimum L2 base fee from L2 pricing state.
    pub min_base_fee: U256,
    /// Per-block gas limit from L2 pricing state.
    pub per_block_gas_limit: u64,
    /// Per-tx gas limit from L2 pricing state (ArbOS v50+).
    pub per_tx_gas_limit: u64,
    /// Block coinbase (poster address).
    pub coinbase: Address,
    /// Whether this is an eth_call (non-mutating).
    pub is_eth_call: bool,
    /// Cached L1 base fee for poster cost computation.
    pub l1_base_fee: U256,
}

impl DefaultArbOsHooks {
    pub fn new(
        coinbase: Address,
        arbos_version: u64,
        network_fee_account: Address,
        infra_fee_account: Address,
        min_base_fee: U256,
        per_block_gas_limit: u64,
        per_tx_gas_limit: u64,
        is_eth_call: bool,
        l1_base_fee: U256,
    ) -> Self {
        Self {
            tx_proc: TxProcessor::new(coinbase),
            arbos_version,
            network_fee_account,
            infra_fee_account,
            min_base_fee,
            per_block_gas_limit,
            per_tx_gas_limit,
            coinbase,
            is_eth_call,
            l1_base_fee,
        }
    }

    /// Compute the end-of-tx fee distribution for a normal transaction.
    pub fn compute_end_tx_fees(&self, ctx: &EndTxContext) -> EndTxFeeDistribution {
        self.tx_proc.compute_end_tx_fee_distribution(&EndTxNormalParams {
            gas_used: ctx.gas_used,
            gas_price: ctx.gas_price,
            base_fee: ctx.base_fee,
            coinbase: self.coinbase,
            network_fee_account: self.network_fee_account,
            infra_fee_account: self.infra_fee_account,
            min_base_fee: self.min_base_fee,
            arbos_version: self.arbos_version,
        })
    }
}

/// Error type for ArbOS hooks.
#[derive(Debug, thiserror::Error)]
pub enum ArbHookError {
    #[error("gas charging: {0}")]
    GasCharging(#[from] GasChargingError),
    #[error("state access: {0}")]
    StateAccess(String),
}

impl ArbOsHooks for DefaultArbOsHooks {
    type Error = ArbHookError;

    fn start_tx(&mut self, ctx: &StartTxContext) -> Result<(), Self::Error> {
        self.tx_proc.set_tx_type(ctx.tx_type as u8);
        Ok(())
    }

    fn gas_charging(&mut self, ctx: &GasChargingContext) -> Result<GasChargingResult, Self::Error> {
        let mut gas_remaining = ctx.gas_limit.saturating_sub(ctx.intrinsic_gas);

        let skip_l1_charging = !tx_type_has_poster_costs(ctx.tx_type.as_u8());
        let poster_cost = if skip_l1_charging {
            U256::ZERO
        } else {
            // Compute poster cost from calldata and L1 base fee.
            let (poster_gas, _calldata_units) = get_poster_gas(
                &ctx.tx_data,
                ctx.l1_base_fee,
                ctx.base_fee,
                self.arbos_version,
            );
            ctx.l1_base_fee.saturating_mul(U256::from(poster_gas))
        };

        let params = GasChargingParams {
            base_fee: ctx.base_fee,
            poster_cost,
            is_gas_estimation: false,
            is_eth_call: self.is_eth_call,
            skip_l1_charging,
            min_base_fee: self.min_base_fee,
            per_block_gas_limit: self.per_block_gas_limit,
            per_tx_gas_limit: self.per_tx_gas_limit,
            arbos_version: self.arbos_version,
        };

        self.tx_proc.gas_charging_hook(&mut gas_remaining, ctx.intrinsic_gas, &params)?;

        Ok(GasChargingResult {
            poster_cost: self.tx_proc.poster_fee,
            poster_gas: self.tx_proc.poster_gas,
            compute_hold_gas: self.tx_proc.compute_hold_gas,
        })
    }

    fn end_tx(&mut self, _ctx: &EndTxContext) -> Result<(), Self::Error> {
        // Fee distribution and backlog update are handled by the block executor
        // using compute_end_tx_fees(). The hooks trait just signals completion.
        Ok(())
    }

    fn nonrefundable_gas(&self) -> u64 {
        self.tx_proc.nonrefundable_gas()
    }

    fn held_gas(&self) -> u64 {
        self.tx_proc.held_gas()
    }

    fn scheduled_txs(&mut self) -> Vec<Vec<u8>> {
        core::mem::take(&mut self.tx_proc.scheduled_txs)
    }

    fn l1_block_number(&self) -> Result<u64, Self::Error> {
        self.tx_proc
            .cached_l1_block_number
            .ok_or_else(|| ArbHookError::StateAccess("L1 block number not cached".into()))
    }

    fn l1_block_hash(&self, block_number: u64) -> Result<B256, Self::Error> {
        self.tx_proc
            .cached_l1_block_hashes
            .get(&block_number)
            .copied()
            .ok_or_else(|| {
                ArbHookError::StateAccess(format!("L1 block hash not cached for {block_number}"))
            })
    }

    fn drop_tip(&self) -> bool {
        self.tx_proc.drop_tip(self.arbos_version)
    }

    fn gas_price_op(&self, gas_price: U256, base_fee: U256) -> U256 {
        self.tx_proc.gas_price_op(self.arbos_version, base_fee, gas_price)
    }

    fn msg_is_non_mutating(&self) -> bool {
        self.is_eth_call
    }

    fn is_calldata_pricing_increase_enabled(&self) -> bool {
        // Feature gated by ArbOS state. Default to true for recent versions.
        true
    }
}

