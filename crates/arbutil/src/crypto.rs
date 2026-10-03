// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::mem::MaybeUninit;

use tiny_keccak::{Hasher, Keccak};

use crate::Bytes32;

pub fn keccak<T: AsRef<[u8]>>(preimage: T) -> [u8; 32] {
    *keccak_seq(&[preimage.as_ref()])
}

/// Hashes the sequential concatenation of all `inputs` with Keccak-256,
/// equivalent to pre-concatenating the slices and hashing the result.
pub fn keccak_seq(inputs: &[&[u8]]) -> Bytes32 {
    let mut h = Keccak::v256();
    for input in inputs {
        h.update(input);
    }
    // SAFETY: `&mut *out.as_mut_ptr()` produces a `&mut [u8; 32]`, coercing to a
    // `&mut [u8]` of length 32. `Keccak::v256().finalize()` writes `output.len()`
    // bytes, so all 32 bytes are initialized before `assume_init` is called.
    unsafe {
        let mut out = MaybeUninit::<[u8; 32]>::uninit();
        h.finalize(&mut *out.as_mut_ptr());
        out.assume_init().into()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn keccak_seq_concatenation_equivalence() {
        let a = b"hello";
        let b = b"world";
        let combined = [a.as_ref(), b.as_ref()].concat();
        assert_eq!(keccak_seq(&[a, b]), keccak_seq(&[&combined]));
    }

    #[test]
    fn keccak_seq_empty_known_vector() {
        // Keccak-256("") = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
        let expected: [u8; 32] = [
            0xc5, 0xd2, 0x46, 0x01, 0x86, 0xf7, 0x23, 0x3c, 0x92, 0x7e, 0x7d, 0xb2, 0xdc, 0xc7,
            0x03, 0xc0, 0xe5, 0x00, 0xb6, 0x53, 0xca, 0x82, 0x27, 0x3b, 0x7b, 0xfa, 0xd8, 0x04,
            0x5d, 0x85, 0xa4, 0x70,
        ];
        assert_eq!(*keccak_seq(&[b""]), expected);
    }

    #[test]
    fn keccak_seq_matches_direct_update_finalize() {
        let inputs: &[&[u8]] = &[b"hello", b", ", b"world"];

        let mut h = Keccak::v256();
        for input in inputs {
            h.update(input);
        }
        let mut expected = [0u8; 32];
        h.finalize(&mut expected);

        assert_eq!(*keccak_seq(inputs), expected);
    }
}
