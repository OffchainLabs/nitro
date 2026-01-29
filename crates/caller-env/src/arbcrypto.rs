use crate::arbcrypto::ECRecoveryStatus::*;
use crate::{ExecEnv, GuestPtr, MemAccess};
use core::mem::MaybeUninit;
use secp256k1::ecdsa::{RecoverableSignature, RecoveryId};
use secp256k1::Message;
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
const SIGNATURE_LENGTH: usize = 65; // 64 bytes for actual signature + 1 byte for recovery id
const SIGNATURE_RECOVERY_ID_INDEX: usize = 64;

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
    } else if sig_len as usize != SIGNATURE_LENGTH {
        return InvalidSignatureLength as u32;
    }

    let hash = Message::from_digest(mem.read_fixed(hash_ptr));

    let sig_bytes = mem.read_fixed::<SIGNATURE_LENGTH>(sig_ptr);
    let Ok(recovery_id) = RecoveryId::try_from(sig_bytes[SIGNATURE_RECOVERY_ID_INDEX] as i32)
    else {
        return InvalidRecoveryId as u32;
    };
    let Ok(sig) =
        RecoverableSignature::from_compact(&sig_bytes[..SIGNATURE_LENGTH - 1], recovery_id)
    else {
        return RecoveryFailed as u32;
    };

    let Ok(pubkey) = secp256k1::Secp256k1::new().recover_ecdsa(hash, &sig) else {
        return RecoveryFailed as u32;
    };

    mem.write_slice(pub_ptr, pubkey.serialize_uncompressed().as_ref());
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
