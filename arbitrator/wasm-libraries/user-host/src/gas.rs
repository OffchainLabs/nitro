// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use prover::programs::config::PricingParams;
use std::ops::Deref;

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn user_gas_left() -> u64;
    fn user_gas_status() -> u32;
    fn user_set_gas(gas: u64, status: u32);
}

pub(crate) struct Pricing(pub PricingParams);

impl Deref for Pricing {
    type Target = PricingParams;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl Pricing {
    pub fn begin(&self) {
        self.buy_gas(self.hostio_cost)
    }

    pub fn buy_gas(&self, gas: u64) {
        unsafe {
            if user_gas_status() != 0 {
                panic!("out of gas");
            }
            let gas_left = user_gas_left();
            if gas_left < gas {
                panic!("out of gas");
            }
            user_set_gas(gas_left - gas, 0);
        }
    }

    #[allow(clippy::inconsistent_digit_grouping)]
    pub fn buy_evm_gas(&self, evm: u64) {
        let wasm_gas = evm.saturating_mul(100_00) / self.wasm_gas_price;
        self.buy_gas(wasm_gas)
    }
}
