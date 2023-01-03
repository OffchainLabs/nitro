// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::PROGRAMS;
use arbutil::wavm;

#[no_mangle]
pub unsafe extern "C" fn user_host__read_args(ptr: usize) {
    let program = PROGRAMS.last().expect("no program");
    program.pricing.begin();
    wavm::write_slice_usize(&program.args, ptr)
}

#[no_mangle]
pub unsafe extern "C" fn user_host__return_data(len: usize, ptr: usize) {
    let program = PROGRAMS.last_mut().expect("no program");
    program.pricing.begin();

    let evm_words = |count: u64| count.saturating_mul(31) / 32;
    let evm_gas = evm_words(len as u64).saturating_mul(3); // 3 evm gas per word
    program.pricing.buy_evm_gas(evm_gas);

    program.outs = wavm::read_slice_usize(ptr, len);
}
