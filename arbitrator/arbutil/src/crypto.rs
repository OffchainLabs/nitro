// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use siphasher::sip::SipHasher24;
use std::mem::MaybeUninit;
use tiny_keccak::{Hasher, Keccak};

pub fn keccak<T: AsRef<[u8]>>(preimage: T) -> [u8; 32] {
    let mut output = MaybeUninit::<[u8; 32]>::uninit();
    let mut hasher = Keccak::v256();
    hasher.update(preimage.as_ref());

    // SAFETY: finalize() writes 32 bytes
    unsafe {
        hasher.finalize(&mut *output.as_mut_ptr());
        output.assume_init()
    }
}

pub fn siphash(preimage: &[u8], key: &[u8; 16]) -> u64 {
    use std::hash::Hasher;
    let mut hasher = SipHasher24::new_with_key(key);
    hasher.write(preimage);
    hasher.finish()
}
