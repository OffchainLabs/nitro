// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::Program;
use arbutil::evm;

#[link(wasm_import_module = "hostio")]
extern "C" {
    fn user_ink_left() -> u64;
    fn user_ink_status() -> u32;
    fn user_set_ink(ink: u64, status: u32);
}

impl Program {
    pub fn buy_ink(&self, ink: u64) {
        unsafe {
            if user_ink_status() != 0 {
                panic!("out of ink");
            }
            let ink_left = user_ink_left();
            if ink_left < ink {
                panic!("out of ink");
            }
            user_set_ink(ink_left - ink, 0);
        }
    }

    #[allow(clippy::inconsistent_digit_grouping)]
    pub fn buy_gas(&self, gas: u64) {
        let ink = gas.saturating_mul(100_00) / self.config.pricing.ink_price;
        self.buy_ink(ink)
    }

    pub fn pay_for_evm_copy(&self, bytes: usize) {
        let evm_words = |count: u64| count.saturating_mul(31) / 32;
        let gas = evm_words(bytes as u64).saturating_mul(evm::COPY_WORD_GAS);
        self.buy_gas(gas)
    }
}
