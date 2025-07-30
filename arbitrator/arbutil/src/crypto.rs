// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use siphasher::sip::SipHasher24;
use tiny_keccak::{Hasher, Keccak};

pub fn keccak<T: AsRef<[u8]>>(preimage: T) -> [u8; 32] {
    let mut output = [0u8; 32];
    let mut hasher = Keccak::v256();
    hasher.update(preimage.as_ref());
    hasher.finalize(&mut output);
    output
}

pub fn siphash(preimage: &[u8], key: &[u8; 16]) -> u64 {
    use std::hash::Hasher;
    let mut hasher = SipHasher24::new_with_key(key);
    hasher.write(preimage);
    hasher.finish()
}
