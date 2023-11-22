// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::{
    evm::{js::JsEvmApi, EvmData},
    pricing,
};
use evm_api::ApiCaller;
use prover::programs::{meter::MeteredMachine, prelude::StylusConfig};

mod evm_api;
mod guard;
mod host;
mod ink;
mod link;

pub(crate) static mut PROGRAMS: Vec<Program> = vec![];

pub(crate) struct Program {
    args: Vec<u8>,
    outs: Vec<u8>,
    evm_api: JsEvmApi<ApiCaller>,
    evm_data: EvmData,
    config: StylusConfig,
}

impl Program {
    pub fn new(
        args: Vec<u8>,
        evm_api: JsEvmApi<ApiCaller>,
        evm_data: EvmData,
        config: StylusConfig,
    ) -> Self {
        Self {
            args,
            outs: vec![],
            evm_api,
            evm_data,
            config,
        }
    }

    pub fn into_outs(self) -> Vec<u8> {
        self.outs
    }

    pub fn start(cost: u64) -> &'static mut Self {
        let program = Self::start_free();
        program.buy_ink(pricing::HOSTIO_INK + cost).unwrap();
        program
    }

    pub fn start_free() -> &'static mut Self {
        unsafe { PROGRAMS.last_mut().expect("no program") }
    }
}
