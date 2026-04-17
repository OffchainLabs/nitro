//! Arbitrum RPC conformance tests.
//!
//! Verifies that arb-rpc emits the Arbitrum-specific fields the protocol
//! requires, and that field extraction (mix_hash layout, extra_data,
//! tx-specific fields) matches the canonical Nitro layout.

use alloy_consensus::{Header, TxEip1559, TxEip2930, TxLegacy};
use alloy_primitives::{address, Bytes, TxKind, B256, U256};
use alloy_serde::WithOtherFields;
use arb_alloy_consensus::tx::{
    ArbContractTx, ArbDepositTx, ArbInternalTx, ArbRetryTx, ArbSubmitRetryableTx, ArbUnsignedTx,
};
use arb_primitives::ArbTypedTransaction;
use arb_rpc::header::{l1_block_number_from_mix_hash, ArbHeaderConverter};
use arb_rpc::response::arb_tx_fields;
use reth_primitives_traits::SealedHeader;
use reth_rpc_convert::transaction::HeaderConverter;

// ======================================================================
// Mix hash extraction (Arbitrum encodes l1BlockNumber + sendCount here)
// ======================================================================

#[test]
fn l1_block_number_extracted_from_bytes_8_15() {
    let mut mix = [0u8; 32];
    mix[8..16].copy_from_slice(&42u64.to_be_bytes());
    assert_eq!(l1_block_number_from_mix_hash(&B256::from(mix)), 42);
}

#[test]
fn l1_block_number_ignores_other_bytes() {
    let mut mix = [0u8; 32];
    mix[0..8].copy_from_slice(&99u64.to_be_bytes());
    mix[8..16].copy_from_slice(&5u64.to_be_bytes());
    mix[16..24].copy_from_slice(&77u64.to_be_bytes());
    assert_eq!(l1_block_number_from_mix_hash(&B256::from(mix)), 5);
}

#[test]
fn l1_block_number_zero_mix_hash_returns_zero() {
    assert_eq!(l1_block_number_from_mix_hash(&B256::ZERO), 0);
}

// ======================================================================
// Header converter emits sendRoot, sendCount, l1BlockNumber
// ======================================================================

fn mk_header(send_count: u64, l1_block: u64, send_root: B256) -> SealedHeader<Header> {
    let mut mix = [0u8; 32];
    mix[0..8].copy_from_slice(&send_count.to_be_bytes());
    mix[8..16].copy_from_slice(&l1_block.to_be_bytes());
    let mut extra = vec![0u8; 32];
    extra[..32].copy_from_slice(send_root.as_slice());
    let header = Header {
        mix_hash: B256::from(mix),
        extra_data: extra.into(),
        ..Default::default()
    };
    SealedHeader::seal_slow(header)
}

#[test]
fn header_converter_exposes_send_count_hex() {
    let h = mk_header(0x2A, 0, B256::ZERO);
    let out: WithOtherFields<_> = ArbHeaderConverter.convert_header(h, 0).unwrap();
    assert_eq!(
        out.other.get("sendCount").and_then(|v| v.as_str()),
        Some("0x2a")
    );
}

#[test]
fn header_converter_exposes_l1_block_number_hex() {
    let h = mk_header(0, 0x123, B256::ZERO);
    let out: WithOtherFields<_> = ArbHeaderConverter.convert_header(h, 0).unwrap();
    assert_eq!(
        out.other.get("l1BlockNumber").and_then(|v| v.as_str()),
        Some("0x123")
    );
}

#[test]
fn header_converter_exposes_send_root_from_extra_data() {
    let root = B256::repeat_byte(0xAB);
    let h = mk_header(0, 0, root);
    let out: WithOtherFields<_> = ArbHeaderConverter.convert_header(h, 0).unwrap();
    let got: B256 = serde_json::from_value(out.other.get("sendRoot").unwrap().clone()).unwrap();
    assert_eq!(got, root);
}

#[test]
fn header_converter_send_root_zero_if_extra_too_short() {
    let header = Header {
        extra_data: vec![0xAA; 16].into(),
        ..Default::default()
    };
    let sealed = SealedHeader::seal_slow(header);
    let out: WithOtherFields<_> = ArbHeaderConverter.convert_header(sealed, 0).unwrap();
    let got: B256 = serde_json::from_value(out.other.get("sendRoot").unwrap().clone()).unwrap();
    assert_eq!(got, B256::ZERO);
}

#[test]
fn header_converter_emits_all_three_required_fields() {
    let h = mk_header(1, 2, B256::repeat_byte(3));
    let out: WithOtherFields<_> = ArbHeaderConverter.convert_header(h, 0).unwrap();
    assert!(out.other.contains_key("sendRoot"));
    assert!(out.other.contains_key("sendCount"));
    assert!(out.other.contains_key("l1BlockNumber"));
}

// ======================================================================
// arb_tx_fields emits Arbitrum-specific fields per tx type
// ======================================================================


#[test]
fn deposit_tx_emits_request_id_field() {
    let req_id = B256::repeat_byte(0x42);
    let tx = ArbTypedTransaction::Deposit(ArbDepositTx {
        chain_id: U256::from(421614u64),
        l1_request_id: req_id,
        from: address!("1111111111111111111111111111111111111111"),
        to: address!("2222222222222222222222222222222222222222"),
        value: U256::from(1_000u64),
    });
    let fields = arb_tx_fields(&tx);
    assert_eq!(fields.len(), 1);
    let got: B256 = serde_json::from_value(fields.get("requestId").unwrap().clone()).unwrap();
    assert_eq!(got, req_id);
}

#[test]
fn contract_tx_emits_request_id_field() {
    let req_id = B256::repeat_byte(0x77);
    let tx = ArbTypedTransaction::Contract(ArbContractTx {
        chain_id: U256::from(1u64),
        request_id: req_id,
        from: address!("c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0"),
        gas_fee_cap: U256::from(1u64),
        gas: 100_000,
        to: None,
        value: U256::ZERO,
        data: Bytes::new(),
    });
    let fields = arb_tx_fields(&tx);
    assert_eq!(fields.len(), 1);
    let got: B256 = serde_json::from_value(fields.get("requestId").unwrap().clone()).unwrap();
    assert_eq!(got, req_id);
}

#[test]
fn retry_tx_emits_ticket_refund_max_and_submission_fee_refund() {
    let ticket = B256::repeat_byte(0xAB);
    let refund_to = address!("00000000000000000000000000000000000B0B00");
    let tx = ArbTypedTransaction::Retry(ArbRetryTx {
        chain_id: U256::from(42161u64),
        nonce: 0,
        from: address!("1111111111111111111111111111111111111111"),
        gas_fee_cap: U256::from(1u64),
        gas: 100_000,
        to: None,
        value: U256::ZERO,
        data: Bytes::new(),
        ticket_id: ticket,
        refund_to,
        max_refund: U256::from(1_000u64),
        submission_fee_refund: U256::from(500u64),
    });
    let f = arb_tx_fields(&tx);
    assert_eq!(f.len(), 4);
    let got_ticket: B256 = serde_json::from_value(f.get("ticketId").unwrap().clone()).unwrap();
    assert_eq!(got_ticket, ticket);
    let got_max: U256 = serde_json::from_value(f.get("maxRefund").unwrap().clone()).unwrap();
    assert_eq!(got_max, U256::from(1_000u64));
    let got_sub: U256 =
        serde_json::from_value(f.get("submissionFeeRefund").unwrap().clone()).unwrap();
    assert_eq!(got_sub, U256::from(500u64));
}

#[test]
fn submit_retryable_emits_all_ten_arbitrum_fields() {
    let req_id = B256::repeat_byte(0x12);
    let retry_to = address!("dddddddddddddddddddddddddddddddddddddddd");
    let beneficiary = address!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
    let tx = ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
        chain_id: U256::from(42161u64),
        request_id: req_id,
        from: address!("1111111111111111111111111111111111111111"),
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
    });
    let f = arb_tx_fields(&tx);
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
        assert!(f.contains_key(key), "missing field: {key}");
    }
}

#[test]
fn submit_retryable_omits_retry_to_when_none() {
    let tx = ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
        chain_id: U256::from(42161u64),
        request_id: B256::ZERO,
        from: address!("1111111111111111111111111111111111111111"),
        l1_base_fee: U256::ZERO,
        deposit_value: U256::ZERO,
        gas_fee_cap: U256::ZERO,
        gas: 0,
        retry_to: None,
        retry_value: U256::ZERO,
        beneficiary: address!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
        max_submission_fee: U256::ZERO,
        fee_refund_addr: address!("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
        retry_data: Bytes::new(),
    });
    let f = arb_tx_fields(&tx);
    assert!(!f.contains_key("retryTo"));
    assert!(f.contains_key("requestId"));
}

#[test]
fn internal_tx_has_no_extra_arbitrum_fields() {
    let tx = ArbTypedTransaction::Internal(ArbInternalTx {
        chain_id: U256::from(42161u64),
        data: Bytes::new(),
    });
    assert!(arb_tx_fields(&tx).is_empty());
}

#[test]
fn unsigned_tx_has_no_extra_arbitrum_fields() {
    let tx = ArbTypedTransaction::Unsigned(ArbUnsignedTx {
        chain_id: U256::from(42161u64),
        from: address!("1111111111111111111111111111111111111111"),
        nonce: 0,
        gas_fee_cap: U256::ZERO,
        gas: 0,
        to: None,
        value: U256::ZERO,
        data: Bytes::new(),
    });
    assert!(arb_tx_fields(&tx).is_empty());
}

#[test]
fn legacy_tx_has_no_extra_arbitrum_fields() {
    let t = TxLegacy {
        chain_id: Some(1),
        nonce: 0,
        gas_price: 1_000_000_000,
        gas_limit: 21_000,
        to: TxKind::Call(address!("2222222222222222222222222222222222222222")),
        value: U256::ZERO,
        input: Bytes::new(),
    };
    assert!(arb_tx_fields(&ArbTypedTransaction::Legacy(t)).is_empty());
}

#[test]
fn eip1559_tx_has_no_extra_arbitrum_fields() {
    let t = TxEip1559 {
        chain_id: 1,
        nonce: 0,
        gas_limit: 21_000,
        max_fee_per_gas: 1_000_000_000,
        max_priority_fee_per_gas: 0,
        to: TxKind::Call(address!("2222222222222222222222222222222222222222")),
        value: U256::ZERO,
        access_list: Default::default(),
        input: Bytes::new(),
    };
    assert!(arb_tx_fields(&ArbTypedTransaction::Eip1559(t)).is_empty());
}

#[test]
fn eip2930_tx_has_no_extra_arbitrum_fields() {
    let t = TxEip2930 {
        chain_id: 1,
        nonce: 0,
        gas_price: 1_000_000_000,
        gas_limit: 21_000,
        to: TxKind::Call(address!("2222222222222222222222222222222222222222")),
        value: U256::ZERO,
        access_list: Default::default(),
        input: Bytes::new(),
    };
    assert!(arb_tx_fields(&ArbTypedTransaction::Eip2930(t)).is_empty());
}
