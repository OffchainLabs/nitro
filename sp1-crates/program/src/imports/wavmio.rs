//! wavmio functions
//!
//! The code here is heavily borrowed from nitro's own implementation:
//! https://github.com/OffchainLabs/nitro/blob/3710544c6b36a8927a8dab26d928ad553f08175d/arbitrator/jit/src/wavmio.rs

use crate::{Escape, MaybeEscape, Ptr, read_bytes32, replay::CustomEnvData};
use std::ops::Deref;
use wasmer::{FunctionEnvMut, MemoryView};

pub fn get_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    out_ptr: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let Some(global) = data.input().large_globals.get(idx as usize) else {
        return Escape::logical("global read out of bounds in wavmio.getGlobalStateBytes32");
    };

    memory.write(out_ptr.offset() as u64, &global[..32])?;

    Ok(())
}

pub fn set_global_state_bytes32(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    src_ptr: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let slice = read_bytes32(src_ptr, &memory)?;
    match data.input_mut().large_globals.get_mut(idx as usize) {
        Some(global) => *global = *slice,
        None => return Escape::logical("global write oob in wavmio.setGlobalStateBytes32"),
    }
    Ok(())
}

pub fn get_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
) -> Result<u64, Escape> {
    let (data, _store) = ctx.data_and_store_mut();

    Ok(match data.input().small_globals.get(idx as usize) {
        Some(global) => *global,
        None => return Escape::logical("global read out of bounds in wavmio.getGlobalStateU64"),
    })
}

pub fn set_global_state_u64(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    idx: u32,
    val: u64,
) -> MaybeEscape {
    let (data, _store) = ctx.data_and_store_mut();

    match data.input_mut().small_globals.get_mut(idx as usize) {
        Some(global) => *global = val,
        None => return Escape::logical("global write out of bounds in wavmio.setGlobalStateU64"),
    }
    Ok(())
}

pub fn read_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let message = match data.input().sequencer_messages.get(&msg_num) {
        Some(message) => message,
        None => return Escape::logical("missing sequencer inbox message {msg_num}"),
    };
    let offset = offset as usize;
    let len = std::cmp::min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    memory.write(out_ptr.offset() as u64, read)?;

    Ok(read.len() as u32)
}

pub fn read_delayed_inbox_message(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    msg_num: u64,
    offset: u32,
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let message = match data.input().delayed_messages.get(&msg_num) {
        Some(message) => message,
        None => return Escape::logical("missing delayed inbox message {msg_num}"),
    };
    let offset = offset as usize;
    let len = std::cmp::min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    memory.write(out_ptr.offset() as u64, read)?;

    Ok(read.len() as u32)
}

pub fn resolve_keccak_preimage(
    ctx: FunctionEnvMut<CustomEnvData>,
    hash_ptr: Ptr,
    offset: u32,
    out_ptr: Ptr,
) -> Result<u32, Escape> {
    resolve_preimage_impl(ctx, 0, hash_ptr, offset, out_ptr, "wavmio.ResolvePreImage")
}

pub fn validate_certificate(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: Ptr,
) -> Result<u8, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let hash = read_bytes32(hash_ptr, &memory)?;

    // Check if preimage exists
    let exists = data
        .input()
        .preimages
        .get(&preimage_type)
        .and_then(|m| m.get(hash.deref()))
        .is_some();

    Ok(if exists { 1 } else { 0 })
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

fn resolve_preimage_impl(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    preimage_type: u8,
    hash_ptr: Ptr,
    offset: u32,
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
        let hash_hex = hex::encode(hash);
        return Escape::logical(format!(
            "Missing requested preimage for hash {hash_hex} in {name}"
        ));
    };

    #[cfg(debug_assertions)]
    {
        use crate::input_types::PreimageType;
        use sha2::Sha256;
        use sha3::{Digest, Keccak256};

        // Check if preimage rehashes to the provided hash. Exclude blob preimages
        let calculated_hash: [u8; 32] = match preimage_type {
            PreimageType::Keccak256 => Keccak256::digest(preimage).into(),
            PreimageType::Sha2_256 => Sha256::digest(preimage).into(),
            PreimageType::EthVersionedHash => *hash,
        };
        if calculated_hash != *hash {
            panic!(
                "Calculated hash {} of preimage {} does not match provided hash {}",
                hex::encode(calculated_hash),
                hex::encode(preimage),
                hex::encode(*hash)
            );
        }
    }

    if offset % 32 != 0 {
        return Escape::logical(format!("bad offset {offset} in {name}"));
    }

    let len = std::cmp::min(32, preimage.len().saturating_sub(offset));
    let read = preimage.get(offset..(offset + len)).unwrap_or_default();
    memory.write(out_ptr.offset() as u64, read)?;

    Ok(read.len() as u32)
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
        let hash_hex = hex::encode(hash);
        return Escape::logical(format!(
            "Missing requested preimage for hash {hash_hex} in {name}"
        ));
    };

    #[cfg(debug_assertions)]
    {
        use crate::input_types::PreimageType;
        use sha2::Sha256;
        use sha3::{Digest, Keccak256};

        // Check if preimage rehashes to the provided hash. Exclude blob preimages
        let calculated_hash: [u8; 32] = match preimage_type {
            PreimageType::Keccak256 => Keccak256::digest(preimage).into(),
            PreimageType::Sha2_256 => Sha256::digest(preimage).into(),
            PreimageType::EthVersionedHash => *hash,
        };
        if calculated_hash != *hash {
            panic!(
                "Calculated hash {} of preimage {} does not match provided hash {}",
                hex::encode(calculated_hash),
                hex::encode(preimage),
                hex::encode(*hash)
            );
        }
    }

    greedy_read(&preimage, &memory, offset, available, out_ptr)
}
