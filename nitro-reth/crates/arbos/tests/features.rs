use alloy_primitives::B256;
use arb_storage::Storage;
use arb_test_utils::ArbosHarness;
use arbos::features::open_features;

#[test]
fn calldata_price_increase_defaults_to_false() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let sub = Storage::new(state_ptr, B256::ZERO).open_sub_storage(&[9]);
    let f = open_features(state_ptr, sub.base_key(), 0);
    assert!(!f.is_increased_calldata_price_enabled().unwrap());
}

#[test]
fn enable_then_read_round_trips() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let sub = Storage::new(state_ptr, B256::ZERO).open_sub_storage(&[9]);
    let f = open_features(state_ptr, sub.base_key(), 0);
    f.set_calldata_price_increase(true).unwrap();
    assert!(f.is_increased_calldata_price_enabled().unwrap());
    f.set_calldata_price_increase(false).unwrap();
    assert!(!f.is_increased_calldata_price_enabled().unwrap());
}
