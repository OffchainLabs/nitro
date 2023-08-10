// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{evm_api::ApiCaller, Program};
use arbutil::{
    crypto,
    evm::{self, api::EvmApi, js::JsEvmApi, user::UserOutcomeKind},
    pricing::{EVM_API_INK, HOSTIO_INK, PTR_INK},
    wavm, Bytes20, Bytes32,
};
use prover::programs::meter::{GasMeteredMachine, MeteredMachine};

#[no_mangle]
pub unsafe extern "C" fn user_host__read_args(ptr: usize) {
    let program = Program::start(0);
    program.pay_for_write(program.args.len() as u64).unwrap();
    wavm::write_slice_usize(&program.args, ptr);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__write_result(ptr: usize, len: usize) {
    let program = Program::start(0);
    program.pay_for_read(len as u64).unwrap();
    program.outs = wavm::read_slice_usize(ptr, len);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__storage_load_bytes32(key: usize, dest: usize) {
    let program = Program::start(2 * PTR_INK + EVM_API_INK);
    let key = wavm::read_bytes32(key);

    let (value, gas_cost) = program.evm_api.get_bytes32(key);
    program.buy_gas(gas_cost).unwrap();
    wavm::write_bytes32(dest, value);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__storage_store_bytes32(key: usize, value: usize) {
    let program = Program::start(2 * PTR_INK + EVM_API_INK);
    program.require_gas(evm::SSTORE_SENTRY_GAS).unwrap();

    let api = &mut program.evm_api;
    let key = wavm::read_bytes32(key);
    let value = wavm::read_bytes32(value);

    let gas_cost = api.set_bytes32(key, value).unwrap();
    program.buy_gas(gas_cost).unwrap();
}

type EvmCaller<'a> = &'a mut JsEvmApi<ApiCaller>;

#[no_mangle]
pub unsafe extern "C" fn user_host__call_contract(
    contract: usize,
    calldata: usize,
    calldata_len: usize,
    value: usize,
    gas: u64,
    ret_len: usize,
) -> u8 {
    let value = Some(value);
    let call = |api: EvmCaller, contract, input, gas, value: Option<_>| {
        api.contract_call(contract, input, gas, value.unwrap())
    };
    do_call(contract, calldata, calldata_len, value, gas, ret_len, call)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__delegate_call_contract(
    contract: usize,
    calldata: usize,
    calldata_len: usize,
    gas: u64,
    ret_len: usize,
) -> u8 {
    let call = |api: EvmCaller, contract, input, gas, _| api.delegate_call(contract, input, gas);
    do_call(contract, calldata, calldata_len, None, gas, ret_len, call)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__static_call_contract(
    contract: usize,
    calldata: usize,
    calldata_len: usize,
    gas: u64,
    ret_len: usize,
) -> u8 {
    let call = |api: EvmCaller, contract, input, gas, _| api.static_call(contract, input, gas);
    do_call(contract, calldata, calldata_len, None, gas, ret_len, call)
}

unsafe fn do_call<F>(
    contract: usize,
    calldata: usize,
    calldata_len: usize,
    value: Option<usize>,
    mut gas: u64,
    return_data_len: usize,
    call: F,
) -> u8
where
    F: FnOnce(EvmCaller, Bytes20, Vec<u8>, u64, Option<Bytes32>) -> (u32, u64, UserOutcomeKind),
{
    let program = Program::start(3 * PTR_INK + EVM_API_INK);
    program.pay_for_read(calldata_len as u64).unwrap();
    gas = gas.min(program.gas_left().unwrap());

    let contract = wavm::read_bytes20(contract);
    let input = wavm::read_slice_usize(calldata, calldata_len);
    let value = value.map(|x| wavm::read_bytes32(x));
    let api = &mut program.evm_api;

    let (outs_len, gas_cost, status) = call(api, contract, input, gas, value);
    program.buy_gas(gas_cost).unwrap();
    program.evm_data.return_data_len = outs_len;
    wavm::caller_store32(return_data_len, outs_len);
    status as u8
}

#[no_mangle]
pub unsafe extern "C" fn user_host__create1(
    code: usize,
    code_len: usize,
    endowment: usize,
    contract: usize,
    revert_data_len: usize,
) {
    let program = Program::start(3 * PTR_INK + EVM_API_INK);
    program.pay_for_read(code_len as u64).unwrap();

    let code = wavm::read_slice_usize(code, code_len);
    let endowment = wavm::read_bytes32(endowment);
    let gas = program.gas_left().unwrap();
    let api = &mut program.evm_api;

    let (result, ret_len, gas_cost) = api.create1(code, endowment, gas);
    program.buy_gas(gas_cost).unwrap();
    program.evm_data.return_data_len = ret_len;
    wavm::caller_store32(revert_data_len, ret_len);
    wavm::write_bytes20(contract, result.unwrap());
}

#[no_mangle]
pub unsafe extern "C" fn user_host__create2(
    code: usize,
    code_len: usize,
    endowment: usize,
    salt: usize,
    contract: usize,
    revert_data_len: usize,
) {
    let program = Program::start(4 * PTR_INK + EVM_API_INK);
    program.pay_for_read(code_len as u64).unwrap();

    let code = wavm::read_slice_usize(code, code_len);
    let endowment = wavm::read_bytes32(endowment);
    let salt = wavm::read_bytes32(salt);
    let gas = program.gas_left().unwrap();
    let api = &mut program.evm_api;

    let (result, ret_len, gas_cost) = api.create2(code, endowment, salt, gas);
    program.buy_gas(gas_cost).unwrap();
    program.evm_data.return_data_len = ret_len;
    wavm::caller_store32(revert_data_len, ret_len);
    wavm::write_bytes20(contract, result.unwrap());
}

#[no_mangle]
pub unsafe extern "C" fn user_host__read_return_data(
    ptr: usize,
    offset: usize,
    size: usize,
) -> usize {
    let program = Program::start(EVM_API_INK);
    program.pay_for_write(size as u64).unwrap();

    let data = program.evm_api.get_return_data(offset as u32, size as u32);
    assert!(data.len() <= size);
    wavm::write_slice_usize(&data, ptr);
    data.len()
}

#[no_mangle]
pub unsafe extern "C" fn user_host__return_data_size() -> u32 {
    let program = Program::start(0);
    program.evm_data.return_data_len
}

#[no_mangle]
pub unsafe extern "C" fn user_host__emit_log(data: usize, len: u32, topics: u32) {
    let program = Program::start(EVM_API_INK);
    if topics > 4 || len < topics * 32 {
        panic!("bad topic data");
    }
    program.pay_for_read(len.into()).unwrap();
    program.pay_for_evm_log(topics, len - topics * 32).unwrap();

    let data = wavm::read_slice_usize(data, len as usize);
    program.evm_api.emit_log(data, topics).unwrap();
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_balance(address: usize, ptr: usize) {
    let program = Program::start(2 * PTR_INK + EVM_API_INK);
    let address = wavm::read_bytes20(address);

    let (value, gas_cost) = program.evm_api.account_balance(address);
    program.buy_gas(gas_cost).unwrap();
    wavm::write_bytes32(ptr, value);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_codehash(address: usize, ptr: usize) {
    let program = Program::start(2 * PTR_INK + EVM_API_INK);
    let address = wavm::read_bytes20(address);

    let (value, gas_cost) = program.evm_api.account_codehash(address);
    program.buy_gas(gas_cost).unwrap();
    wavm::write_bytes32(ptr, value);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__evm_gas_left() -> u64 {
    let program = Program::start(0);
    program.gas_left().unwrap()
}

#[no_mangle]
pub unsafe extern "C" fn user_host__evm_ink_left() -> u64 {
    let program = Program::start(0);
    program.ink_ready().unwrap()
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_basefee(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes32(ptr, program.evm_data.block_basefee)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__chainid(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes32(ptr, program.evm_data.chainid)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_coinbase(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes20(ptr, program.evm_data.block_coinbase)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_gas_limit() -> u64 {
    let program = Program::start(0);
    program.evm_data.block_gas_limit
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_number(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes32(ptr, program.evm_data.block_number)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__block_timestamp() -> u64 {
    let program = Program::start(0);
    program.evm_data.block_timestamp
}

#[no_mangle]
pub unsafe extern "C" fn user_host__contract_address(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes20(ptr, program.evm_data.contract_address)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__msg_reentrant() -> u32 {
    let program = Program::start(0);
    program.evm_data.reentrant
}

#[no_mangle]
pub unsafe extern "C" fn user_host__msg_sender(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes20(ptr, program.evm_data.msg_sender)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__msg_value(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes32(ptr, program.evm_data.msg_value)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__native_keccak256(bytes: usize, len: usize, output: usize) {
    let program = Program::start(0);
    program.pay_for_keccak(len as u64).unwrap();

    let preimage = wavm::read_slice_usize(bytes, len);
    let digest = crypto::keccak(preimage);
    wavm::write_bytes32(output, digest.into())
}

#[no_mangle]
pub unsafe extern "C" fn user_host__tx_gas_price(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes32(ptr, program.evm_data.tx_gas_price)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__tx_ink_price() -> u32 {
    let program = Program::start(0);
    program.pricing().ink_price
}

#[no_mangle]
pub unsafe extern "C" fn user_host__tx_origin(ptr: usize) {
    let program = Program::start(PTR_INK);
    wavm::write_bytes20(ptr, program.evm_data.tx_origin)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__memory_grow(pages: u16) {
    let program = Program::start_free();
    if pages == 0 {
        return program.buy_ink(HOSTIO_INK).unwrap();
    }
    let gas_cost = program.evm_api.add_pages(pages);
    program.buy_gas(gas_cost).unwrap();
}
