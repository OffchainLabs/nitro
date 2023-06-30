// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/offchainlabs/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, Inbox, MaybeEscape, WasmEnv, WasmEnvMut},
    socket,
};

use arbutil::Color;
use std::{
    io,
    io::{BufReader, BufWriter, ErrorKind, Write},
    net::TcpStream,
    time::Instant,
};

pub type Bytes32 = [u8; 32];

pub fn get_global_state_bytes32(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, &mut env);
    ready_hostio(env)?;

    let global = sp.read_u64(0) as u32 as usize;
    let out_ptr = sp.read_u64(1);
    let mut out_len = sp.read_u64(2) as usize;
    if out_len < 32 {
        eprintln!("Go trying to read block hash into {out_len} bytes long buffer");
    } else {
        out_len = 32;
    }

    let global = match env.large_globals.get(global) {
        Some(global) => global,
        None => return Escape::hostio("global read out of bounds in wavmio.getGlobalStateBytes32"),
    };
    sp.write_slice(out_ptr, &global[..out_len]);
    Ok(())
}

pub fn set_global_state_bytes32(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, &mut env);
    ready_hostio(env)?;

    let global = sp.read_u64(0) as u32 as usize;
    let src_ptr = sp.read_u64(1);
    let src_len = sp.read_u64(2);
    if src_len != 32 {
        eprintln!("Go trying to set 32-byte global with a {src_len} bytes long buffer");
        return Ok(());
    }

    let slice = sp.read_slice(src_ptr, src_len);
    let slice = &slice.try_into().unwrap();
    match env.large_globals.get_mut(global) {
        Some(global) => *global = *slice,
        None => {
            return Escape::hostio("global write out of bounds in wavmio.setGlobalStateBytes32")
        }
    }
    Ok(())
}

pub fn get_global_state_u64(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, &mut env);
    ready_hostio(env)?;

    let global = sp.read_u64(0) as u32 as usize;
    match env.small_globals.get(global) {
        Some(global) => sp.write_u64(1, *global),
        None => return Escape::hostio("global read out of bounds in wavmio.getGlobalStateU64"),
    }
    Ok(())
}

pub fn set_global_state_u64(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, &mut env);
    ready_hostio(env)?;

    let global = sp.read_u64(0) as u32 as usize;
    match env.small_globals.get_mut(global) {
        Some(global) => *global = sp.read_u64(1),
        None => return Escape::hostio("global write out of bounds in wavmio.setGlobalStateU64"),
    }
    Ok(())
}

pub fn read_inbox_message(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, &mut env);
    ready_hostio(env)?;

    let inbox = &env.sequencer_messages;
    inbox_message_impl(&sp, inbox, "wavmio.readInboxMessage")
}

pub fn read_delayed_inbox_message(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, &mut env);
    ready_hostio(env)?;

    let inbox = &env.delayed_messages;
    inbox_message_impl(&sp, inbox, "wavmio.readDelayedInboxMessage")
}

/// Reads an inbox message
/// note: the order of the checks is very important.
fn inbox_message_impl(sp: &GoStack, inbox: &Inbox, name: &str) -> MaybeEscape {
    let msg_num = sp.read_u64(0);
    let offset = sp.read_u64(1);
    let out_ptr = sp.read_u64(2);
    let out_len = sp.read_u64(3);
    if out_len != 32 {
        eprintln!("Go trying to read inbox message with out len {out_len} in {name}");
        sp.write_u64(5, 0);
        return Ok(());
    }

    macro_rules! error {
        ($text:expr $(,$args:expr)*) => {{
            let text = format!($text $(,$args)*);
            return Escape::hostio(&text)
        }};
    }

    let message = match inbox.get(&msg_num) {
        Some(message) => message,
        None => error!("missing inbox message {msg_num} in {name}"),
    };

    if out_ptr + 32 > sp.memory_size() {
        error!("unknown message type in {name}");
    }
    let offset = match u32::try_from(offset) {
        Ok(offset) => offset as usize,
        Err(_) => error!("bad offset {offset} in {name}"),
    };

    let len = std::cmp::min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    sp.write_slice(out_ptr, read);
    sp.write_u64(5, read.len() as u64);
    Ok(())
}

pub fn resolve_preimage(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, &mut env);

    let name = "wavmio.resolvePreImage";

    let hash_ptr = sp.read_u64(0);
    let hash_len = sp.read_u64(1);
    let offset = sp.read_u64(3);
    let out_ptr = sp.read_u64(4);
    let out_len = sp.read_u64(5);
    if hash_len != 32 || out_len != 32 {
        eprintln!("Go trying to resolve pre image with hash len {hash_len} and out len {out_len}");
        sp.write_u64(7, 0);
        return Ok(());
    }

    macro_rules! error {
        ($text:expr $(,$args:expr)*) => {{
            let text = format!($text $(,$args)*);
            return Escape::hostio(&text)
        }};
    }

    let hash = sp.read_slice(hash_ptr, hash_len);
    let hash: &[u8; 32] = &hash.try_into().unwrap();
    let hash_hex = hex::encode(hash);

    let mut preimage = None;
    let temporary; // makes the borrow checker happy

    // see if we've cached the preimage
    if let Some((key, cached)) = &env.process.last_preimage {
        if key == hash {
            preimage = Some(cached);
        }
    }

    // see if this is a known preimage
    if preimage.is_none() {
        preimage = env.preimages.get(hash);
    }

    // see if Go has the preimage
    if preimage.is_none() {
        if let Some((writer, reader)) = &mut env.process.socket {
            socket::write_u8(writer, socket::PREIMAGE)?;
            socket::write_bytes32(writer, hash)?;
            writer.flush()?;

            if socket::read_u8(reader)? == socket::SUCCESS {
                temporary = socket::read_bytes(reader)?;
                env.process.last_preimage = Some((*hash, temporary.clone()));
                preimage = Some(&temporary);
            }
        }
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
    sp.write_slice(out_ptr, read);
    sp.write_u64(7, read.len() as u64);
    Ok(())
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
            return Ok(());
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

    let preimage_count = socket::read_u64(stream)?;
    for _ in 0..preimage_count {
        let hash = socket::read_bytes32(stream)?;
        let preimage = socket::read_bytes(stream)?;
        env.preimages.insert(hash, preimage);
    }

    if socket::read_u8(stream)? != socket::READY {
        return Escape::hostio("failed to parse global state");
    }

    let writer = BufWriter::new(socket);
    env.process.socket = Some((writer, reader));
    env.process.forks = false;
    Ok(())
}
