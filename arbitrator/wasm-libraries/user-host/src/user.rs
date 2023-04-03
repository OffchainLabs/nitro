// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::PROGRAMS;
use arbutil::wavm;

#[no_mangle]
pub unsafe extern "C" fn user_host__read_args(ptr: usize) {
    let program = PROGRAMS.last().expect("no program");
    program.pricing.begin();
    program.pricing.pay_for_evm_copy(program.args.len());
    wavm::write_slice_usize(&program.args, ptr);
}

#[no_mangle]
pub unsafe extern "C" fn user_host__return_data(ptr: usize, len: usize) {
    let program = PROGRAMS.last_mut().expect("no program");
    program.pricing.begin();
    program.pricing.pay_for_evm_copy(len);
    program.outs = wavm::read_slice_usize(ptr, len);
}
