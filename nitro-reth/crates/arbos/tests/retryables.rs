use alloy_primitives::{address, b256, keccak256, Address, B256, U256};
use arb_test_utils::ArbosHarness;
use arbos::retryables::{retryable_escrow_address, retryable_submission_fee};

const FROM: Address = address!("00000000000000000000000000000000000A11CE");
const BENEFICIARY: Address = address!("00000000000000000000000000000000000B0B00");
const DEST: Address = address!("00000000000000000000000000000000C4A841E0");
const TICKET_ID: B256 = b256!("0000000000000000000000000000000000000000000000000000000000000042");

fn submit(h: &mut ArbosHarness, id: B256, timeout: u64, calldata: &[u8]) {
    let rs = h.retryable_state();
    rs.create_retryable(
        id,
        timeout,
        FROM,
        Some(DEST),
        U256::from(1_000_000u64),
        BENEFICIARY,
        calldata,
    )
    .unwrap();
}

#[test]
fn create_open_round_trip() {
    let mut h = ArbosHarness::new().initialize();
    submit(&mut h, TICKET_ID, 1_000, b"hello world");

    let rs = h.retryable_state();
    let opened = rs.open_retryable(TICKET_ID, 999).unwrap().expect("exists");
    assert_eq!(opened.from().unwrap(), FROM);
    assert_eq!(opened.to().unwrap(), Some(DEST));
    assert_eq!(opened.callvalue().unwrap(), U256::from(1_000_000u64));
    assert_eq!(opened.beneficiary().unwrap(), BENEFICIARY);
    assert_eq!(opened.calldata().unwrap(), b"hello world".to_vec());
    assert_eq!(opened.num_tries().unwrap(), 0);
}

#[test]
fn open_returns_none_after_timeout() {
    let mut h = ArbosHarness::new().initialize();
    submit(&mut h, TICKET_ID, 100, &[]);

    let rs = h.retryable_state();
    assert!(rs.open_retryable(TICKET_ID, 50).unwrap().is_some());
    assert!(rs.open_retryable(TICKET_ID, 100).unwrap().is_some());
    assert!(rs.open_retryable(TICKET_ID, 101).unwrap().is_none());
    assert!(rs.open_retryable(TICKET_ID, 200).unwrap().is_none());
}

#[test]
fn delete_returns_false_for_unknown_id() {
    let mut h = ArbosHarness::new().initialize();
    let rs = h.retryable_state();

    let mut transfers = Vec::new();
    let did = rs
        .delete_retryable(
            b256!("0000000000000000000000000000000000000000000000000000000000000099"),
            |from, to, amount| {
                transfers.push((from, to, amount));
                Ok(())
            },
            |_| U256::ZERO,
        )
        .unwrap();

    assert!(!did);
    assert!(transfers.is_empty());
}

#[test]
fn delete_clears_storage_and_transfers_escrow() {
    let mut h = ArbosHarness::new().initialize();
    submit(&mut h, TICKET_ID, 1_000, b"data");
    let escrow = retryable_escrow_address(TICKET_ID);
    let escrow_balance = U256::from(7_777_777u64);

    let mut transfers = Vec::new();
    let did = {
        let rs = h.retryable_state();
        rs.delete_retryable(
            TICKET_ID,
            |from, to, amount| {
                transfers.push((from, to, amount));
                Ok(())
            },
            |addr| if addr == escrow { escrow_balance } else { U256::ZERO },
        )
        .unwrap()
    };

    assert!(did);
    assert_eq!(transfers, vec![(escrow, BENEFICIARY, escrow_balance)]);

    let rs = h.retryable_state();
    assert!(rs.open_retryable(TICKET_ID, 500).unwrap().is_none());
}

#[test]
fn increment_num_tries_sequence() {
    let mut h = ArbosHarness::new().initialize();
    submit(&mut h, TICKET_ID, 1_000, &[]);
    let rs = h.retryable_state();
    let r = rs.open_retryable(TICKET_ID, 500).unwrap().unwrap();
    assert_eq!(r.increment_num_tries().unwrap(), 1);
    assert_eq!(r.increment_num_tries().unwrap(), 2);
    assert_eq!(r.increment_num_tries().unwrap(), 3);
    assert_eq!(r.num_tries().unwrap(), 3);
}

#[test]
fn escrow_address_is_deterministic() {
    let id = b256!("00000000000000000000000000000000000000000000000000000000DEADBEEF");
    let mut data = Vec::from(b"retryable escrow".as_ref());
    data.extend_from_slice(id.as_slice());
    let expected = Address::from_slice(&keccak256(&data)[12..]);
    assert_eq!(retryable_escrow_address(id), expected);
}

#[test]
fn submission_fee_scales_with_calldata_length() {
    let l1_base_fee = U256::from(1_000_000_000u64);
    let small = retryable_submission_fee(0, l1_base_fee);
    let medium = retryable_submission_fee(100, l1_base_fee);
    let large = retryable_submission_fee(10_000, l1_base_fee);
    assert!(small < medium);
    assert!(medium < large);
}

#[test]
fn submission_fee_scales_with_l1_base_fee() {
    let calldata_len = 100;
    let cheap = retryable_submission_fee(calldata_len, U256::from(1u64));
    let expensive = retryable_submission_fee(calldata_len, U256::from(1_000_000_000u64));
    assert!(cheap < expensive);
}
