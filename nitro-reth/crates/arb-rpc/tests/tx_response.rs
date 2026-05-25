//! Golden tests for [`arb_rpc::response::ArbRpcTxConverter`].
//!
//! Locks the exact `WithOtherFields<Transaction>` shape per `ArbTxType`:
//! the inner Transaction matches the consensus payload, and the `other`
//! map carries the Arbitrum-specific fields extracted by `arb_tx_fields`.

use alloy_consensus::TxLegacy;
use alloy_primitives::{address, Bytes, Signature, TxKind, B256, U256};
use alloy_rpc_types_eth::TransactionInfo;
use arb_alloy_consensus::tx::{
    ArbContractTx, ArbDepositTx, ArbInternalTx, ArbRetryTx, ArbSubmitRetryableTx, ArbUnsignedTx,
};
use arb_primitives::{ArbTransactionSigned, ArbTypedTransaction};
use arb_rpc::response::ArbRpcTxConverter;
use reth_rpc_convert::transaction::RpcTxConverter;

const SIGNER: alloy_primitives::Address = address!("0000000000000000000000000000000000000aaa");

fn info() -> TransactionInfo {
    TransactionInfo {
        hash: Some(B256::repeat_byte(0xAA)),
        index: Some(0),
        block_hash: Some(B256::repeat_byte(0xBB)),
        block_number: Some(42),
        base_fee: Some(1_000_000_000),
    }
}

fn convert(
    tx: ArbTransactionSigned,
) -> alloy_serde::WithOtherFields<alloy_rpc_types_eth::Transaction<ArbTransactionSigned>> {
    ArbRpcTxConverter
        .convert_rpc_tx(tx, SIGNER, info())
        .expect("convert ok")
}

#[test]
fn legacy_tx_emits_no_extra_fields() {
    let tx = TxLegacy {
        chain_id: Some(42161),
        nonce: 0,
        gas_price: 1_000_000_000,
        gas_limit: 21_000,
        to: TxKind::Call(address!("00000000000000000000000000000000000000bb")),
        value: U256::ZERO,
        input: Bytes::new(),
    };
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Legacy(tx),
        Signature::test_signature(),
    );
    let out = convert(signed);
    assert!(out.other.is_empty());
    assert_eq!(out.inner.block_number, Some(42));
}

#[test]
fn deposit_tx_emits_request_id() {
    let req_id = B256::repeat_byte(0x42);
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Deposit(ArbDepositTx {
            chain_id: U256::from(42161u64),
            l1_request_id: req_id,
            from: SIGNER,
            to: address!("00000000000000000000000000000000000000bb"),
            value: U256::from(1_000u64),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    let got: B256 = serde_json::from_value(out.other.get("requestId").unwrap().clone()).unwrap();
    assert_eq!(got, req_id);
}

#[test]
fn contract_tx_emits_request_id() {
    let req_id = B256::repeat_byte(0x77);
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Contract(ArbContractTx {
            chain_id: U256::from(1u64),
            request_id: req_id,
            from: SIGNER,
            gas_fee_cap: U256::from(1u64),
            gas: 100_000,
            to: None,
            value: U256::ZERO,
            data: Bytes::new(),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    let got: B256 = serde_json::from_value(out.other.get("requestId").unwrap().clone()).unwrap();
    assert_eq!(got, req_id);
}

#[test]
fn retry_tx_emits_all_four_arb_fields() {
    let ticket = B256::repeat_byte(0xAB);
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Retry(ArbRetryTx {
            chain_id: U256::from(42161u64),
            nonce: 0,
            from: SIGNER,
            gas_fee_cap: U256::from(1u64),
            gas: 100_000,
            to: None,
            value: U256::ZERO,
            data: Bytes::new(),
            ticket_id: ticket,
            refund_to: SIGNER,
            max_refund: U256::from(1_000u64),
            submission_fee_refund: U256::from(500u64),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    assert_eq!(out.other.len(), 4);
    for key in ["ticketId", "refundTo", "maxRefund", "submissionFeeRefund"] {
        assert!(out.other.contains_key(key), "missing {key}");
    }
}

#[test]
fn submit_retryable_tx_emits_full_field_set() {
    let req_id = B256::repeat_byte(0x12);
    let retry_to = address!("dddddddddddddddddddddddddddddddddddddddd");
    let beneficiary = address!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
            chain_id: U256::from(42161u64),
            request_id: req_id,
            from: SIGNER,
            l1_base_fee: U256::from(1_000_000u64),
            deposit_value: U256::from(10u64).pow(U256::from(18u64)),
            gas_fee_cap: U256::from(1u64),
            gas: 100_000,
            retry_to: Some(retry_to),
            retry_value: U256::from(1u64),
            beneficiary,
            max_submission_fee: U256::from(100u64),
            fee_refund_addr: beneficiary,
            retry_data: Bytes::from(vec![0xDE, 0xAD]),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    for key in [
        "requestId",
        "l1BaseFee",
        "depositValue",
        "retryTo",
        "retryValue",
        "beneficiary",
        "maxSubmissionFee",
        "refundTo",
        "retryData",
    ] {
        assert!(out.other.contains_key(key), "missing {key}");
    }
}

#[test]
fn submit_retryable_omits_retry_to_when_none() {
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
            chain_id: U256::from(42161u64),
            request_id: B256::ZERO,
            from: SIGNER,
            l1_base_fee: U256::ZERO,
            deposit_value: U256::ZERO,
            gas_fee_cap: U256::ZERO,
            gas: 0,
            retry_to: None,
            retry_value: U256::ZERO,
            beneficiary: SIGNER,
            max_submission_fee: U256::ZERO,
            fee_refund_addr: SIGNER,
            retry_data: Bytes::new(),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    assert!(!out.other.contains_key("retryTo"));
    assert!(out.other.contains_key("requestId"));
}

#[test]
fn internal_tx_emits_no_extra_fields() {
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(42161u64),
            data: Bytes::new(),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    assert!(out.other.is_empty());
}

#[test]
fn unsigned_tx_emits_no_extra_fields() {
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Unsigned(ArbUnsignedTx {
            chain_id: U256::from(42161u64),
            from: SIGNER,
            nonce: 0,
            gas_fee_cap: U256::ZERO,
            gas: 0,
            to: None,
            value: U256::ZERO,
            data: Bytes::new(),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    assert!(out.other.is_empty());
}

#[test]
fn converter_preserves_signer_address() {
    let signed = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(42161u64),
            data: Bytes::new(),
        }),
        Signature::test_signature(),
    );
    let out = convert(signed);
    assert_eq!(out.inner.inner.signer(), SIGNER);
}
