use crate::caller_env::{JitEnv, JitExecEnv};
use crate::machine::{Escape, MaybeEscape, WasmEnvMut};
use caller_env::GuestPtr;

pub fn ecrecovery(
    mut src: WasmEnvMut,
    hash_ptr: GuestPtr,
    sig_ptr: GuestPtr,
    pub_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, wenv) = src.jit_env();

    Ok(caller_env::arbcrypto::ecrecovery(
        &mut mem,
        &mut JitExecEnv { wenv },
        hash_ptr,
        sig_ptr,
        pub_ptr,
    ))
}

pub fn keccak256(
    mut src: WasmEnvMut,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, wenv) = src.jit_env();

    caller_env::arbcrypto::keccak256(
        &mut mem,
        &mut JitExecEnv { wenv },
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
    );
    Ok(())
}
