use std::collections::HashMap;

use alloy_primitives::{Address, B256};
use arb_evm::context::{ActivatedWasm, ArbitrumExtraData, RecentWasms};

#[test]
fn recent_wasms_insert_new_returns_false() {
    let mut r = RecentWasms::new(4);
    assert!(!r.insert(B256::repeat_byte(1)));
}

#[test]
fn recent_wasms_insert_duplicate_returns_true_and_promotes() {
    let mut r = RecentWasms::new(3);
    r.insert(B256::repeat_byte(1));
    r.insert(B256::repeat_byte(2));
    assert!(r.insert(B256::repeat_byte(1)));
    r.insert(B256::repeat_byte(3));
    r.insert(B256::repeat_byte(4));
    assert!(!r.contains(&B256::repeat_byte(2)));
    assert!(r.contains(&B256::repeat_byte(1)));
}

#[test]
fn recent_wasms_evicts_oldest_when_over_capacity() {
    let mut r = RecentWasms::new(2);
    r.insert(B256::repeat_byte(1));
    r.insert(B256::repeat_byte(2));
    r.insert(B256::repeat_byte(3));
    assert!(!r.contains(&B256::repeat_byte(1)));
    assert!(r.contains(&B256::repeat_byte(2)));
    assert!(r.contains(&B256::repeat_byte(3)));
}

#[test]
fn recent_wasms_contains_returns_false_for_unknown() {
    let r = RecentWasms::new(4);
    assert!(!r.contains(&B256::repeat_byte(9)));
}

#[test]
fn activate_wasm_fresh_module_ok() {
    let mut x = ArbitrumExtraData::default();
    let mut asm = HashMap::new();
    asm.insert("x86_64".to_string(), vec![0xAA]);
    let r = x.activate_wasm(B256::repeat_byte(1), asm.clone(), vec![0xBB]);
    assert!(r.is_ok());
    assert_eq!(x.activated_wasms.len(), 1);
}

#[test]
fn activate_wasm_same_targets_replaces_ok() {
    let mut x = ArbitrumExtraData::default();
    let mut asm1 = HashMap::new();
    asm1.insert("x86_64".to_string(), vec![1]);
    asm1.insert("arm64".to_string(), vec![2]);
    x.activate_wasm(B256::repeat_byte(1), asm1.clone(), vec![0xBB]).unwrap();
    let mut asm2 = HashMap::new();
    asm2.insert("x86_64".to_string(), vec![3]);
    asm2.insert("arm64".to_string(), vec![4]);
    assert!(x.activate_wasm(B256::repeat_byte(1), asm2, vec![0xCC]).is_ok());
}

#[test]
fn activate_wasm_inconsistent_targets_errors() {
    let mut x = ArbitrumExtraData::default();
    let mut asm1 = HashMap::new();
    asm1.insert("x86_64".to_string(), vec![1]);
    asm1.insert("arm64".to_string(), vec![2]);
    x.activate_wasm(B256::repeat_byte(1), asm1, vec![]).unwrap();
    let mut asm2 = HashMap::new();
    asm2.insert("x86_64".to_string(), vec![3]);
    let err = x.activate_wasm(B256::repeat_byte(1), asm2, vec![]);
    assert!(err.is_err());
}

#[test]
fn balance_burn_mint_adjust_delta() {
    let mut x = ArbitrumExtraData::default();
    x.expect_balance_burn(100);
    assert_eq!(x.unexpected_balance_delta(), 100);
    x.expect_balance_mint(30);
    assert_eq!(x.unexpected_balance_delta(), 70);
}

#[test]
fn stylus_pages_add_updates_open_and_ever() {
    let mut x = ArbitrumExtraData::default();
    let prev = x.add_stylus_pages(5);
    assert_eq!(prev, (0, 0));
    assert_eq!(x.get_stylus_pages(), (5, 5));
    x.add_stylus_pages(3);
    assert_eq!(x.get_stylus_pages(), (8, 8));
}

#[test]
fn stylus_pages_ever_is_max_never_decreases() {
    let mut x = ArbitrumExtraData::default();
    x.add_stylus_pages(10);
    x.set_stylus_pages_open(3);
    x.add_stylus_pages(2);
    let (open, ever) = x.get_stylus_pages();
    assert_eq!(open, 5);
    assert_eq!(ever, 10);
}

#[test]
fn stylus_pages_saturate_at_u16_max() {
    let mut x = ArbitrumExtraData::default();
    x.set_stylus_pages_open(u16::MAX - 1);
    x.add_stylus_pages(100);
    assert_eq!(x.get_stylus_pages_open(), u16::MAX);
}

#[test]
fn stylus_reset_zeros_both_counters() {
    let mut x = ArbitrumExtraData::default();
    x.add_stylus_pages(42);
    x.reset_stylus_pages();
    assert_eq!(x.get_stylus_pages(), (0, 0));
}

#[test]
fn add_stylus_pages_ever_is_direct_watermark_increase() {
    let mut x = ArbitrumExtraData::default();
    x.add_stylus_pages_ever(50);
    let (open, ever) = x.get_stylus_pages();
    assert_eq!(open, 0);
    assert_eq!(ever, 50);
}

#[test]
fn tx_filter_default_false_then_toggle() {
    let mut x = ArbitrumExtraData::default();
    assert!(!x.is_tx_filtered());
    x.filter_tx();
    assert!(x.is_tx_filtered());
    x.clear_tx_filter();
    assert!(!x.is_tx_filtered());
}

#[test]
fn zombie_default_empty_add_and_check() {
    let mut x = ArbitrumExtraData::default();
    let addr = Address::repeat_byte(0xAA);
    assert!(!x.is_zombie(&addr));
    x.create_zombie(addr);
    assert!(x.is_zombie(&addr));
    assert!(!x.is_zombie(&Address::repeat_byte(0xBB)));
}

#[test]
fn start_recording_clears_user_wasms() {
    let mut x = ArbitrumExtraData::default();
    x.record_program(
        B256::repeat_byte(1),
        ActivatedWasm { asm: HashMap::new(), module: vec![] },
    );
    assert_eq!(x.user_wasms.len(), 1);
    x.start_recording();
    assert_eq!(x.user_wasms.len(), 0);
}
