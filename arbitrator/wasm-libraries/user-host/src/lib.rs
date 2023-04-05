// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use gas::Pricing;
use prover::programs::config::PricingParams;

mod gas;
mod link;
mod user;

pub(crate) static mut PROGRAMS: Vec<Program> = vec![];

pub(crate) struct Program {
    args: Vec<u8>,
    outs: Vec<u8>,
    pricing: Pricing,
}

impl Program {
    pub fn new(args: Vec<u8>, params: PricingParams) -> Self {
        Self {
            args,
            outs: vec![],
            pricing: Pricing(params),
        }
    }

    pub fn into_outs(self) -> Vec<u8> {
        self.outs
    }
}

#[no_mangle]
pub unsafe extern "C" fn user_host__push_program(
    len: usize,
    price: u64,
    hostio: u64,
    memory_fill: u64,
    memory_copy: u64,
) -> *const u8 {
    let args = vec![0; len];
    let pricing = PricingParams::new(price, hostio, memory_fill, memory_copy);
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
