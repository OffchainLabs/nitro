use alloy_primitives::{Address, B256, U256};
use arbos::programs::api::{ApiStatus, RequestParser, EVM_API_METHOD_REQ_OFFSET};
use arbos::programs::memory::MemoryModel;
use arbos::programs::types::{evm_memory_cost, to_word_size, RequestType, UserOutcome};

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
fn request_type_from_u32_covers_all_variants() {
    assert_eq!(RequestType::from_u32(0), Some(RequestType::GetBytes32));
    assert_eq!(RequestType::from_u32(1), Some(RequestType::SetTrieSlots));
    assert_eq!(RequestType::from_u32(2), Some(RequestType::GetTransientBytes32));
    assert_eq!(RequestType::from_u32(3), Some(RequestType::SetTransientBytes32));
    assert_eq!(RequestType::from_u32(4), Some(RequestType::ContractCall));
    assert_eq!(RequestType::from_u32(5), Some(RequestType::DelegateCall));
    assert_eq!(RequestType::from_u32(6), Some(RequestType::StaticCall));
    assert_eq!(RequestType::from_u32(7), Some(RequestType::Create1));
    assert_eq!(RequestType::from_u32(8), Some(RequestType::Create2));
    assert_eq!(RequestType::from_u32(9), Some(RequestType::EmitLog));
    assert_eq!(RequestType::from_u32(10), Some(RequestType::AccountBalance));
    assert_eq!(RequestType::from_u32(11), Some(RequestType::AccountCode));
    assert_eq!(RequestType::from_u32(12), Some(RequestType::AccountCodeHash));
    assert_eq!(RequestType::from_u32(13), Some(RequestType::AddPages));
    assert_eq!(RequestType::from_u32(14), Some(RequestType::CaptureHostIO));
}

#[test]
fn request_type_from_u32_rejects_undefined() {
    assert_eq!(RequestType::from_u32(15), None);
    assert_eq!(RequestType::from_u32(u32::MAX), None);
}

#[test]
fn api_status_from_u8_matches_representation() {
    assert_eq!(ApiStatus::from_u8(0), Some(ApiStatus::Success));
    assert_eq!(ApiStatus::from_u8(1), Some(ApiStatus::Failure));
    assert_eq!(ApiStatus::from_u8(2), Some(ApiStatus::OutOfGas));
    assert_eq!(ApiStatus::from_u8(3), Some(ApiStatus::WriteProtection));
    assert_eq!(ApiStatus::from_u8(4), None);
}

#[test]
fn evm_api_method_offset_is_nonzero() {
    assert_eq!(EVM_API_METHOD_REQ_OFFSET, 0x1000_0000);
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

#[test]
fn parser_take_fixed_beyond_len_is_none() {
    let data = [1u8, 2, 3];
    let mut p = RequestParser::new(&data);
    assert_eq!(p.take_fixed(4), None);
    assert_eq!(p.take_fixed(3), Some(&data[..]));
    assert_eq!(p.take_fixed(1), None);
}

#[test]
fn parser_take_address() {
    let data = vec![0xAA; 20];
    let mut p = RequestParser::new(&data);
    assert_eq!(p.take_address(), Some(Address::repeat_byte(0xAA)));
    assert_eq!(p.take_address(), None);
}

#[test]
fn parser_take_hash() {
    let data = vec![0xBB; 32];
    let mut p = RequestParser::new(&data);
    assert_eq!(p.take_hash(), Some(B256::repeat_byte(0xBB)));
}

#[test]
fn parser_take_u256_big_endian() {
    let mut data = vec![0u8; 32];
    data[31] = 42;
    let mut p = RequestParser::new(&data);
    assert_eq!(p.take_u256(), Some(U256::from(42u64)));
}

#[test]
fn parser_take_u64_u32_u16() {
    let mut data = Vec::new();
    data.extend_from_slice(&0x1122_3344_5566_7788u64.to_be_bytes());
    data.extend_from_slice(&0xAABB_CCDDu32.to_be_bytes());
    data.extend_from_slice(&0xBEEFu16.to_be_bytes());
    let mut p = RequestParser::new(&data);
    assert_eq!(p.take_u64(), Some(0x1122_3344_5566_7788));
    assert_eq!(p.take_u32(), Some(0xAABB_CCDD));
    assert_eq!(p.take_u16(), Some(0xBEEF));
}

#[test]
fn parser_take_rest_consumes_remainder() {
    let data = [0x01u8, 0x02, 0x03, 0x04];
    let mut p = RequestParser::new(&data);
    let _ = p.take_fixed(2);
    assert_eq!(p.take_rest(), &[0x03, 0x04]);
    assert_eq!(p.take_rest(), &[] as &[u8]);
    assert_eq!(p.remaining(), &[] as &[u8]);
}

#[test]
fn parser_remaining_matches_untaken() {
    let data = [1u8, 2, 3, 4, 5];
    let mut p = RequestParser::new(&data);
    let _ = p.take_fixed(2);
    assert_eq!(p.remaining(), &[3, 4, 5]);
}
