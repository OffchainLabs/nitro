// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use prover::programs::prelude::StylusConfig;

mod ink;
mod link;
mod user;

pub(crate) static mut PROGRAMS: Vec<Program> = vec![];

pub(crate) struct Program {
    args: Vec<u8>,
    outs: Vec<u8>,
    config: StylusConfig,
}

impl Program {
    pub fn new(args: Vec<u8>, config: StylusConfig) -> Self {
        Self {
            args,
            outs: vec![],
            config,
        }
    }

    pub fn into_outs(self) -> Vec<u8> {
        self.outs
    }

    pub fn start() -> &'static mut Self {
        let program = unsafe { PROGRAMS.last_mut().expect("no program") };
        program.buy_ink(program.config.pricing.hostio_ink);
        program
    }

    pub fn start_free() -> &'static mut Self {
        unsafe { PROGRAMS.last_mut().expect("no program") }
    }
}

#[no_mangle]
pub unsafe extern "C" fn user_host__push_program(
    len: usize,
    version: u32,
    max_depth: u32,
    ink_price: u64,
    hostio_ink: u64,
) -> *const u8 {
    let args = vec![0; len];
    let config = StylusConfig::new(version, max_depth, ink_price, hostio_ink);
    let program = Program::new(args, config);
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
