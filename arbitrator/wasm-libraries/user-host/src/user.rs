// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::Program;
use arbutil::{
    evm::{self, api::EvmApi},
    wavm,
};
use prover::programs::meter::{GasMeteredMachine, MeteredMachine};

#[no_mangle]
pub unsafe extern "C" fn user_host__read_args(ptr: usize) {
    let program = Program::start();
    program.pay_for_evm_copy(program.args.len() as u64).unwrap();
    wavm::write_slice_usize(&program.args, ptr);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__return_data(ptr: usize, len: usize) {
    let program = Program::start();
    program.pay_for_evm_copy(len as u64).unwrap();
    program.outs = wavm::read_slice_usize(ptr, len);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_load_bytes32(key: usize, dest: usize) {
    let program = Program::start();
    let key = wavm::read_bytes32(key);

    let (value, gas_cost) = program.evm_api.get_bytes32(key.into());
    program.buy_gas(gas_cost).unwrap();
    wavm::write_slice_usize(&value.0, dest);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_store_bytes32(key: usize, value: usize) {
    let program = Program::start();
    program.require_gas(evm::SSTORE_SENTRY_GAS).unwrap();

    let api = &mut program.evm_api;
    let key = wavm::read_bytes32(key);
    let value = wavm::read_bytes32(value);

    let gas_cost = api.set_bytes32(key.into(), value.into()).unwrap();
    program.buy_gas(gas_cost).unwrap();
}

#[no_mangle]
pub unsafe extern "C" fn user_host__call_contract(
    contract: usize,
    calldata: usize,
    calldata_len: usize,
    value: usize,
    mut ink: u64,
    return_data_len: usize,
) -> u8 {
    let program = Program::start();
    program.pay_for_evm_copy(calldata_len as u64).unwrap();
    ink = ink.min(program.ink_ready().unwrap());

    let gas = program.pricing().ink_to_gas(ink);
    let contract = wavm::read_bytes20(contract).into();
    let input = wavm::read_slice_usize(calldata, calldata_len);
    let value = wavm::read_bytes32(value).into();
    let api = &mut program.evm_api;

    let (outs_len, gas_cost, status) = api.contract_call(contract, input, gas, value);
    program.evm_data.return_data_len = outs_len;
    wavm::caller_store32(return_data_len, outs_len);
    program.buy_gas(gas_cost).unwrap();
    status as u8
}

#[no_mangle]
pub unsafe extern "C" fn user_host__delegate_call_contract(
    contract: usize,
    calldata: usize,
    calldata_len: usize,
    mut ink: u64,
    return_data_len: usize,
) -> u8 {
    let program = Program::start();
    program.pay_for_evm_copy(calldata_len as u64).unwrap();
    ink = ink.min(program.ink_ready().unwrap());

    let gas = program.pricing().ink_to_gas(ink);
    let contract = wavm::read_bytes20(contract).into();
    let input = wavm::read_slice_usize(calldata, calldata_len);
    let api = &mut program.evm_api;

    let (outs_len, gas_cost, status) = api.delegate_call(contract, input, gas);
    program.evm_data.return_data_len = outs_len;
    wavm::caller_store32(return_data_len, outs_len);
    program.buy_gas(gas_cost).unwrap();
    status as u8
}

#[no_mangle]
pub unsafe extern "C" fn user_host__static_call_contract(
    contract: usize,
    calldata: usize,
    calldata_len: usize,
    mut ink: u64,
    return_data_len: usize,
) -> u8 {
    let program = Program::start();
    program.pay_for_evm_copy(calldata_len as u64).unwrap();
    ink = ink.min(program.ink_ready().unwrap());

    let gas = program.pricing().ink_to_gas(ink);
    let contract = wavm::read_bytes20(contract).into();
    let input = wavm::read_slice_usize(calldata, calldata_len);
    let api = &mut program.evm_api;

    let (outs_len, gas_cost, status) = api.static_call(contract, input, gas);
    program.evm_data.return_data_len = outs_len;
    wavm::caller_store32(return_data_len, outs_len);
    program.buy_gas(gas_cost).unwrap();
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
    let program = Program::start();
    program.pay_for_evm_copy(code_len as u64).unwrap();

    let code = wavm::read_slice_usize(code, code_len);
    let endowment = wavm::read_bytes32(endowment).into();
    let gas = program.gas_left().unwrap();
    let api = &mut program.evm_api;

    let (result, ret_len, gas_cost) = api.create1(code, endowment, gas);
    program.evm_data.return_data_len = ret_len;
    wavm::caller_store32(revert_data_len, ret_len);
    program.buy_gas(gas_cost).unwrap();
    wavm::write_slice_usize(&result.unwrap().0, contract);
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
    let program = Program::start();
    program.pay_for_evm_copy(code_len as u64).unwrap();

    let code = wavm::read_slice_usize(code, code_len);
    let endowment = wavm::read_bytes32(endowment).into();
    let salt = wavm::read_bytes32(salt).into();
    let gas = program.gas_left().unwrap();
    let api = &mut program.evm_api;

    let (result, ret_len, gas_cost) = api.create2(code, endowment, salt, gas);
    program.evm_data.return_data_len = ret_len;
    wavm::caller_store32(revert_data_len, ret_len);
    program.buy_gas(gas_cost).unwrap();
    wavm::write_slice_usize(&result.unwrap().0, contract);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__read_return_data(dest: usize) {
    let program = Program::start();
    let len = program.evm_data.return_data_len;
    program.pay_for_evm_copy(len.into()).unwrap();

    let data = program.evm_api.get_return_data();
    wavm::write_slice_usize(&data, dest);
    assert_eq!(data.len(), len as usize);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__return_data_size() -> u32 {
    let program = Program::start();
    program.evm_data.return_data_len
}

#[no_mangle]
pub unsafe extern "C" fn user_host__emit_log(data: usize, len: u32, topics: u32) {
    let program = Program::start();
    if topics > 4 || len < topics * 32 {
        panic!("bad topic data");
    }
    program.pay_for_evm_log(topics, len - topics * 32).unwrap();

    let data = wavm::read_slice_usize(data, len as usize);
    program.evm_api.emit_log(data, topics).unwrap();
}

#[no_mangle]
pub unsafe extern "C" fn user_host__tx_origin(ptr: usize) {
    let program = Program::start();
    let origin = program.evm_data.origin;
    wavm::write_slice_usize(&origin.0, ptr)
}
