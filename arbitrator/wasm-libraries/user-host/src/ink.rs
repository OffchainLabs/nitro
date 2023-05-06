// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::Program;
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
    fn ink_left(&mut self) -> MachineMeter {
        unsafe {
            match user_ink_status() {
                0 => MachineMeter::Ready(user_ink_left()),
                _ => MachineMeter::Exhausted,
            }
        }
    }

    fn set_meter(&mut self, meter: MachineMeter) {
        unsafe {
            user_set_ink(meter.ink(), meter.status());
        }
    }
}

impl GasMeteredMachine for Program {
    fn pricing(&mut self) -> PricingParams {
        self.config.pricing
    }
}
