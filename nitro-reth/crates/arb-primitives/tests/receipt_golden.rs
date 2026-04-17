use alloy_consensus::{Eip658Value, Receipt as AlloyReceipt, Typed2718};
use alloy_eips::eip2718::Encodable2718;
use arb_primitives::receipt::{ArbDepositReceipt, ArbReceipt, ArbReceiptKind};

fn empty_receipt() -> AlloyReceipt {
    AlloyReceipt {
        status: Eip658Value::Eip658(true),
        cumulative_gas_used: 0,
        logs: vec![],
    }
}

fn encode(r: &ArbReceipt) -> Vec<u8> {
    let mut out = Vec::new();
    r.encode_2718(&mut out);
    out
}

#[test]
fn deposit_receipt_type_byte_is_0x64() {
    let r = ArbReceipt::new(ArbReceiptKind::Deposit(ArbDepositReceipt));
    assert_eq!(r.ty(), 0x64);
    let bytes = encode(&r);
    assert_eq!(bytes[0], 0x64);
}

#[test]
fn unsigned_receipt_type_byte_is_0x65() {
    let r = ArbReceipt::new(ArbReceiptKind::Unsigned(empty_receipt()));
    assert_eq!(r.ty(), 0x65);
    assert_eq!(encode(&r)[0], 0x65);
}

#[test]
fn contract_receipt_type_byte_is_0x66() {
    let r = ArbReceipt::new(ArbReceiptKind::Contract(empty_receipt()));
    assert_eq!(r.ty(), 0x66);
    assert_eq!(encode(&r)[0], 0x66);
}

#[test]
fn retry_receipt_type_byte_is_0x68() {
    let r = ArbReceipt::new(ArbReceiptKind::Retry(empty_receipt()));
    assert_eq!(r.ty(), 0x68);
    assert_eq!(encode(&r)[0], 0x68);
}

#[test]
fn submit_retryable_receipt_type_byte_is_0x69() {
    let r = ArbReceipt::new(ArbReceiptKind::SubmitRetryable(empty_receipt()));
    assert_eq!(r.ty(), 0x69);
    assert_eq!(encode(&r)[0], 0x69);
}

#[test]
fn internal_receipt_type_byte_is_0x6a() {
    let r = ArbReceipt::new(ArbReceiptKind::Internal(empty_receipt()));
    assert_eq!(r.ty(), 0x6A);
    assert_eq!(encode(&r)[0], 0x6A);
}

#[test]
fn legacy_receipt_no_type_prefix() {
    let r = ArbReceipt::new(ArbReceiptKind::Legacy(empty_receipt()));
    assert_eq!(r.ty(), 0x00);
    let bytes = encode(&r);
    // RLP list header byte starts with 0xC0+, not a type prefix.
    assert!(
        bytes[0] >= 0xC0,
        "first byte should be RLP list header, got {:#x}",
        bytes[0]
    );
}

#[test]
fn eip1559_receipt_type_byte_is_0x02() {
    let r = ArbReceipt::new(ArbReceiptKind::Eip1559(empty_receipt()));
    assert_eq!(r.ty(), 0x02);
    assert_eq!(encode(&r)[0], 0x02);
}

#[test]
fn empty_unsigned_receipt_golden_bytes() {
    let r = ArbReceipt::new(ArbReceiptKind::Unsigned(empty_receipt()));
    let bytes = encode(&r);
    // Type prefix 0x65, then RLP list of (status=true, cumulative_gas=0, logs=[]).
    // status true → 0x01, cumulative 0 → 0x80, empty logs → 0xc0.
    // Inner list payload: 0x01 0x80 0xc0 = 3 bytes. Header: 0xc3.
    assert_eq!(bytes, vec![0x65, 0xc3, 0x01, 0x80, 0xc0]);
}

#[test]
fn empty_internal_receipt_golden_bytes() {
    let r = ArbReceipt::new(ArbReceiptKind::Internal(empty_receipt()));
    let bytes = encode(&r);
    assert_eq!(bytes, vec![0x6A, 0xc3, 0x01, 0x80, 0xc0]);
}

#[test]
fn empty_deposit_receipt_golden_bytes() {
    let r = ArbReceipt::new(ArbReceiptKind::Deposit(ArbDepositReceipt));
    let bytes = encode(&r);
    assert_eq!(bytes, vec![0x64, 0xc3, 0x01, 0x80, 0xc0]);
}
