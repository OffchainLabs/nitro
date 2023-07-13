// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use sha3::{Digest, Keccak256};

stylus_sdk::entrypoint!(user_main);

fn user_main(input: Vec<u8>) -> Result<Vec<u8>, Vec<u8>> {
    let mut data = keccak(&input[1..]);
    let rounds = input[0];
    for _ in 1..rounds {
        data = keccak(&data);
    }
    Ok(data.as_ref().into())
}

fn keccak(preimage: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak256::new();
    hasher.update(preimage);
    hasher.finalize().into()
}
