use alloy_primitives::{Address, B256, U256};
use arbos::parse_l2::parse_l2_transactions;
use proptest::prelude::*;

const KINDS: &[u8] = &[0, 1, 3, 6, 7, 9, 11, 12, 100, 255];

proptest! {
    #[test]
    fn parse_l2_transactions_no_panic(
        kind in prop::sample::select(KINDS),
        poster in prop::array::uniform20(any::<u8>()),
        body in prop::collection::vec(any::<u8>(), 0..512),
        request_id in prop::option::of(any::<[u8; 32]>()),
        l1_base_fee in prop::option::of(any::<[u8; 32]>()),
        chain_id in any::<u64>(),
    ) {
        let _ = parse_l2_transactions(
            kind,
            Address::from(poster),
            &body,
            request_id.map(B256::from),
            l1_base_fee.map(U256::from_be_bytes),
            chain_id,
        );
    }

    #[test]
    fn parse_l2_transactions_random_kind_no_panic(
        kind in any::<u8>(),
        body in prop::collection::vec(any::<u8>(), 0..256),
    ) {
        let _ = parse_l2_transactions(
            kind,
            Address::ZERO,
            &body,
            None,
            None,
            42_161,
        );
    }
}

#[test]
fn empty_message_returns_no_txs() {
    let res = parse_l2_transactions(3, Address::ZERO, &[], None, None, 42_161).unwrap();
    assert!(res.is_empty());
}

#[test]
fn end_of_block_returns_no_txs() {
    let res = parse_l2_transactions(6, Address::ZERO, &[], None, None, 42_161).unwrap();
    assert!(res.is_empty());
}
