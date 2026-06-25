// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::evm::{api::Gas, user::UserOutcomeKind};
use caller_env::GuestPtr;
use user_host_trait::UserHost;

use crate::program::Program;

#[link(wasm_import_module = "forward")]
unsafe extern "C" {
    fn set_trap();
}

macro_rules! hostio {
    ($($func:tt)*) => {
        match Program::current().$($func)* {
            Ok(value) => value,
            Err(_) => {
                set_trap();
                Default::default()
            }
        }
    };
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__read_args(ptr: GuestPtr) {
    unsafe { hostio!(read_args(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__exit_early(status: u32) {
    unsafe {
        hostio!(exit_early(status));
        Program::current().early_exit = Some(match status {
            0 => UserOutcomeKind::Success,
            _ => UserOutcomeKind::Revert,
        });
        set_trap();
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__write_result(ptr: GuestPtr, len: u32) {
    unsafe { hostio!(write_result(ptr, len)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__storage_load_bytes32(key: GuestPtr, dest: GuestPtr) {
    unsafe { hostio!(storage_load_bytes32(key, dest)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__storage_cache_bytes32(key: GuestPtr, value: GuestPtr) {
    unsafe { hostio!(storage_cache_bytes32(key, value)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__storage_flush_cache(clear: u32) {
    unsafe { hostio!(storage_flush_cache(clear != 0)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__transient_load_bytes32(key: GuestPtr, dest: GuestPtr) {
    unsafe { hostio!(transient_load_bytes32(key, dest)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__transient_store_bytes32(key: GuestPtr, value: GuestPtr) {
    unsafe { hostio!(transient_store_bytes32(key, value)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__call_contract(
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    value: GuestPtr,
    gas: u64,
    ret_len: GuestPtr,
) -> u8 {
    unsafe {
        hostio!(call_contract(
            contract,
            data,
            data_len,
            value,
            Gas(gas),
            ret_len
        ))
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__delegate_call_contract(
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    gas: u64,
    ret_len: GuestPtr,
) -> u8 {
    unsafe {
        hostio!(delegate_call_contract(
            contract,
            data,
            data_len,
            Gas(gas),
            ret_len
        ))
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__static_call_contract(
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    gas: u64,
    ret_len: GuestPtr,
) -> u8 {
    unsafe {
        hostio!(static_call_contract(
            contract,
            data,
            data_len,
            Gas(gas),
            ret_len
        ))
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__create1(
    code: GuestPtr,
    code_len: u32,
    value: GuestPtr,
    contract: GuestPtr,
    revert_len: GuestPtr,
) {
    unsafe { hostio!(create1(code, code_len, value, contract, revert_len)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__create2(
    code: GuestPtr,
    code_len: u32,
    value: GuestPtr,
    salt: GuestPtr,
    contract: GuestPtr,
    revert_len: GuestPtr,
) {
    unsafe { hostio!(create2(code, code_len, value, salt, contract, revert_len)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__read_return_data(
    dest: GuestPtr,
    offset: u32,
    size: u32,
) -> u32 {
    unsafe { hostio!(read_return_data(dest, offset, size)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__return_data_size() -> u32 {
    unsafe { hostio!(return_data_size()) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__emit_log(data: GuestPtr, len: u32, topics: u32) {
    unsafe { hostio!(emit_log(data, len, topics)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__account_balance(address: GuestPtr, ptr: GuestPtr) {
    unsafe { hostio!(account_balance(address, ptr)) }
}
#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__account_code(
    address: GuestPtr,
    offset: u32,
    size: u32,
    dest: GuestPtr,
) -> u32 {
    unsafe { hostio!(account_code(address, offset, size, dest)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__account_code_size(address: GuestPtr) -> u32 {
    unsafe { hostio!(account_code_size(address)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__account_codehash(address: GuestPtr, ptr: GuestPtr) {
    unsafe { hostio!(account_codehash(address, ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__block_basefee(ptr: GuestPtr) {
    unsafe { hostio!(block_basefee(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__block_coinbase(ptr: GuestPtr) {
    unsafe { hostio!(block_coinbase(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__block_gas_limit() -> u64 {
    unsafe { hostio!(block_gas_limit()) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__block_number() -> u64 {
    unsafe { hostio!(block_number()) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__block_timestamp() -> u64 {
    unsafe { hostio!(block_timestamp()) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__chainid() -> u64 {
    unsafe { hostio!(chainid()) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__contract_address(ptr: GuestPtr) {
    unsafe { hostio!(contract_address(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__evm_gas_left() -> u64 {
    unsafe { hostio!(evm_gas_left()).0 }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__evm_ink_left() -> u64 {
    unsafe { hostio!(evm_ink_left()).0 }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__math_div(value: GuestPtr, divisor: GuestPtr) {
    unsafe { hostio!(math_div(value, divisor)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__math_mod(value: GuestPtr, modulus: GuestPtr) {
    unsafe { hostio!(math_mod(value, modulus)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__math_pow(value: GuestPtr, exponent: GuestPtr) {
    unsafe { hostio!(math_pow(value, exponent)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__math_add_mod(
    value: GuestPtr,
    addend: GuestPtr,
    modulus: GuestPtr,
) {
    unsafe { hostio!(math_add_mod(value, addend, modulus)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__math_mul_mod(
    value: GuestPtr,
    multiplier: GuestPtr,
    modulus: GuestPtr,
) {
    unsafe { hostio!(math_mul_mod(value, multiplier, modulus)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__msg_reentrant() -> u32 {
    unsafe { hostio!(msg_reentrant()) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__msg_sender(ptr: GuestPtr) {
    unsafe { hostio!(msg_sender(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__msg_value(ptr: GuestPtr) {
    unsafe { hostio!(msg_value(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__native_keccak256(input: GuestPtr, len: u32, output: GuestPtr) {
    unsafe { hostio!(native_keccak256(input, len, output)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__tx_gas_price(ptr: GuestPtr) {
    unsafe { hostio!(tx_gas_price(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__tx_ink_price() -> u32 {
    unsafe { hostio!(tx_ink_price()) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__tx_origin(ptr: GuestPtr) {
    unsafe { hostio!(tx_origin(ptr)) }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn user_host__pay_for_memory_grow(pages: u32) {
    unsafe { hostio!(pay_for_memory_grow(pages)) }
}
