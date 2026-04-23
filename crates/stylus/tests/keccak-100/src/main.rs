// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![no_main]

use sha3::{Digest, Keccak256};
use stylus_sdk::{host::VM, prelude::*};

#[entrypoint]
fn user_main(_: Vec<u8>, _vm: VM) -> Result<Vec<u8>, Vec<u8>> {
    let mut data = [0; 32];
    for _ in 0..100 {
        data = keccak(&data);
    }
    assert_ne!(data, [0; 32]);
    Ok(data.as_ref().into())
}

fn keccak(preimage: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak256::new();
    hasher.update(preimage);
    hasher.finalize().into()
}
