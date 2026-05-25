use alloy_primitives::B256;
use arb_storage::Storage;
use arb_test_utils::ArbosHarness;
use arbos::features::open_features;

#[test]
fn calldata_price_increase_defaults_to_false() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let sub = Storage::new(unsafe { &mut *state_ptr }, B256::ZERO).open_sub_storage(&[9]);
    let f = open_features::<()>(sub.base_key(), 0);
    let backend = unsafe { &mut *state_ptr };
    assert!(!f.is_increased_calldata_price_enabled(backend).unwrap());
}

#[test]
fn enable_then_read_round_trips() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let sub = Storage::new(unsafe { &mut *state_ptr }, B256::ZERO).open_sub_storage(&[9]);
    let f = open_features::<()>(sub.base_key(), 0);
    let backend = unsafe { &mut *state_ptr };
    f.set_calldata_price_increase(backend, true).unwrap();
    assert!(f.is_increased_calldata_price_enabled(backend).unwrap());
    f.set_calldata_price_increase(backend, false).unwrap();
    assert!(!f.is_increased_calldata_price_enabled(backend).unwrap());
}
