// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

pub(crate) struct PricingParams {
    /// The price of wasm gas, measured in bips of an evm gas
    pub wasm_gas_price: u64,
    /// The amount of wasm gas one pays to do a user_host call
    pub hostio_cost: u64,
}

impl PricingParams {
    pub fn new(wasm_gas_price: u64, hostio_cost: u64) -> Self {
        Self { wasm_gas_price, hostio_cost }
    }

    pub fn begin(&self) {
        self.buy_gas(self.hostio_cost)
    }

    pub fn buy_gas(&self, gas: u64) {
        // TODO: actually buy gas
    }

    #[allow(clippy::inconsistent_digit_grouping)]
    pub fn buy_evm_gas(&self, evm: u64) {
        let wasm_gas = evm.saturating_mul(self.wasm_gas_price) / 100_00;
        self.buy_gas(wasm_gas)
    }
}
