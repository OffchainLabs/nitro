use arb_stylus::pricing::{keccak_price, pow_price, read_price, write_price};
use proptest::prelude::*;

#[test]
fn read_price_floor_for_small_payloads() {
    assert_eq!(read_price(0).0, 16381);
    assert_eq!(read_price(32).0, 16381);
}

#[test]
fn read_price_per_byte_above_32() {
    assert_eq!(read_price(33).0, 16381 + 55);
    assert_eq!(read_price(64).0, 16381 + 55 * 32);
}

#[test]
fn write_price_floor_for_small_payloads() {
    assert_eq!(write_price(0).0, 5040);
    assert_eq!(write_price(32).0, 5040);
}

#[test]
fn write_price_per_byte_above_32() {
    assert_eq!(write_price(33).0, 5040 + 30);
    assert_eq!(write_price(64).0, 5040 + 30 * 32);
}

#[test]
fn keccak_price_floor_for_zero_to_two_words() {
    // div_ceil(0/32) saturating-subs to 0, so floor cost is 121800.
    assert_eq!(keccak_price(0).0, 121800);
    assert_eq!(keccak_price(32).0, 121800);
    assert_eq!(keccak_price(64).0, 121800);
}

#[test]
fn keccak_price_grows_per_word_above_two_words() {
    assert_eq!(keccak_price(96).0, 121800 + 21000);
    assert_eq!(keccak_price(33).0, 121800);
}

#[test]
#[allow(clippy::identity_op)]
fn pow_price_for_zero_exponent() {
    let exp = [0u8; 32];
    assert_eq!(pow_price(&exp).0, 3000 + 1 * 17500);
}

#[test]
fn pow_price_for_one_byte_exponent() {
    let mut exp = [0u8; 32];
    exp[31] = 1;
    // 31 leading zero bytes -> exp counter = 33 - 31 = 2.
    assert_eq!(pow_price(&exp).0, 3000 + 2 * 17500);
}

#[test]
fn pow_price_for_full_byte_exponent() {
    let exp = [0xFFu8; 32];
    assert_eq!(pow_price(&exp).0, 3000 + 33 * 17500);
}

proptest! {
    #[test]
    fn read_price_monotonic_in_bytes(a in 0u32..1024, b in 0u32..1024) {
        let (lo, hi) = if a < b { (a, b) } else { (b, a) };
        prop_assert!(read_price(lo).0 <= read_price(hi).0);
    }

    #[test]
    fn write_price_monotonic_in_bytes(a in 0u32..1024, b in 0u32..1024) {
        let (lo, hi) = if a < b { (a, b) } else { (b, a) };
        prop_assert!(write_price(lo).0 <= write_price(hi).0);
    }

    #[test]
    fn keccak_price_monotonic_above_two_words(a in 65u32..2048, b in 65u32..2048) {
        let (lo, hi) = if a < b { (a, b) } else { (b, a) };
        prop_assert!(keccak_price(lo).0 <= keccak_price(hi).0);
    }
}
