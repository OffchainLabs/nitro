use alloy_primitives::{address, U256};
use arb_evm::hooks::{
    ArbOsHooks, EndTxContext, GasChargingContext, NoopArbOsHooks, StartTxContext,
};
use arb_primitives::tx_types::ArbTxType;

fn dummy_start_ctx() -> StartTxContext {
    StartTxContext {
        sender: address!("00000000000000000000000000000000000A11CE"),
        to: Some(address!("00000000000000000000000000000000000B0B00")),
        nonce: 0,
        gas_limit: 21_000,
        gas_price: U256::from(1u64),
        value: U256::from(100u64),
        data: vec![],
        tx_type: ArbTxType::ArbitrumLegacyTx,
        is_gas_estimation: false,
    }
}

fn dummy_gas_ctx() -> GasChargingContext {
    GasChargingContext {
        sender: address!("00000000000000000000000000000000000A11CE"),
        poster_address: address!("a4b0000000000000000000000000000073657175"),
        gas_limit: 21_000,
        intrinsic_gas: 21_000,
        gas_price: U256::from(1u64),
        base_fee: U256::from(1u64),
        tx_type: ArbTxType::ArbitrumLegacyTx,
        poster_cost: U256::ZERO,
        calldata_units: 0,
    }
}

fn dummy_end_ctx() -> EndTxContext {
    EndTxContext {
        sender: address!("00000000000000000000000000000000000A11CE"),
        gas_left: 0,
        gas_used: 21_000,
        gas_price: U256::from(1u64),
        base_fee: U256::from(1u64),
        tx_type: ArbTxType::ArbitrumLegacyTx,
        success: true,
        refund_to: address!("00000000000000000000000000000000000A11CE"),
    }
}

#[test]
fn noop_start_tx_returns_ok() {
    let mut h = NoopArbOsHooks;
    assert!(h.start_tx(&dummy_start_ctx()).is_ok());
}

#[test]
fn noop_gas_charging_returns_zeroed_result() {
    let mut h = NoopArbOsHooks;
    let result = h.gas_charging(&dummy_gas_ctx()).expect("ok");
    assert_eq!(result.poster_cost, U256::ZERO);
    assert_eq!(result.poster_gas, 0);
    assert_eq!(result.compute_hold_gas, 0);
    assert_eq!(result.calldata_units, 0);
}

#[test]
fn noop_end_tx_returns_ok() {
    let mut h = NoopArbOsHooks;
    assert!(h.end_tx(&dummy_end_ctx()).is_ok());
}

#[test]
fn noop_default_accessors() {
    let h = NoopArbOsHooks;
    assert_eq!(h.nonrefundable_gas(), 0);
    assert_eq!(h.held_gas(), 0);
    assert!(!h.drop_tip());
    assert!(!h.msg_is_non_mutating());
    assert!(h.is_calldata_pricing_increase_enabled());
}

#[test]
fn noop_scheduled_txs_empty() {
    let mut h = NoopArbOsHooks;
    assert!(h.scheduled_txs().is_empty());
}

#[test]
fn noop_gas_price_op_returns_input_unchanged() {
    let h = NoopArbOsHooks;
    let gp = U256::from(123u64);
    let bf = U256::from(456u64);
    assert_eq!(h.gas_price_op(gp, bf), gp);
}

#[test]
fn start_then_end_tx_sequence_is_idempotent_for_noop() {
    let mut h = NoopArbOsHooks;
    for _ in 0..5 {
        h.start_tx(&dummy_start_ctx()).unwrap();
        let _ = h.gas_charging(&dummy_gas_ctx()).unwrap();
        h.end_tx(&dummy_end_ctx()).unwrap();
    }
}
