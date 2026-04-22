use alloy_primitives::{address, U256};
use arbos::internal_tx::{
    decode_start_block_data, encode_batch_posting_report, encode_batch_posting_report_v2,
    encode_start_block, INTERNAL_TX_BATCH_POSTING_REPORT_METHOD_ID,
    INTERNAL_TX_BATCH_POSTING_REPORT_V2_METHOD_ID, INTERNAL_TX_START_BLOCK_METHOD_ID,
};
use proptest::prelude::*;

#[test]
fn encode_start_block_starts_with_selector() {
    let data = encode_start_block(U256::from(1u64), 100, 50, 12);
    assert_eq!(&data[..4], &INTERNAL_TX_START_BLOCK_METHOD_ID);
    assert_eq!(data.len(), 4 + 32 * 4);
}

#[test]
fn encode_batch_posting_report_starts_with_selector() {
    let data = encode_batch_posting_report(
        100,
        address!("AAAA000000000000000000000000000000000000"),
        7,
        21000,
        U256::from(1_000_000_000u64),
    );
    assert_eq!(&data[..4], &INTERNAL_TX_BATCH_POSTING_REPORT_METHOD_ID);
    assert_eq!(data.len(), 4 + 32 * 5);
}

#[test]
fn encode_batch_posting_report_v2_starts_with_selector() {
    let data = encode_batch_posting_report_v2(
        100,
        address!("BBBB000000000000000000000000000000000000"),
        7,
        500,
        300,
        2000,
        U256::from(1_000_000_000u64),
    );
    assert_eq!(&data[..4], &INTERNAL_TX_BATCH_POSTING_REPORT_V2_METHOD_ID);
    assert_eq!(data.len(), 4 + 32 * 7);
}

#[test]
fn encode_decode_start_block_round_trip() {
    let data = encode_start_block(U256::from(123_456_789u64), 1000, 500, 12);
    let decoded = decode_start_block_data(&data).unwrap();
    assert_eq!(decoded.l1_base_fee, U256::from(123_456_789u64));
    assert_eq!(decoded.l1_block_number, 1000);
    assert_eq!(decoded.l2_block_number, 500);
    assert_eq!(decoded.time_passed, 12);
}

#[test]
fn decode_start_block_rejects_short_input() {
    assert!(decode_start_block_data(&[]).is_err());
    assert!(decode_start_block_data(&[0u8; 4]).is_err());
    assert!(decode_start_block_data(&[0u8; 100]).is_err());
}

proptest! {
    #[test]
    fn start_block_encode_decode_round_trip_prop(
        l1_base_fee in any::<[u8; 32]>(),
        l1_block_number in any::<u64>(),
        l2_block_number in any::<u64>(),
        time_passed in any::<u64>(),
    ) {
        let l1_base_fee = U256::from_be_bytes(l1_base_fee);
        let data = encode_start_block(l1_base_fee, l1_block_number, l2_block_number, time_passed);
        let decoded = decode_start_block_data(&data).unwrap();
        prop_assert_eq!(decoded.l1_base_fee, l1_base_fee);
        prop_assert_eq!(decoded.l1_block_number, l1_block_number);
        prop_assert_eq!(decoded.l2_block_number, l2_block_number);
        prop_assert_eq!(decoded.time_passed, time_passed);
    }

    #[test]
    fn decode_start_block_no_panic_on_arbitrary(bytes in prop::collection::vec(any::<u8>(), 0..512)) {
        let _ = decode_start_block_data(&bytes);
    }
}
