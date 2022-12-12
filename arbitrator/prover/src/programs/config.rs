// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use wasmer::wasmparser::Operator;

#[repr(C)]
pub struct PolyglotConfig {
    pub costs: fn(&Operator) -> u64,
    pub start_gas: u64,
}

impl Default for PolyglotConfig {
    fn default() -> Self {
        let costs = |_: &Operator| 0;
        Self {
            costs,
            start_gas: 0,
        }
    }
}
