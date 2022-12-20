// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use sha3::{Digest, Keccak256};

pub fn keccak(preimage: &[u8]) -> [u8; 32] {
    let mut hasher = Keccak256::new();
    hasher.update(preimage);
    hasher.finalize().into()
}
