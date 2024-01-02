// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::program::Program;
use user_host_trait::UserHost;

#[link(wasm_import_module = "forward")]
extern "C" {
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

#[no_mangle]
pub unsafe extern "C" fn user_host__read_args(ptr: u32) {
    hostio!(read_args(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__write_result(ptr: u32, len: u32) {
    hostio!(write_result(ptr, len))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__storage_load_bytes32(key: u32, dest: u32) {
    hostio!(storage_load_bytes32(key, dest))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__storage_store_bytes32(key: u32, value: u32) {
    hostio!(storage_store_bytes32(key, value))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__call_contract(
    contract: u32,
    data: u32,
    data_len: u32,
    value: u32,
    gas: u64,
    ret_len: u32,
) -> u8 {
    hostio!(call_contract(contract, data, data_len, value, gas, ret_len))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__delegate_call_contract(
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> u8 {
    hostio!(delegate_call_contract(
        contract, data, data_len, gas, ret_len
    ))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__static_call_contract(
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> u8 {
    hostio!(static_call_contract(contract, data, data_len, gas, ret_len))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__create1(
    code: u32,
    code_len: u32,
    value: u32,
    contract: u32,
    revert_len: u32,
) {
    hostio!(create1(code, code_len, value, contract, revert_len))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__create2(
    code: u32,
    code_len: u32,
    value: u32,
    salt: u32,
    contract: u32,
    revert_len: u32,
) {
    hostio!(create2(code, code_len, value, salt, contract, revert_len))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__read_return_data(dest: u32, offset: u32, size: u32) -> u32 {
    hostio!(read_return_data(dest, offset, size))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__return_data_size() -> u32 {
    hostio!(return_data_size())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__emit_log(data: u32, len: u32, topics: u32) {
    hostio!(emit_log(data, len, topics))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_balance(address: u32, ptr: u32) {
    hostio!(account_balance(address, ptr))
}
#[no_mangle]
pub unsafe extern "C" fn user_host__account_code(address: u32, offset: u32, size: u32, dest: u32) -> u32 {
    hostio!(account_code(address, offset, size, dest))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_codehash(address: u32, ptr: u32) {
    hostio!(account_codehash(address, ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_code_size(address: u32) -> u32 {
    hostio!(account_code_size(address))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_basefee(ptr: u32) {
    hostio!(block_basefee(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_coinbase(ptr: u32) {
    hostio!(block_coinbase(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_gas_limit() -> u64 {
    hostio!(block_gas_limit())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_number() -> u64 {
    hostio!(block_number())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_timestamp() -> u64 {
    hostio!(block_timestamp())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__chainid() -> u64 {
    hostio!(chainid())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__contract_address(ptr: u32) {
    hostio!(contract_address(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__evm_gas_left() -> u64 {
    hostio!(evm_gas_left())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__evm_ink_left() -> u64 {
    hostio!(evm_ink_left())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__msg_reentrant() -> u32 {
    hostio!(msg_reentrant())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__msg_sender(ptr: u32) {
    hostio!(msg_sender(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__msg_value(ptr: u32) {
    hostio!(msg_value(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__native_keccak256(input: u32, len: u32, output: u32) {
    hostio!(native_keccak256(input, len, output))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__tx_gas_price(ptr: u32) {
    hostio!(tx_gas_price(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__tx_ink_price() -> u32 {
    hostio!(tx_ink_price())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__tx_origin(ptr: u32) {
    hostio!(tx_origin(ptr))
}

#[no_mangle]
pub unsafe extern "C" fn user_host__pay_for_memory_grow(pages: u16) {
    hostio!(pay_for_memory_grow(pages))
}
