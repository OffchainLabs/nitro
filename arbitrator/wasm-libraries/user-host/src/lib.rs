// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::{js::JsEvmApi, EvmData};
use evm_api::ApiCaller;
use prover::programs::{meter::MeteredMachine, prelude::StylusConfig};

mod evm_api;
mod ink;
mod link;
mod user;

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

    pub fn start() -> &'static mut Self {
        let program = unsafe { PROGRAMS.last_mut().expect("no program") };
        program.buy_ink(program.config.pricing.hostio_ink).unwrap();
        program
    }
}
