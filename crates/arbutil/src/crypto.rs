// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::mem::MaybeUninit;

use siphasher::sip::SipHasher24;
use tiny_keccak::{Hasher, Keccak};

use crate::Bytes32;

pub fn keccak<T: AsRef<[u8]>>(preimage: T) -> [u8; 32] {
    *keccak_seq(&[preimage.as_ref()])
}

/// Hashes the concatenation of all `inputs` with Keccak-256.
pub fn keccak_seq(inputs: &[&[u8]]) -> Bytes32 {
    let mut h = Keccak::v256();
    for input in inputs {
        h.update(input);
    }
    // SAFETY: finalize() writes exactly 32 bytes
    unsafe {
        let mut out = MaybeUninit::<[u8; 32]>::uninit();
        h.finalize(&mut *out.as_mut_ptr());
        out.assume_init().into()
    }
}
