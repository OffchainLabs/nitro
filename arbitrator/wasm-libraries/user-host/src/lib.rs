// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use prover::programs::prelude::{EvmApi, EvmData, StylusConfig};

mod ink;
mod link;
mod user;

pub(crate) static mut PROGRAMS: Vec<Program> = vec![];

pub(crate) struct Program {
    args: Vec<u8>,
    outs: Vec<u8>,
    //evm_api: EvmApi,
    evm_data: EvmData,
    config: StylusConfig,
}

impl Program {
    pub fn new(args: Vec<u8>, evm_data: EvmData, config: StylusConfig) -> Self {
        Self {
            args,
            outs: vec![],
            evm_data,
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
}

/// Pushes a user program without taking the canonical path in link.rs
///
/// # Safety
///
/// non-reentrant and test-only
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
    let program = Program::new(args, EvmData::default(), config);
    let data = program.args.as_ptr();
    PROGRAMS.push(program);
    data
}

/// Removes a user program without taking the canonical path in link.rs
///
/// # Safety
///
/// non-reentrant and test-only
#[no_mangle]
pub unsafe extern "C" fn user_host__pop_program() -> usize {
    PROGRAMS.pop();
    PROGRAMS.len()
}

/// Gets the length of the current program's output
///
/// # Safety
///
/// non-reentrant and test-only
#[no_mangle]
pub unsafe extern "C" fn user_host__get_output_len() -> usize {
    let program = PROGRAMS.last().expect("no program");
    program.outs.len()
}

/// Gets the pointer to the current program's output
///
/// # Safety
///
/// non-reentrant and test-only
#[no_mangle]
pub unsafe extern "C" fn user_host__get_output_ptr() -> *const u8 {
    let program = PROGRAMS.last().expect("no program");
    program.outs.as_ptr()
}
