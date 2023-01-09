// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use gas::PricingParams;

mod gas;
mod link;
mod user;

static mut PROGRAMS: Vec<Program> = vec![];

struct Program {
    args: Vec<u8>,
    outs: Vec<u8>,
    pricing: PricingParams,
}

impl Program {
    pub fn new(args: Vec<u8>, pricing: PricingParams) -> Self {
        let outs = vec![];
        Self {
            args,
            outs,
            pricing,
        }
    }
}

#[no_mangle]
pub unsafe extern "C" fn user_host__push_program(len: usize, price: u64, hostio: u64) -> *const u8 {
    let args = vec![0; len];
    let pricing = PricingParams::new(price, hostio);
    let program = Program::new(args, pricing);
    let data = program.args.as_ptr();
    PROGRAMS.push(program);
    data
}

#[no_mangle]
pub unsafe extern "C" fn user_host__pop_program() -> usize {
    PROGRAMS.pop();
    PROGRAMS.len()
}

#[no_mangle]
pub unsafe extern "C" fn user_host__get_output_len() -> usize {
    let program = PROGRAMS.last().expect("no program");
    program.outs.len()
}

#[no_mangle]
pub unsafe extern "C" fn user_host__get_output_ptr() -> *const u8 {
    let program = PROGRAMS.last().expect("no program");
    program.outs.as_ptr()
}
