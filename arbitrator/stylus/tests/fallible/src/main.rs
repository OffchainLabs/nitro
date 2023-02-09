// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

arbitrum::arbitrum_main!(user_main);

/// A program that will fail on certain inputs
fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    if input[0] == 0 {
        core::arch::wasm32::unreachable()
    } else {
        return Ok(input)
    }
}
