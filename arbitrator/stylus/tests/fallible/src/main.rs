// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use stylus_sdk::stylus_proc::entrypoint;

/// A program that will fail on certain inputs
#[entrypoint]
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    if input[0] == 0 {
        core::arch::wasm32::unreachable()
    } else {
        Ok(input)
    }
}
