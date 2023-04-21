// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::Program;
use arbutil::wavm;

#[no_mangle]
pub unsafe extern "C" fn user_host__read_args(ptr: usize) {
    let program = Program::start();
    program.pay_for_evm_copy(program.args.len());
    wavm::write_slice_usize(&program.args, ptr);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__return_data(ptr: usize, len: usize) {
    let program = Program::start();
    program.pay_for_evm_copy(len);
    program.outs = wavm::read_slice_usize(ptr, len);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_load_bytes32(key: usize, dest: usize) {
    let program = Program::start();
    let key = wavm::read_bytes32(key);
    let value = [0; 32];
    wavm::write_slice_usize(&value, dest);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__account_store_bytes32(key: usize, value: usize) {
    let program = Program::start();
    let key = wavm::read_bytes32(key);
    let value = wavm::read_bytes32(value);
    program.buy_gas(2200);
    println!("STORE: {} {}", hex::encode(key), hex::encode(value));
}

#[no_mangle]
pub unsafe extern "C" fn console__log_txt(ptr: usize, len: usize) {
    let program = Program::start_free();
    //env.say(Value::from(value.into()));
}
