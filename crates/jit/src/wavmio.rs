// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    io,
    io::{BufReader, BufWriter, ErrorKind},
    net::TcpStream,
    time::Instant,
};

use arbutil::Color;
use caller_env::GuestPtr;
use validation::transfer::receive_validation_input;

use crate::{
    caller_env::JitEnv,
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
};

/// Reads 32-bytes of global state.
pub fn get_global_state_bytes32(mut env: WasmEnvMut, idx: u32, out_ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, exec) = env.jit_env();
    ready_hostio(exec)?;
    caller_env::wavmio::get_global_state_bytes32(&mut mem, exec, idx, out_ptr)
        .map_err(Escape::HostIO)
}

/// Writes 32-bytes of global state.
pub fn set_global_state_bytes32(mut env: WasmEnvMut, idx: u32, src_ptr: GuestPtr) -> MaybeEscape {
    let (mem, exec) = env.jit_env();
    ready_hostio(exec)?;
    caller_env::wavmio::set_global_state_bytes32(&mem, exec, idx, src_ptr).map_err(Escape::HostIO)
}

/// Reads 8-bytes of global state
pub fn get_global_state_u64(mut env: WasmEnvMut, idx: u32) -> Result<u64, Escape> {
    let (_, exec) = env.jit_env();
    ready_hostio(exec)?;
    caller_env::wavmio::get_global_state_u64(exec, idx).map_err(Escape::HostIO)
}

/// Writes 8-bytes of global state
pub fn set_global_state_u64(mut env: WasmEnvMut, idx: u32, val: u64) -> MaybeEscape {
    let (_, exec) = env.jit_env();
    ready_hostio(exec)?;
    caller_env::wavmio::set_global_state_u64(exec, idx, val).map_err(Escape::HostIO)
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
    caller_env::wavmio::read_inbox_message(&mut mem, exec, msg_num, offset, out_ptr)
        .map_err(Escape::HostIO)
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
    caller_env::wavmio::read_delayed_inbox_message(&mut mem, exec, msg_num, offset, out_ptr)
        .map_err(Escape::HostIO)
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
    ready_hostio(exec)?;

    if TryInto::<arbutil::PreimageType>::try_into(preimage_type).is_err() {
        eprintln!("Go trying to resolve pre image with unknown type {preimage_type}");
        return Ok(0);
    }

    #[cfg(debug_assertions)]
    {
        use arbutil::PreimageType;
        use caller_env::MemAccess;
        use sha2::Sha256;
        use sha3::{Digest, Keccak256};

        let hash: [u8; 32] = mem.read_fixed(hash_ptr);
        let pt: PreimageType = preimage_type.try_into().unwrap();
        if let Some(preimage) = exec
            .input
            .preimages
            .get(&preimage_type)
            .and_then(|m| m.get(&hash))
        {
            let calculated_hash: [u8; 32] = match pt {
                PreimageType::Keccak256 => Keccak256::digest(preimage).into(),
                PreimageType::Sha2_256 => Sha256::digest(preimage).into(),
                PreimageType::EthVersionedHash => hash,
                PreimageType::DACertificate => hash,
            };
            if calculated_hash != hash {
                return Escape::hostio(format!(
                    "Calculated hash {} of preimage {} does not match provided hash {}",
                    hex::encode(calculated_hash),
                    hex::encode(preimage),
                    hex::encode(hash)
                ));
            }
        }
    }

    caller_env::wavmio::resolve_preimage(
        &mut mem,
        exec,
        preimage_type,
        hash_ptr,
        offset,
        out_ptr,
        name,
    )
    .map_err(Escape::HostIO)
}

pub fn validate_certificate(
    mut env: WasmEnvMut,
    preimage_type: u8,
    hash_ptr: GuestPtr,
) -> Result<u8, Escape> {
    let (mem, exec) = env.jit_env();
    Ok(caller_env::wavmio::validate_certificate(
        &mem,
        exec,
        preimage_type,
        hash_ptr,
    ))
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
    crate::machine::load_validation_input(env, input);

    let writer = BufWriter::new(socket);
    env.process.socket = Some((writer, reader));
    env.process.already_has_input = true;
    Ok(())
}
