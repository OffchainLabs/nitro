// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![no_main]

use stylus_sdk::{host::VM, prelude::*};

/// A program that will fail on certain inputs
#[entrypoint]
fn user_main(input: Vec<u8>, _vm: VM) -> Result<Vec<u8>, Vec<u8>> {
    if input[0] == 0 {
        core::arch::wasm32::unreachable()
    } else {
        Ok(input)
    }
}
