//! wavmio functions — thin wrappers delegating to caller_env::wavmio.

use ::caller_env::{GuestPtr, wavmio as caller_env};
use wasmer::FunctionEnvMut;

use crate::{Escape, MaybeEscape, replay::CustomEnvData, state::sp1_env};

pub fn get_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    out_ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::get_global_state_bytes32(&mut mem, state, idx, out_ptr)?)
}

pub fn set_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    src_ptr: GuestPtr,
) -> MaybeEscape {
    let (mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::set_global_state_bytes32(&mem, state, idx, src_ptr)?)
}

pub fn get_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
) -> Result<u64, Escape> {
    let (_mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::get_global_state_u64(state, idx)?)
}

pub fn set_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    val: u64,
) -> MaybeEscape {
    let (_mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::set_global_state_u64(state, idx, val)?)
}

pub fn read_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::read_inbox_message(&mut mem, state, msg_num, offset, out_ptr)?)
}

pub fn read_delayed_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::read_delayed_inbox_message(&mut mem, state, msg_num, offset, out_ptr)?)
}

pub fn resolve_keccak_preimage(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::resolve_preimage(
        &mut mem,
        state,
        0,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolvePreImage",
    )?)
}

pub fn resolve_typed_preimage(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::resolve_preimage(
        &mut mem,
        state,
        preimage_type,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolveTypedPreimage",
    )?)
}

pub fn validate_certificate(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: GuestPtr,
) -> Result<u8, Escape> {
    let (mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::validate_certificate(&mem, state, preimage_type, hash_ptr))
}
