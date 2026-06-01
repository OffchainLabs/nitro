use arbos::programs::{
    memory::MemoryModel,
    types::{evm_memory_cost, to_word_size, UserOutcome},
};

#[test]
fn user_outcome_from_u8_all_defined_values() {
    assert_eq!(UserOutcome::from_u8(0), Some(UserOutcome::Success));
    assert_eq!(UserOutcome::from_u8(1), Some(UserOutcome::Revert));
    assert_eq!(UserOutcome::from_u8(2), Some(UserOutcome::Failure));
    assert_eq!(UserOutcome::from_u8(3), Some(UserOutcome::OutOfInk));
    assert_eq!(UserOutcome::from_u8(4), Some(UserOutcome::OutOfStack));
}

#[test]
fn user_outcome_from_u8_rejects_undefined() {
    assert_eq!(UserOutcome::from_u8(5), None);
    assert_eq!(UserOutcome::from_u8(255), None);
}

#[test]
fn to_word_size_rounds_up() {
    assert_eq!(to_word_size(0), 0);
    assert_eq!(to_word_size(1), 1);
    assert_eq!(to_word_size(31), 1);
    assert_eq!(to_word_size(32), 1);
    assert_eq!(to_word_size(33), 2);
    assert_eq!(to_word_size(64), 2);
    assert_eq!(to_word_size(65), 3);
}

#[test]
fn to_word_size_saturates_near_u64_max() {
    assert!(to_word_size(u64::MAX) > 0);
    assert!(to_word_size(u64::MAX - 10) > 0);
}

#[test]
fn evm_memory_cost_zero_for_empty_access() {
    assert_eq!(evm_memory_cost(0), 0);
}

#[test]
fn evm_memory_cost_one_word_is_3_plus_0() {
    assert_eq!(evm_memory_cost(32), 3);
    assert_eq!(evm_memory_cost(1), 3);
}

#[test]
fn evm_memory_cost_two_words_is_6_plus_0() {
    assert_eq!(evm_memory_cost(64), 6);
}

#[test]
fn evm_memory_cost_grows_quadratic_for_large_sizes() {
    let small = evm_memory_cost(32 * 100);
    let big = evm_memory_cost(32 * 10_000);
    assert!(big > small * 100);
}

#[test]
fn memory_model_free_pages_cost_zero() {
    let m = MemoryModel::new(10, 100);
    assert_eq!(m.gas_cost(5, 0, 0), 0);
    assert_eq!(m.gas_cost(10, 0, 0), 0);
}

#[test]
fn memory_model_above_free_pages_charges_linear_and_exp() {
    let m = MemoryModel::new(2, 100);
    let cost_above = m.gas_cost(5, 0, 0);
    assert!(cost_above > 0);
}

#[test]
fn memory_model_growth_is_monotonic_in_new_pages() {
    let m = MemoryModel::new(0, 10);
    let c1 = m.gas_cost(1, 0, 0);
    let c5 = m.gas_cost(5, 0, 0);
    let c20 = m.gas_cost(20, 0, 0);
    assert!(c5 > c1);
    assert!(c20 > c5);
}

#[test]
fn memory_model_does_not_recharge_for_ever_used_pages() {
    let m = MemoryModel::new(0, 10);
    let cost_after_ever = m.gas_cost(0, 0, 50);
    assert_eq!(cost_after_ever, 0);
}

#[test]
fn memory_model_linear_cost_matches_page_gas_times_adding() {
    let m = MemoryModel::new(0, 10);
    let c = m.gas_cost(5, 0, 10);
    assert_eq!(c, 5 * 10);
}

#[test]
fn memory_model_saturates_past_exp_table() {
    let m = MemoryModel::new(0, 100);
    let c = m.gas_cost(200, 0, 0);
    assert!(c > 0);
}
