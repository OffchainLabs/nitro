//! Minimal counter contract for differential fuzzing.
//!
//! Exposes:
//!   `get()              -> uint256`   storage read
//!   `increment()        -> ()`        storage read + write
//!   `add(uint256)       -> ()`        storage read + write with calldata arg
//!   `set(uint256)       -> ()`        storage write
//!
//! Rebuild with `cargo stylus get-initcode --output ../../prebuilt/counter.hex`.

#![cfg_attr(not(any(test, feature = "export-abi")), no_main)]
extern crate alloc;

use stylus_sdk::{alloy_primitives::U256, prelude::*};

sol_storage! {
    #[entrypoint]
    pub struct Counter {
        uint256 value;
    }
}

#[public]
impl Counter {
    pub fn get(&self) -> U256 {
        self.value.get()
    }

    pub fn increment(&mut self) {
        let v = self.value.get();
        self.value.set(v + U256::from(1));
    }

    pub fn add(&mut self, delta: U256) {
        let v = self.value.get();
        self.value.set(v + delta);
    }

    pub fn set(&mut self, new_value: U256) {
        self.value.set(new_value);
    }
}
