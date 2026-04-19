use alloy_primitives::{address, Address};
use arbos::util::{
    does_tx_type_alias, inverse_remap_l1_address, remap_l1_address, tx_type_has_poster_costs,
    ADDRESS_ALIAS_OFFSET, INVERSE_ADDRESS_ALIAS_OFFSET,
};
use proptest::prelude::*;

#[test]
fn remap_zero_returns_offset_constant() {
    assert_eq!(remap_l1_address(Address::ZERO), ADDRESS_ALIAS_OFFSET);
}

#[test]
fn inverse_offset_is_negation_mod_160() {
    assert_eq!(
        remap_l1_address(INVERSE_ADDRESS_ALIAS_OFFSET),
        Address::ZERO
    );
    assert_eq!(
        inverse_remap_l1_address(ADDRESS_ALIAS_OFFSET),
        Address::ZERO
    );
}

#[test]
fn remap_and_inverse_round_trip() {
    let a = address!("DEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF");
    assert_eq!(inverse_remap_l1_address(remap_l1_address(a)), a);
    assert_eq!(remap_l1_address(inverse_remap_l1_address(a)), a);
}

#[test]
fn remap_is_wrapping_addition() {
    let max = Address::from([0xFFu8; 20]);
    let one = remap_l1_address(max);
    let expected = address!("1111000000000000000000000000000000001110");
    assert_eq!(one, expected);
}

#[test]
fn does_tx_type_alias_covers_expected_types() {
    assert!(does_tx_type_alias(0x65));
    assert!(does_tx_type_alias(0x66));
    assert!(does_tx_type_alias(0x68));
    assert!(!does_tx_type_alias(0x64));
    assert!(!does_tx_type_alias(0x67));
    assert!(!does_tx_type_alias(0x69));
    assert!(!does_tx_type_alias(0x6A));
    assert!(!does_tx_type_alias(0x00));
    assert!(!does_tx_type_alias(0x02));
}

#[test]
fn tx_type_has_poster_costs_excludes_deposit_retry_internal_submit() {
    // Matches Nitro util.TxTypeHasPosterCosts: only standard EOA-signed types
    // pay poster costs; all Arbitrum-specific types do not.
    for t in [0x64u8, 0x65, 0x66, 0x68, 0x69, 0x6A] {
        assert!(
            !tx_type_has_poster_costs(t),
            "type {t:#x} should not have poster costs"
        );
    }
    for t in [0x00u8, 0x01, 0x02, 0x04] {
        assert!(
            tx_type_has_poster_costs(t),
            "type {t:#x} should have poster costs"
        );
    }
}

proptest! {
    #[test]
    fn remap_inverse_round_trip_prop(bytes in prop::array::uniform20(any::<u8>())) {
        let a = Address::from(bytes);
        prop_assert_eq!(inverse_remap_l1_address(remap_l1_address(a)), a);
        prop_assert_eq!(remap_l1_address(inverse_remap_l1_address(a)), a);
    }
}
