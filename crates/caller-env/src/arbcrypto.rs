// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use core::mem::MaybeUninit;

use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
use tiny_keccak::{Hasher, Keccak};

use crate::{ExecEnv, GuestPtr, MemAccess, arbcrypto::ECRecoveryStatus::*};

#[derive(Debug, PartialEq)]
#[repr(u32)]
enum ECRecoveryStatus {
    Success = 0,
    InvalidHashLength,
    InvalidSignatureLength,
    InvalidRecoveryId,
    RecoveryFailed,
}

const HASH_LENGTH: usize = 32;
const SIGNATURE_LENGTH: usize = 64;
const SIGNATURE_WITH_ID_LENGTH: usize = SIGNATURE_LENGTH + 1;
const RECOVERY_ID_INDEX: usize = 64;

// Core ECRECOVER logic. Returns the 65-byte uncompressed public key on success.
fn ecrecover_core(
    hash: &[u8; HASH_LENGTH],
    sig_with_id: &[u8; SIGNATURE_WITH_ID_LENGTH],
) -> Result<[u8; 65], ECRecoveryStatus> {
    let mut sig = Signature::from_slice(&sig_with_id[..SIGNATURE_LENGTH]).expect("Length checked");
    let mut recovery_id = RecoveryId::from_byte(sig_with_id[RECOVERY_ID_INDEX])
        .ok_or(ECRecoveryStatus::InvalidRecoveryId)?;

    // k256 rejects high-S in verify_prehashed (EIP-2 / RFC 6979 canonicality).
    // The ECRECOVER precompile must not apply that restriction: normalize s → N−s
    // and flip the recovery point's y-parity so the recovered key is identical.
    if let Some(normalized) = sig.normalize_s() {
        sig = normalized;
        recovery_id = RecoveryId::new(!recovery_id.is_y_odd(), recovery_id.is_x_reduced());
    }

    let pubkey = VerifyingKey::recover_from_prehash(hash, &sig, recovery_id)
        .map_err(|_| ECRecoveryStatus::RecoveryFailed)?;
    let mut out = [0u8; 65];
    out.copy_from_slice(pubkey.to_encoded_point(false).as_bytes());
    Ok(out)
}

pub fn ecrecovery<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    hash_ptr: GuestPtr,
    hash_len: u32,
    sig_ptr: GuestPtr,
    sig_len: u32,
    pub_ptr: GuestPtr,
) -> u32 {
    if hash_len as usize != HASH_LENGTH {
        return InvalidHashLength as u32;
    } else if sig_len as usize != SIGNATURE_WITH_ID_LENGTH {
        return InvalidSignatureLength as u32;
    }

    let hash = mem.read_fixed::<HASH_LENGTH>(hash_ptr);
    let sig_with_id = mem.read_fixed::<SIGNATURE_WITH_ID_LENGTH>(sig_ptr);

    match ecrecover_core(&hash, &sig_with_id) {
        Ok(pubkey) => {
            mem.write_slice(pub_ptr, &pubkey);
            Success as u32
        }
        Err(e) => e as u32,
    }
}

#[cfg(test)]
mod tests {
    use k256::ecdsa::SigningKey;

    use super::*;

    const SECP256K1_N: [u8; 32] = [
        0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
        0xFE, 0xBA, 0xAE, 0xDC, 0xE6, 0xAF, 0x48, 0xA0, 0x3B, 0xBF, 0xD2, 0x5E, 0x8C, 0xD0, 0x36,
        0x41, 0x41,
    ];

    // Big-endian 256-bit subtraction: computes N − s.
    fn n_minus_s(s: &[u8; 32]) -> [u8; 32] {
        let mut result = [0u8; 32];
        let mut borrow = 0i16;
        for i in (0..32).rev() {
            let diff = (SECP256K1_N[i] as i16) - (s[i] as i16) - borrow;
            result[i] = diff.rem_euclid(256) as u8;
            borrow = if diff < 0 { 1 } else { 0 };
        }
        result
    }

    // Generate a random (hash, sig_with_id) pair using RFC 6979 signing.
    // k256 always produces a canonical low-S signature.
    fn random_low_s() -> ([u8; HASH_LENGTH], [u8; SIGNATURE_WITH_ID_LENGTH]) {
        use rand::{Rng as _, SeedableRng as _};

        // Security doesn't matter here — seed from the wall clock for variety.
        let seed = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default()
            .subsec_nanos() as u64;
        let mut rng = rand_pcg::Pcg64::seed_from_u64(seed);

        let mut key_bytes = [0u8; 32];
        rng.fill_bytes(&mut key_bytes);
        let signing_key = SigningKey::from_bytes((&key_bytes).into())
            .unwrap_or_else(|_| SigningKey::from_bytes((&[1u8; 32]).into()).unwrap());

        let mut hash = [0u8; HASH_LENGTH];
        rng.fill_bytes(&mut hash);

        let (sig, rec_id) = signing_key.sign_prehash_recoverable(&hash).unwrap();
        let mut sig_with_id = [0u8; SIGNATURE_WITH_ID_LENGTH];
        sig_with_id[..SIGNATURE_LENGTH].copy_from_slice(&sig.to_bytes());
        sig_with_id[RECOVERY_ID_INDEX] = rec_id.to_byte();
        (hash, sig_with_id)
    }

    fn high_s_version(
        sig_with_id: &[u8; SIGNATURE_WITH_ID_LENGTH],
    ) -> [u8; SIGNATURE_WITH_ID_LENGTH] {
        let s_bytes: [u8; 32] = sig_with_id[32..64].try_into().unwrap();
        let mut out = *sig_with_id;
        out[32..64].copy_from_slice(&n_minus_s(&s_bytes));
        out[RECOVERY_ID_INDEX] ^= 1; // flip y-parity bit
        out
    }

    #[test]
    fn low_s_recovers_expected_key() {
        let (hash, sig_with_id) = random_low_s();
        assert!(ecrecover_core(&hash, &sig_with_id).is_ok());
    }

    #[test]
    fn high_s_recovers_same_key_as_low_s() {
        let (hash, low_s) = random_low_s();
        let high_s = high_s_version(&low_s);
        assert_eq!(
            ecrecover_core(&hash, &low_s),
            ecrecover_core(&hash, &high_s)
        );
    }

    #[test]
    fn invalid_recovery_id_returns_error() {
        let (hash, mut sig_with_id) = random_low_s();
        sig_with_id[RECOVERY_ID_INDEX] = 4; // only 0–3 are valid
        assert_eq!(
            ecrecover_core(&hash, &sig_with_id),
            Err(ECRecoveryStatus::InvalidRecoveryId)
        );
    }
}

pub fn keccak256<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
) {
    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);

    let mut output = MaybeUninit::<[u8; 32]>::uninit();
    let mut hasher = Keccak::v256();
    hasher.update(input.as_ref());

    // SAFETY: finalize() writes 32 bytes
    unsafe {
        hasher.finalize(&mut *output.as_mut_ptr());
        mem.write_slice(out_buf_ptr, output.assume_init().as_slice());
    }
}
