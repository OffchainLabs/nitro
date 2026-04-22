//! debug_trace* conformance markers for Arbitrum tx types.
//!
//! reth's default `DebugApi` (registered automatically via `RpcAddOns`)
//! handles tracing generically: it consumes any `SignedTx` implementing
//! the `Transaction` trait and runs the EVM inspector over the resulting
//! TxEnv. Our `ArbTransactionSigned` implements `Transaction` for all
//! variants (0x64-0x6A), and `ArbEvmConfig::tx_env()` produces the
//! correct `ArbTransaction` envelope, so the out-of-box debug_trace
//! implementation works for the EVM-level call/return/opcode/log events
//! of any Arbitrum transaction.
//!
//! What is *not* captured by reth's default:
//!   - Stylus WASM host I/O (getBytes32, setTrieSlots, contract_call…), which run inside the wasmer
//!     runtime rather than the EVM.
//!   - ArbOS-precompile internal state reads (these appear as opaque CALL → 0x64..0x73 with SLOAD
//!     counts but no per-field detail).
//!
//! Tests below are smoke-level markers: they verify the Transaction
//! trait exposes the fields the tracer reads. Full integration tests
//! require a running node and live against the e2e harness.

use alloy_consensus::{Transaction, TxLegacy};
use alloy_primitives::{address, Bytes, Signature, TxKind, B256, U256};
use arb_alloy_consensus::tx::{
    ArbContractTx, ArbDepositTx, ArbInternalTx, ArbRetryTx, ArbSubmitRetryableTx, ArbUnsignedTx,
};
use arb_primitives::{ArbTransactionSigned, ArbTypedTransaction};

fn zero_sig() -> Signature {
    Signature::new(U256::ZERO, U256::ZERO, false)
}

fn signed(inner: ArbTypedTransaction) -> ArbTransactionSigned {
    ArbTransactionSigned::new_unhashed(inner, zero_sig())
}

/// Baseline: tracer needs to read `to()`, `value()`, `input()`, `gas_limit()`.
/// All Arb tx types must expose these. If any variant returns None / 0
/// when there's a real target, debug_trace will produce an empty trace.

#[test]
fn deposit_tx_exposes_to_value_for_tracing() {
    let to = address!("dddddddddddddddddddddddddddddddddddddddd");
    let value = U256::from(1_000u64);
    let tx = signed(ArbTypedTransaction::Deposit(ArbDepositTx {
        chain_id: U256::from(42161u64),
        l1_request_id: B256::ZERO,
        from: address!("1111111111111111111111111111111111111111"),
        to,
        value,
    }));
    assert_eq!(tx.kind(), TxKind::Call(to));
    assert_eq!(tx.value(), value);
    assert!(tx.input().is_empty());
}

#[test]
fn contract_tx_exposes_data_for_tracing() {
    let tx = signed(ArbTypedTransaction::Contract(ArbContractTx {
        chain_id: U256::from(42161u64),
        request_id: B256::ZERO,
        from: address!("c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0"),
        gas_fee_cap: U256::from(1u64),
        gas: 100_000,
        to: Some(address!("2222222222222222222222222222222222222222")),
        value: U256::ZERO,
        data: Bytes::from(vec![0xFF; 4]),
    }));
    assert_eq!(tx.input().as_ref(), &[0xFFu8; 4]);
    assert!(matches!(tx.kind(), TxKind::Call(_)));
    assert_eq!(tx.gas_limit(), 100_000);
}

#[test]
fn retry_tx_exposes_kind_and_gas_limit() {
    let tx = signed(ArbTypedTransaction::Retry(ArbRetryTx {
        chain_id: U256::from(42161u64),
        nonce: 0,
        from: address!("1111111111111111111111111111111111111111"),
        gas_fee_cap: U256::from(1u64),
        gas: 500_000,
        to: Some(address!("2222222222222222222222222222222222222222")),
        value: U256::from(10u64),
        data: Bytes::new(),
        ticket_id: B256::ZERO,
        refund_to: address!("3333333333333333333333333333333333333333"),
        max_refund: U256::ZERO,
        submission_fee_refund: U256::ZERO,
    }));
    assert_eq!(tx.gas_limit(), 500_000);
    assert!(matches!(tx.kind(), TxKind::Call(_)));
}

#[test]
fn submit_retryable_tx_input_wraps_retry_data_with_selector() {
    // SubmitRetryable's `input()` prepends the
    // `submitRetryable(...)` selector so debug_trace sees the canonical
    // ArbRetryableTx precompile calldata. Verify the non-empty wrapper
    // contains the user's retry_data in its tail.
    let data = Bytes::from(vec![0xDE, 0xAD, 0xBE, 0xEF]);
    let tx = signed(ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
        chain_id: U256::from(42161u64),
        request_id: B256::ZERO,
        from: address!("1111111111111111111111111111111111111111"),
        l1_base_fee: U256::ZERO,
        deposit_value: U256::ZERO,
        gas_fee_cap: U256::from(1u64),
        gas: 100_000,
        retry_to: Some(address!("2222222222222222222222222222222222222222")),
        retry_value: U256::ZERO,
        beneficiary: address!("3333333333333333333333333333333333333333"),
        max_submission_fee: U256::ZERO,
        fee_refund_addr: address!("3333333333333333333333333333333333333333"),
        retry_data: data.clone(),
    }));
    let input = tx.input();
    assert!(input.len() >= 4, "input must have a selector prefix");
    // retry_data appears at the end of the ABI-encoded wrapper.
    let tail = &input[input.len() - data.len()..];
    assert_eq!(tail, data.as_ref());
}

#[test]
fn internal_tx_exposes_data_as_calldata() {
    let data = Bytes::from(vec![0x11, 0x22, 0x33]);
    let tx = signed(ArbTypedTransaction::Internal(ArbInternalTx {
        chain_id: U256::from(42161u64),
        data: data.clone(),
    }));
    assert_eq!(tx.input().as_ref(), data.as_ref());
}

#[test]
fn unsigned_tx_exposes_full_call_for_tracing() {
    let tx = signed(ArbTypedTransaction::Unsigned(ArbUnsignedTx {
        chain_id: U256::from(42161u64),
        from: address!("1111111111111111111111111111111111111111"),
        nonce: 5,
        gas_fee_cap: U256::from(1u64),
        gas: 21_000,
        to: Some(address!("2222222222222222222222222222222222222222")),
        value: U256::from(100u64),
        data: Bytes::new(),
    }));
    assert_eq!(tx.nonce(), 5);
    assert_eq!(tx.gas_limit(), 21_000);
    assert_eq!(tx.value(), U256::from(100u64));
}

#[test]
fn legacy_tx_exposes_standard_fields() {
    let t = TxLegacy {
        chain_id: Some(1),
        nonce: 0,
        gas_price: 1_000_000_000,
        gas_limit: 21_000,
        to: TxKind::Call(address!("2222222222222222222222222222222222222222")),
        value: U256::ZERO,
        input: Bytes::new(),
    };
    let tx = signed(ArbTypedTransaction::Legacy(t));
    assert_eq!(tx.gas_limit(), 21_000);
}

// Stylus host I/O tracing TODO: requires custom tracer hooked into the
// wasmer runtime. See Nitro's stylusTracer for reference format.
#[test]
#[ignore = "TODO: Stylus WASM host-I/O tracing — requires wasmer-level tracer"]
fn stylus_program_hostio_capture_emits_per_call_record() {}

// debug_traceTransaction integration test skeleton. Requires a running
// node with a persisted block. TODO: wire via ArbosHarness + a test
// helper that constructs a DebugApi against an in-memory provider.
#[test]
#[ignore = "TODO: integration test — requires node stack"]
fn debug_trace_transaction_returns_expected_opcodes() {}
