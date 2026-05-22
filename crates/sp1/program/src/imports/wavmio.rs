//! wavmio functions — thin wrappers delegating to caller_env::wavmio.

use core::ops::Deref;

use ::caller_env::{GuestPtr, wavmio as caller_env};
use wasmer::{FunctionEnvMut, MemoryView};

use crate::{
    Escape, MaybeEscape, read_bytes32,
    replay::CustomEnvData,
    state::sp1_env,
};

pub fn get_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    out_ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, state) = sp1_env(&mut ctx);
    caller_env::get_global_state_bytes32(&mut mem, state, idx, out_ptr).map_err(Escape::Logical)
}

pub fn set_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    src_ptr: GuestPtr,
) -> MaybeEscape {
    let (mem, state) = sp1_env(&mut ctx);
    caller_env::set_global_state_bytes32(&mem, state, idx, src_ptr).map_err(Escape::Logical)
}

pub fn get_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
) -> Result<u64, Escape> {
    let (_mem, state) = sp1_env(&mut ctx);
    caller_env::get_global_state_u64(state, idx).map_err(Escape::Logical)
}

pub fn set_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    val: u64,
) -> MaybeEscape {
    let (_mem, state) = sp1_env(&mut ctx);
    caller_env::set_global_state_u64(state, idx, val).map_err(Escape::Logical)
}

pub fn read_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    caller_env::read_inbox_message(&mut mem, state, msg_num, offset, out_ptr)
        .map_err(Escape::Logical)
}

pub fn read_delayed_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    caller_env::read_delayed_inbox_message(&mut mem, state, msg_num, offset, out_ptr)
        .map_err(Escape::Logical)
}

pub fn resolve_keccak_preimage(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    caller_env::resolve_preimage(
        &mut mem,
        state,
        0,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolvePreImage",
    )
    .map_err(Escape::Logical)
}

pub fn resolve_typed_preimage(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    caller_env::resolve_preimage(
        &mut mem,
        state,
        preimage_type,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolveTypedPreimage",
    )
    .map_err(Escape::Logical)
}

pub fn validate_certificate(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: GuestPtr,
) -> Result<u8, Escape> {
    let (mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::validate_certificate(
        &mem,
        state,
        preimage_type,
        hash_ptr,
    ))
}

// Greedy preimage resolution — kept separate, will be refactored independently.

pub fn greedy_resolve_typed_preimage(
    ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    available: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    greedy_resolve_typed_preimage_impl(
        ctx,
        preimage_type,
        hash_ptr,
        offset,
        available,
        out_ptr,
        "wavmio.ResolveTypedPreimage2",
    )
}

fn greedy_read(
    data: &[u8],
    memory: &MemoryView,
    offset: usize,
    available: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let full_len = data.len().saturating_sub(offset) as u32;
    let len = std::cmp::min(available, full_len);
    let read = data
        .get(offset..(offset + len as usize))
        .unwrap_or_default();
    memory.write(out_ptr.into(), read)?;
    Ok(full_len)
}

fn greedy_resolve_typed_preimage_impl(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    available: u32,
    out_ptr: GuestPtr,
    name: &str,
) -> Result<u32, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);
    let offset = offset as usize;
    let hash = read_bytes32(hash_ptr, &memory)?;
    let Some(preimage) = data
        .input()
        .preimages
        .get(&preimage_type)
        .and_then(|m| m.get(hash.deref()))
    else {
        return Escape::logical(format!(
            "Missing requested preimage for hash {} in {name}",
            hex::encode(hash)
        ));
    };
    greedy_read(preimage, &memory, offset, available, out_ptr)
}
