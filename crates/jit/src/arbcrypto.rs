// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use crate::caller_env::{jit_env, JitExecEnv};
use crate::machine::{Escape, MaybeEscape, WasmEnvMut};
use caller_env::GuestPtr;

pub fn ecrecovery(
    mut src: WasmEnvMut,
    hash_ptr: GuestPtr,
    hash_len: u32,
    sig_ptr: GuestPtr,
    sig_len: u32,
    pub_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = jit_env(&mut src);

    Ok(caller_env::arbcrypto::ecrecovery(
        &mut mem,
        &mut JitExecEnv { wenv: state.0 },
        hash_ptr,
        hash_len,
        sig_ptr,
        sig_len,
        pub_ptr,
    ))
}

pub fn keccak256(
    mut src: WasmEnvMut,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, state) = jit_env(&mut src);

    caller_env::arbcrypto::keccak256(
        &mut mem,
        &mut JitExecEnv { wenv: state.0 },
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
    );
    Ok(())
}
