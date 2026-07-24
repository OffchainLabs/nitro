use alloy_eips::eip4895::Withdrawal;
use alloy_primitives::{address, Bytes, B256};
use alloy_rpc_types_engine::PayloadAttributes as AlloyPayloadAttributes;
use arb_payload::{arb_payload_id, ArbPayloadAttributes};
use reth_payload_primitives::PayloadAttributes;

fn base_attrs() -> ArbPayloadAttributes {
    ArbPayloadAttributes {
        inner: AlloyPayloadAttributes {
            timestamp: 1_700_000_000,
            prev_randao: B256::repeat_byte(0x01),
            suggested_fee_recipient: address!("1111111111111111111111111111111111111111"),
            withdrawals: None,
            parent_beacon_block_root: None,
        },
        transactions: None,
        no_tx_pool: false,
    }
}

#[test]
fn payload_id_is_deterministic() {
    let parent = B256::repeat_byte(0xAB);
    let attrs = base_attrs();
    let id1 = arb_payload_id(&parent, &attrs);
    let id2 = arb_payload_id(&parent, &attrs);
    assert_eq!(id1, id2);
}

#[test]
fn payload_id_differs_with_parent_hash() {
    let a = base_attrs();
    let id1 = arb_payload_id(&B256::repeat_byte(0x11), &a);
    let id2 = arb_payload_id(&B256::repeat_byte(0x22), &a);
    assert_ne!(id1, id2);
}

#[test]
fn payload_id_differs_with_timestamp() {
    let parent = B256::repeat_byte(0x01);
    let mut a = base_attrs();
    let id1 = arb_payload_id(&parent, &a);
    a.inner.timestamp = 1_700_000_001;
    let id2 = arb_payload_id(&parent, &a);
    assert_ne!(id1, id2);
}

#[test]
fn payload_id_differs_with_prev_randao() {
    let parent = B256::repeat_byte(0x01);
    let mut a = base_attrs();
    let id1 = arb_payload_id(&parent, &a);
    a.inner.prev_randao = B256::repeat_byte(0x99);
    let id2 = arb_payload_id(&parent, &a);
    assert_ne!(id1, id2);
}

#[test]
fn payload_id_differs_with_fee_recipient() {
    let parent = B256::repeat_byte(0x01);
    let mut a = base_attrs();
    let id1 = arb_payload_id(&parent, &a);
    a.inner.suggested_fee_recipient = address!("2222222222222222222222222222222222222222");
    let id2 = arb_payload_id(&parent, &a);
    assert_ne!(id1, id2);
}

#[test]
fn payload_id_differs_with_no_tx_pool_flag() {
    let parent = B256::repeat_byte(0x01);
    let mut a = base_attrs();
    let id1 = arb_payload_id(&parent, &a);
    a.no_tx_pool = true;
    let id2 = arb_payload_id(&parent, &a);
    assert_ne!(id1, id2);
}

#[test]
fn payload_id_differs_with_forced_transactions() {
    let parent = B256::repeat_byte(0x01);
    let mut a = base_attrs();
    let id1 = arb_payload_id(&parent, &a);
    a.transactions = Some(vec![Bytes::from(vec![0xDE, 0xAD])]);
    let id2 = arb_payload_id(&parent, &a);
    assert_ne!(id1, id2);
}

#[test]
fn payload_id_differs_with_parent_beacon_root() {
    let parent = B256::repeat_byte(0x01);
    let mut a = base_attrs();
    let id1 = arb_payload_id(&parent, &a);
    a.inner.parent_beacon_block_root = Some(B256::repeat_byte(0xFF));
    let id2 = arb_payload_id(&parent, &a);
    assert_ne!(id1, id2);
}

#[test]
fn payload_id_differs_with_withdrawals() {
    let parent = B256::repeat_byte(0x01);
    let mut a = base_attrs();
    let id1 = arb_payload_id(&parent, &a);
    a.inner.withdrawals = Some(vec![Withdrawal {
        index: 1,
        validator_index: 2,
        address: address!("1111111111111111111111111111111111111111"),
        amount: 100,
    }]);
    let id2 = arb_payload_id(&parent, &a);
    assert_ne!(id1, id2);
}

// ==== ArbPayloadAttributes trait ====

#[test]
fn payload_attributes_trait_reads_timestamp() {
    let a = base_attrs();
    assert_eq!(a.timestamp(), 1_700_000_000);
}

#[test]
fn payload_attributes_trait_reads_withdrawals_none_by_default() {
    let a = base_attrs();
    assert!(a.withdrawals().is_none());
}

#[test]
fn payload_attributes_trait_reads_parent_beacon_root_none_by_default() {
    let a = base_attrs();
    assert!(a.parent_beacon_block_root().is_none());
}

#[test]
fn payload_attributes_serialize_roundtrip() {
    let a = base_attrs();
    let json = serde_json::to_string(&a).expect("serialize");
    let b: ArbPayloadAttributes = serde_json::from_str(&json).expect("deserialize");
    assert_eq!(a, b);
}

#[test]
fn payload_attributes_default_no_tx_pool_is_false() {
    let json = r#"{
        "timestamp": "0x0",
        "prevRandao": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "suggestedFeeRecipient": "0x0000000000000000000000000000000000000000"
    }"#;
    let a: ArbPayloadAttributes = serde_json::from_str(json).expect("deserialize");
    assert!(!a.no_tx_pool);
    assert!(a.transactions.is_none());
}
