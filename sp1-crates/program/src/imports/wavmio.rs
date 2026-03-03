//! wavmio functions — thin wrappers delegating to caller_env::wavmio.

use crate::{
    Escape, MaybeEscape, Ptr, caller_env_adapters::Sp1Env, read_bytes32, replay::CustomEnvData,
};
use ::caller_env::wavmio as caller_env;
use ::caller_env::GuestPtr;
use core::ops::Deref;
use wasmer::{FunctionEnvMut, MemoryView};

fn gp(p: Ptr) -> GuestPtr {
    GuestPtr(p.offset())
}

pub fn get_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    out_ptr: Ptr,
) -> MaybeEscape {
    let (mut mem, state) = ctx.sp1_env();
    caller_env::get_global_state_bytes32(&mut mem, &state, idx, gp(out_ptr)).map_err(Escape::Logical)
}

pub fn set_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    src_ptr: Ptr,
) -> MaybeEscape {
    let (mem, mut state) = ctx.sp1_env();
    caller_env::set_global_state_bytes32(&mem, &mut state, idx, gp(src_ptr)).map_err(Escape::Logical)
}

pub fn get_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
) -> Result<u64, Escape> {
    let (_mem, state) = ctx.sp1_env();
    caller_env::get_global_state_u64(&state, idx).map_err(Escape::Logical)
}

pub fn set_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    val: u64,
) -> MaybeEscape {
    let (_mem, mut state) = ctx.sp1_env();
    caller_env::set_global_state_u64(&mut state, idx, val).map_err(Escape::Logical)
}

pub fn read_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    let (mut mem, state) = ctx.sp1_env();
    caller_env::read_inbox_message(&mut mem, &state, msg_num, offset, gp(out_ptr))
        .map_err(Escape::Logical)
}

pub fn read_delayed_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    let (mut mem, state) = ctx.sp1_env();
    caller_env::read_delayed_inbox_message(&mut mem, &state, msg_num, offset, gp(out_ptr))
        .map_err(Escape::Logical)
}

pub fn resolve_keccak_preimage(
    ctx: FunctionEnvMut<CustomEnvData>,
    hash_ptr: Ptr,
    offset: u32,
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    resolve_preimage_impl(ctx, 0, hash_ptr, offset, out_ptr, "wavmio.ResolvePreImage")
}

pub fn resolve_typed_preimage(
    ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: Ptr,
    offset: u32,
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    resolve_preimage_impl(
        ctx,
        preimage_type,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolveTypedPreimage",
    )
}

pub fn validate_certificate(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: Ptr,
) -> Result<u8, Escape> {
    let (mem, state) = ctx.sp1_env();
    Ok(caller_env::validate_certificate(
        &mem,
        &state,
        preimage_type,
        gp(hash_ptr),
    ))
}

fn resolve_preimage_impl(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: Ptr,
    offset: u32,
    out_ptr: Ptr,
    name: &str,
) -> Result<u32, Escape> {
    let (mut mem, state) = ctx.sp1_env();
    caller_env::resolve_preimage(
        &mut mem,
        &state,
        preimage_type,
        gp(hash_ptr),
        offset,
        gp(out_ptr),
        name,
    )
    .map_err(Escape::Logical)
}

// Greedy preimage resolution — kept separate, will be refactored independently.

pub fn greedy_resolve_typed_preimage(
    ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: Ptr,
    offset: u32,
    available: u32,
    out_ptr: Ptr,
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
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    let full_len = data.len().saturating_sub(offset) as u32;
    let len = std::cmp::min(available, full_len);
    let read = data
        .get(offset..(offset + len as usize))
        .unwrap_or_default();
    memory.write(out_ptr.offset() as u64, read)?;
    Ok(full_len)
}

fn greedy_resolve_typed_preimage_impl(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: Ptr,
    offset: u32,
    available: u32,
    out_ptr: Ptr,
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
    greedy_read(&preimage, &memory, offset, available, out_ptr)
}
