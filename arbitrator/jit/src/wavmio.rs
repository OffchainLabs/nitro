// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    caller_env::JitEnv,
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
    socket,
};
use arbutil::{Color, PreimageType};
use caller_env::{GuestPtr, MemAccess};
use sha2::Sha256;
use sha3::{Digest, Keccak256};
use std::{
    io,
    io::{BufReader, BufWriter, ErrorKind},
    net::TcpStream,
    time::Instant,
};

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

pub fn read_hotshot_commitment(
    mut env: WasmEnvMut,
    h: u64,
    out_ptr: GuestPtr,
) -> Result<(), Escape> {
    let (mut mem, exec) = env.jit_env();
    ready_hostio(exec)?;

    let comm = match exec.hotshot_comm_map.get(&h) {
        Some(comm) => comm,
        None => {
            return Escape::hostio(format!(
                "jit machine failed to read the hotshot commitment at {}",
                h
            ))
        }
    };
    mem.write_slice(out_ptr, comm);
    Ok(())
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

    // Check if preimage rehashes to the provided hash. Exclude blob preimages
    let calculated_hash: [u8; 32] = match preimage_type {
        PreimageType::Keccak256 => Keccak256::digest(preimage).into(),
        PreimageType::Sha2_256 => Sha256::digest(preimage).into(),
        PreimageType::EthVersionedHash => *hash,
    };
    if calculated_hash != *hash {
        error!(
            "Calculated hash {} of preimage {} does not match provided hash {}",
            hex::encode(calculated_hash),
            hex::encode(preimage),
            hex::encode(*hash)
        );
    }

    if offset % 32 != 0 {
        error!("bad offset {offset} in {name}")
    };

    let len = std::cmp::min(32, preimage.len().saturating_sub(offset));
    let read = preimage.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
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

    if !env.process.forks {
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
    let stream = &mut reader;

    let inbox_position = socket::read_u64(stream)?;
    let position_within_message = socket::read_u64(stream)?;
    let last_block_hash = socket::read_bytes32(stream)?;
    let last_send_root = socket::read_bytes32(stream)?;
    let last_hotshot_height = socket::read_u64(stream)?;
    let validating_hotshot_height = socket::read_u64(stream)?;
    let hotshot_comm = socket::read_bytes32(stream)?;

    env.small_globals = [inbox_position, position_within_message, last_hotshot_height];
    env.large_globals = [last_block_hash, last_send_root];
    env.hotshot_comm_map
        .insert(validating_hotshot_height, hotshot_comm.0);

    while socket::read_u8(stream)? == socket::ANOTHER {
        let position = socket::read_u64(stream)?;
        let message = socket::read_bytes(stream)?;
        env.sequencer_messages.insert(position, message);
    }
    while socket::read_u8(stream)? == socket::ANOTHER {
        let position = socket::read_u64(stream)?;
        let message = socket::read_bytes(stream)?;
        env.delayed_messages.insert(position, message);
    }

    let preimage_types = socket::read_u32(stream)?;
    for _ in 0..preimage_types {
        let preimage_ty = PreimageType::try_from(socket::read_u8(stream)?)
            .map_err(|e| Escape::Failure(e.to_string()))?;
        let map = env.preimages.entry(preimage_ty).or_default();
        let preimage_count = socket::read_u32(stream)?;
        for _ in 0..preimage_count {
            let hash = socket::read_bytes32(stream)?;
            let preimage = socket::read_bytes(stream)?;
            map.insert(hash, preimage);
        }
    }

    let programs_count = socket::read_u32(stream)?;
    for _ in 0..programs_count {
        let module_hash = socket::read_bytes32(stream)?;
        let module_asm = socket::read_boxed_slice(stream)?;
        env.module_asms.insert(module_hash, module_asm.into());
    }

    if socket::read_u8(stream)? != socket::READY {
        return Escape::hostio("failed to parse global state");
    }

    let writer = BufWriter::new(socket);
    env.process.socket = Some((writer, reader));
    env.process.forks = false;
    Ok(())
}
