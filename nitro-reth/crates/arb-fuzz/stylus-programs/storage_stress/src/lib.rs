//! Hammer storage cache + flush.
//!
//! Exposes:
//!   `write_range(uint256 start, uint256 count, uint256 base) -> ()`
//!       writes count slots from `start..start+count`, value = base ^ i
//!   `read_range(uint256 start, uint256 count)                -> uint256`
//!       reads count slots and xor-folds them, returns the accumulator
//!   `flush(bool clear)                                       -> ()`
//!       explicit flush invocation for cache-vs-flush gas comparisons
//!
//! Rebuild: `cargo stylus get-initcode --output ../../prebuilt/storage_stress.hex`.

#![cfg_attr(not(any(test, feature = "export-abi")), no_main)]
extern crate alloc;

use stylus_sdk::{
    alloy_primitives::{B256, U256},
    prelude::*,
};

#[storage]
#[entrypoint]
pub struct StorageStress {}

#[public]
impl StorageStress {
    pub fn write_range(&mut self, start: U256, count: U256, base: U256) {
        let n: u64 = count.try_into().unwrap_or(0);
        let n = n.min(64);
        for i in 0..n {
            let slot = start + U256::from(i);
            let val = base ^ U256::from(i);
            unsafe {
                self.vm()
                    .storage_cache_bytes32(slot, B256::from(val.to_be_bytes::<32>()));
            }
        }
        self.vm().flush_cache(false);
    }

    pub fn read_range(&self, start: U256, count: U256) -> U256 {
        let n: u64 = count.try_into().unwrap_or(0);
        let n = n.min(64);
        let mut acc = U256::ZERO;
        for i in 0..n {
            let slot = start + U256::from(i);
            let v = self.vm().storage_load_bytes32(slot);
            acc ^= U256::from_be_bytes(v.0);
        }
        acc
    }

    pub fn flush(&mut self, clear: bool) {
        self.vm().flush_cache(clear);
    }
}
