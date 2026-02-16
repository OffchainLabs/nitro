// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{
    caller_env::JitEnv,
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
};
use arbutil::Color;
use caller_env::{GuestPtr, MemAccess};
use std::{
    io,
    io::{BufReader, BufWriter, ErrorKind},
    net::TcpStream,
    time::Instant,
};
use validation::local_target;
use validation::transfer::receive_validation_input;

/// Reads 32-bytes of global state.
pub fn get_global_state_bytes32(mut env: WasmEnvMut, idx: u32, out_ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, exec) = env.jit_env();
    ready_hostio(exec)?;

    let Some(global) = exec.large_globals.get(idx as usize) else {
        return Escape::hostio("global read out of bounds in wavmio.getGlobalStateBytes32");
    };
    mem.write_slice(out_ptr, &global[..32]);
    Ok(())
}

/// Writes 32-bytes of global state.
pub fn set_global_state_bytes32(mut env: WasmEnvMut, idx: u32, src_ptr: GuestPtr) -> MaybeEscape {
    let (mem, exec) = env.jit_env();
    ready_hostio(exec)?;

    let slice = mem.read_slice(src_ptr, 32);
    let slice = &slice.try_into().unwrap();
    match exec.large_globals.get_mut(idx as usize) {
        Some(global) => *global = *slice,
        None => return Escape::hostio("global write oob in wavmio.setGlobalStateBytes32"),
    };
    Ok(())
}

/// Reads 8-bytes of global state
pub fn get_global_state_u64(mut env: WasmEnvMut, idx: u32) -> Result<u64, Escape> {
    let (_, exec) = env.jit_env();
    ready_hostio(exec)?;

    match exec.small_globals.get(idx as usize) {
        Some(global) => Ok(*global),
        None => Escape::hostio("global read out of bounds in wavmio.getGlobalStateU64"),
    }
}

/// Writes 8-bytes of global state
pub fn set_global_state_u64(mut env: WasmEnvMut, idx: u32, val: u64) -> MaybeEscape {
    let (_, exec) = env.jit_env();
    ready_hostio(exec)?;

    match exec.small_globals.get_mut(idx as usize) {
        Some(global) => *global = val,
        None => return Escape::hostio("global write out of bounds in wavmio.setGlobalStateU64"),
    }
    Ok(())
}

/// Reads an inbox message.
pub fn read_inbox_message(
    mut env: WasmEnvMut,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, exec) = env.jit_env();
    ready_hostio(exec)?;

    let message = match exec.sequencer_messages.get(&msg_num) {
        Some(message) => message,
        None => return Escape::hostio(format!("missing sequencer inbox message {msg_num}")),
    };
    let offset = offset as usize;
    let len = std::cmp::min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Reads a delayed inbox message.
pub fn read_delayed_inbox_message(
    mut env: WasmEnvMut,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, exec) = env.jit_env();
    ready_hostio(exec)?;

    let message = match exec.delayed_messages.get(&msg_num) {
        Some(message) => message,
        None => return Escape::hostio(format!("missing delayed inbox message {msg_num}")),
    };
    let offset = offset as usize;
    let len = std::cmp::min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Retrieves the preimage of the given hash.
#[deprecated] // we're just keeping this around until we no longer need to validate old replay binaries
pub fn resolve_keccak_preimage(
    env: WasmEnvMut,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    resolve_preimage_impl(env, 0, hash_ptr, offset, out_ptr, "wavmio.ResolvePreImage")
}

pub fn resolve_typed_preimage(
    env: WasmEnvMut,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    resolve_preimage_impl(
        env,
        preimage_type,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolveTypedPreimage",
    )
}

pub fn resolve_preimage_impl(
    mut env: WasmEnvMut,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
    name: &str,
) -> Result<u32, Escape> {
    let (mut mem, exec) = env.jit_env();
    let offset = offset as usize;

    let Ok(preimage_type) = preimage_type.try_into() else {
        eprintln!("Go trying to resolve pre image with unknown type {preimage_type}");
        return Ok(0);
    };

    macro_rules! error {
        ($text:expr $(,$args:expr)*) => {{
            let text = format!($text $(,$args)*);
            return Escape::hostio(&text)
        }};
    }

    let hash = mem.read_bytes32(hash_ptr);

    let Some(preimage) = exec
        .preimages
        .get(&preimage_type)
        .and_then(|m| m.get(&hash))
    else {
        let hash_hex = hex::encode(hash);
        error!("Missing requested preimage for hash {hash_hex} in {name}")
    };

    #[cfg(debug_assertions)]
    {
        use sha2::Sha256;
        use sha3::{Digest, Keccak256};
        use arbutil::PreimageType;

        // Check if preimage rehashes to the provided hash. Exclude blob preimages
        let calculated_hash: [u8; 32] = match preimage_type {
            PreimageType::Keccak256 => Keccak256::digest(preimage).into(),
            PreimageType::Sha2_256 => Sha256::digest(preimage).into(),
            PreimageType::EthVersionedHash => *hash,
            PreimageType::DACertificate => *hash, // Can't verify DACertificate hash, just accept it
        };
        if calculated_hash != *hash {
            error!(
                "Calculated hash {} of preimage {} does not match provided hash {}",
                hex::encode(calculated_hash),
                hex::encode(preimage),
                hex::encode(*hash)
            );
        }
    }

    if offset % 32 != 0 {
        error!("bad offset {offset} in {name}")
    };

    let len = std::cmp::min(32, preimage.len().saturating_sub(offset));
    let read = preimage.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

pub fn validate_certificate(
    mut env: WasmEnvMut,
    preimage_type: u8,
    hash_ptr: GuestPtr,
) -> Result<u8, Escape> {
    let (mut mem, exec) = env.jit_env();
    let hash = mem.read_bytes32(hash_ptr);

    let Ok(preimage_type) = preimage_type.try_into() else {
        eprintln!(
            "Go trying to validate certificate for preimage with unknown type {preimage_type}"
        );
        return Ok(0);
    };

    // Check if preimage exists
    let exists = exec
        .preimages
        .get(&preimage_type)
        .and_then(|m| m.get(&hash))
        .is_some();

    Ok(if exists { 1 } else { 0 })
}

fn ready_hostio(env: &mut WasmEnv) -> MaybeEscape {
    let debug = env.process.debug;

    if !env.process.reached_wavmio {
        if debug {
            let time = format!("{}ms", env.process.timestamp.elapsed().as_millis());
            println!("Created the machine in {}.", time.pink());
        }
        env.process.timestamp = Instant::now();
        env.process.reached_wavmio = true;
    }

    if env.process.already_has_input {
        return Ok(());
    }

    unsafe {
        libc::signal(libc::SIGCHLD, libc::SIG_IGN); // avoid making zombies
    }

    let stdin = io::stdin();
    let mut address = String::new();

    loop {
        if let Err(error) = stdin.read_line(&mut address) {
            return match error.kind() {
                ErrorKind::UnexpectedEof => Escape::exit(0),
                error => Escape::hostio(format!("Error reading stdin: {error}")),
            };
        }

        address.pop(); // pop the newline
        if address.is_empty() {
            return Escape::exit(0);
        }
        if debug {
            println!("Child will connect to {address}");
        }

        unsafe {
            match libc::fork() {
                -1 => return Escape::hostio("Failed to fork"),
                0 => break,                   // we're the child process
                _ => address = String::new(), // we're the parent process
            }
        }
    }

    env.process.timestamp = Instant::now();
    if debug {
        println!("Connecting to {address}");
    }
    let socket = TcpStream::connect(&address)?;
    socket.set_nodelay(true)?;

    let mut reader = BufReader::new(socket.try_clone()?);
    let input = receive_validation_input(&mut reader)?;

    env.small_globals = [input.start_state.batch, input.start_state.pos_in_batch];
    env.large_globals = [input.start_state.block_hash, input.start_state.send_root];

    for batch in input.batch_info {
        env.sequencer_messages.insert(batch.number, batch.data);
    }
    if input.has_delayed_msg {
        env.delayed_messages
            .insert(input.delayed_msg_nr, input.delayed_msg);
    }
    for (preimage_type, preimages) in input.preimages {
        let preimage_map = env.preimages.entry(preimage_type).or_default();
        for (hash, preimage) in preimages {
            preimage_map.insert(hash, preimage);
        }
    }
    for (module_hash, module_asm) in &input.user_wasms[local_target()] {
        env.module_asms
            .insert(*module_hash, module_asm.as_vec().into());
    }

    let writer = BufWriter::new(socket);
    env.process.socket = Some((writer, reader));
    env.process.already_has_input = true;
    Ok(())
}
