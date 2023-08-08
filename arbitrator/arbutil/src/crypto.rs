// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use sha3::{Digest, Keccak256};
use siphasher::sip::SipHasher24;
use std::hash::Hasher;

pub fn keccak<T: AsRef<[u8]>>(preimage: T) -> [u8; 32] {
    let mut hasher = Keccak256::new();
    hasher.update(preimage);
    hasher.finalize().into()
}

pub fn siphash(preimage: &[u8], key: &[u8; 16]) -> u64 {
    let mut hasher = SipHasher24::new_with_key(key);
    hasher.write(preimage);
    hasher.finish()
}
