// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::caller_env::JitWavm;
use crate::machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut};
use arbutil::Color;
use ::caller_env::wavmio as caller_env;
use ::caller_env::GuestPtr;
use std::{
    io,
    io::{BufReader, BufWriter, ErrorKind},
    net::TcpStream,
    time::Instant,
};
use validation::local_target;
use validation::transfer::receive_validation_input;

pub fn get_global_state_bytes32(mut env: WasmEnvMut, idx: u32, out_ptr: GuestPtr) -> MaybeEscape {
    ready_hostio(env.data_mut())?;
    caller_env::get_global_state_bytes32(&mut JitWavm(env), idx, out_ptr).map_err(Escape::HostIO)
}

pub fn set_global_state_bytes32(mut env: WasmEnvMut, idx: u32, src_ptr: GuestPtr) -> MaybeEscape {
    ready_hostio(env.data_mut())?;
    caller_env::set_global_state_bytes32(&mut JitWavm(env), idx, src_ptr).map_err(Escape::HostIO)
}

pub fn get_global_state_u64(mut env: WasmEnvMut, idx: u32) -> Result<u64, Escape> {
    ready_hostio(env.data_mut())?;
    caller_env::get_global_state_u64(&mut JitWavm(env), idx).map_err(Escape::HostIO)
}

pub fn set_global_state_u64(mut env: WasmEnvMut, idx: u32, val: u64) -> MaybeEscape {
    ready_hostio(env.data_mut())?;
    caller_env::set_global_state_u64(&mut JitWavm(env), idx, val).map_err(Escape::HostIO)
}

pub fn read_inbox_message(
    mut env: WasmEnvMut,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    ready_hostio(env.data_mut())?;
    caller_env::read_inbox_message(&mut JitWavm(env), msg_num, offset, out_ptr)
        .map_err(Escape::HostIO)
}

pub fn read_delayed_inbox_message(
    mut env: WasmEnvMut,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, Escape> {
    ready_hostio(env.data_mut())?;
    caller_env::read_delayed_inbox_message(&mut JitWavm(env), msg_num, offset, out_ptr)
        .map_err(Escape::HostIO)
}

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

fn resolve_preimage_impl(
    mut env: WasmEnvMut,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
    name: &str,
) -> Result<u32, Escape> {
    ready_hostio(env.data_mut())?;

    #[cfg(debug_assertions)]
    {
        use crate::caller_env::jit_env;
        use arbutil::PreimageType;
        use sha2::Sha256;
        use sha3::{Digest, Keccak256};

        let (mut mem, state) = jit_env(&mut env);
        let hash = mem.read_bytes32(hash_ptr);

        let Ok(preimage_type) = preimage_type.try_into() else {
            eprintln!("Go trying to resolve pre image with unknown type {preimage_type}");
            return Ok(0);
        };

        if let Some(preimage) = state.0
            .preimages
            .get(&preimage_type)
            .and_then(|m| m.get(&hash))
        {
            let calculated_hash: [u8; 32] = match preimage_type {
                PreimageType::Keccak256 => Keccak256::digest(preimage).into(),
                PreimageType::Sha2_256 => Sha256::digest(preimage).into(),
                PreimageType::EthVersionedHash => *hash,
                PreimageType::DACertificate => *hash,
            };
            if calculated_hash != *hash {
                return Escape::hostio(format!(
                    "Calculated hash {} of preimage {} does not match provided hash {}",
                    hex::encode(calculated_hash),
                    hex::encode(preimage),
                    hex::encode(*hash)
                ));
            }
        }
    }

    caller_env::resolve_preimage(&mut JitWavm(env), preimage_type, hash_ptr, offset, out_ptr, name)
        .map_err(Escape::HostIO)
}

pub fn validate_certificate(
    env: WasmEnvMut,
    preimage_type: u8,
    hash_ptr: GuestPtr,
) -> Result<u8, Escape> {
    Ok(caller_env::validate_certificate(&mut JitWavm(env), preimage_type, hash_ptr))
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
