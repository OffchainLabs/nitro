use crate::arbcrypto::ECRecoveryStatus::*;
use crate::{ExecEnv, GuestPtr, MemAccess};
use core::mem::MaybeUninit;
use k256::ecdsa::{RecoveryId, Signature, VerifyingKey};
use tiny_keccak::{Hasher, Keccak};

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

    let sig_bytes = mem.read_fixed::<SIGNATURE_WITH_ID_LENGTH>(sig_ptr);
    let sig = Signature::from_slice(&sig_bytes[..SIGNATURE_LENGTH]).expect("Length checked");
    let Some(recovery_id) = RecoveryId::from_byte(sig_bytes[RECOVERY_ID_INDEX]) else {
        return InvalidRecoveryId as u32;
    };

    let Ok(pubkey) = VerifyingKey::recover_from_prehash(&hash, &sig, recovery_id) else {
        return RecoveryFailed as u32;
    };

    mem.write_slice(pub_ptr, pubkey.to_encoded_point(false).as_bytes());
    Success as u32
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
