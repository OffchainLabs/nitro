// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    goenv::GoEnv,
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
    socket,
};

use arbutil::Color;
use std::{
    io,
    io::{BufReader, BufWriter, ErrorKind},
    net::TcpStream,
    time::Instant,
};

type Uptr = u32;

/// Reads 32-bytes of global state
pub fn get_global_state_bytes32(mut env: WasmEnvMut, idx: u32, out_ptr: Uptr) -> MaybeEscape {
    let genv = GoEnv::new(&mut env);
    ready_hostio(genv.wenv)?;

    let global = match genv.wenv.large_globals.get(idx as usize) {
        Some(global) => global,
        None => return Escape::hostio("global read out of bounds in wavmio.getGlobalStateBytes32"),
    };
    genv.caller_write_slice(out_ptr, &global[..32]);
    Ok(())
}

/// Writes 32-bytes of global state
pub fn set_global_state_bytes32(mut env: WasmEnvMut, idx: u32, src_ptr: Uptr) -> MaybeEscape {
    let genv = GoEnv::new(&mut env);
    ready_hostio(genv.wenv)?;

    let slice = genv.caller_read_slice(src_ptr, 32);
    let slice = &slice.try_into().unwrap();
    match genv.wenv.large_globals.get_mut(idx as usize) {
        Some(global) => *global = *slice,
        None => {
            return Escape::hostio("global write out of bounds in wavmio.setGlobalStateBytes32")
        }
    }
    Ok(())
}

/// Reads 8-bytes of global state
pub fn get_global_state_u64(mut env: WasmEnvMut, idx: u32) -> Result<u64, Escape> {
    let genv = GoEnv::new(&mut env);
    ready_hostio(genv.wenv)?;

    match genv.wenv.small_globals.get(idx as usize) {
        Some(global) => Ok(*global),
        None => Escape::hostio("global read out of bounds in wavmio.getGlobalStateU64"),
    }
}

/// Writes 8-bytes of global state
pub fn set_global_state_u64(mut env: WasmEnvMut, idx: u32, val: u64) -> MaybeEscape {
    let genv = GoEnv::new(&mut env);
    ready_hostio(genv.wenv)?;

    match genv.wenv.small_globals.get_mut(idx as usize) {
        Some(global) => {
            *global = val;
            Ok(())
        }
        None => Escape::hostio("global write out of bounds in wavmio.setGlobalStateU64"),
    }
}

/// Reads an inbox message
pub fn read_inbox_message(mut env: WasmEnvMut, msg_num: u64, offset: u32, out_ptr: Uptr) -> Result<u32, Escape> {
    let genv = GoEnv::new(&mut env);
    ready_hostio(genv.wenv)?;

    let message = match genv.wenv.sequencer_messages.get(&msg_num) {
        Some(message) => message,
        None => return Escape::hostio(format!("missing sequencer inbox message {msg_num}")),
    };
    let offset = offset as usize;
    let len = std::cmp::min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    genv.caller_write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Reads a delayed inbox message
pub fn read_delayed_inbox_message(mut env: WasmEnvMut, msg_num: u64, offset: u32, out_ptr: Uptr) -> Result<u32, Escape> {
    let genv = GoEnv::new(&mut env);
    ready_hostio(genv.wenv)?;

    let message = match genv.wenv.delayed_messages.get(&msg_num) {
        Some(message) => message,
        None => return Escape::hostio(format!("missing delayed inbox message {msg_num}")),
    };
    let offset = offset as usize;
    let len = std::cmp::min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    genv.caller_write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Retrieves the preimage of the given hash.
pub fn resolve_preimage(mut env: WasmEnvMut, hash_ptr: Uptr, offset: u32, out_ptr: Uptr) -> Result<u32, Escape> {
    let mut genv = GoEnv::new(&mut env);

    let name = "wavmio.resolvePreImage";

    macro_rules! error {
        ($text:expr $(,$args:expr)*) => {{
            let text = format!($text $(,$args)*);
            return Escape::hostio(&text)
        }};
    }

    let hash = genv.caller_read_bytes32(hash_ptr);
    let hash_hex = hex::encode(hash);

    let mut preimage = None;

    // see if we've cached the preimage
    if let Some((key, cached)) = &genv.wenv.process.last_preimage {
        if *key == hash {
            preimage = Some(cached);
        }
    }

    // see if this is a known preimage
    if preimage.is_none() {
        preimage = genv.wenv.preimages.get(&hash);
    }

    let preimage = match preimage {
        Some(preimage) => preimage,
        None => error!("Missing requested preimage for hash {hash_hex} in {name}"),
    };
    let offset = match u32::try_from(offset) {
        Ok(offset) => offset as usize,
        Err(_) => error!("bad offset {offset} in {name}"),
    };

    let len = std::cmp::min(32, preimage.len().saturating_sub(offset));
    let read = preimage.get(offset..(offset + len)).unwrap_or_default();
    genv.caller_write_slice(out_ptr, read);
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

    env.small_globals = [inbox_position, position_within_message];
    env.large_globals = [last_block_hash, last_send_root];

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

    let preimage_count = socket::read_u32(stream)?;
    for _ in 0..preimage_count {
        let hash = socket::read_bytes32(stream)?;
        let preimage = socket::read_bytes(stream)?;
        env.preimages.insert(hash, preimage);
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
