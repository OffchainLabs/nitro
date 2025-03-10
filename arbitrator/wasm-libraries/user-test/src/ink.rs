// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{program::Program, CONFIG};
use arbutil::evm::api::Ink;
use prover::programs::{
    config::PricingParams,
    prelude::{GasMeteredMachine, MachineMeter, MeteredMachine},
};

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn user_ink_left() -> u64;
    fn user_ink_status() -> u32;
    fn user_set_ink(ink: u64, status: u32);
}

impl MeteredMachine for Program {
    fn ink_left(&self) -> MachineMeter {
        unsafe {
            match user_ink_status() {
                0 => MachineMeter::Ready(Ink(user_ink_left())),
                _ => MachineMeter::Exhausted,
            }
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        unsafe {
            user_set_ink(meter.ink().0, meter.status());
        }
    }
}

impl GasMeteredMachine for Program {
    fn pricing(&self) -> PricingParams {
        unsafe { CONFIG.unwrap().pricing }
    }
}
