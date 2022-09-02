// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::{Escape, GoStack, MaybeEscape, WasmEnvArc};

pub type Bytes32 = [u8; 32];

pub fn get_global_state_bytes32(env: &WasmEnvArc, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, env);

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

pub fn set_global_state_bytes32(env: &WasmEnvArc, sp: u32) -> MaybeEscape {
    let (sp, mut env) = GoStack::new(sp, env);

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

pub fn get_global_state_u64(env: &WasmEnvArc, sp: u32) -> MaybeEscape {
    let (sp, env) = GoStack::new(sp, env);
    let global = sp.read_u64(0) as u32 as usize;
    match env.small_globals.get(global) {
        Some(global) => sp.write_u64(1, *global),
        None => return Escape::hostio("global read out of bounds in wavmio.getGlobalStateU64"),
    }
    Ok(())
}

pub fn set_global_state_u64(env: &WasmEnvArc, sp: u32) -> MaybeEscape {
    let (sp, mut env) = GoStack::new(sp, env);
    let global = sp.read_u64(0) as u32 as usize;
    match env.small_globals.get_mut(global) {
        Some(global) => *global = sp.read_u64(1),
        None => return Escape::hostio("global write out of bounds in wavmio.setGlobalStateU64"),
    }
    Ok(())
}

pub fn read_inbox_message(env: &WasmEnvArc, sp: u32) -> MaybeEscape {
    let (sp, mut env) = GoStack::new(sp, env);

    let msg_num = sp.read_u64(0);
    let offset = sp.read_u64(1);
    let out_ptr = sp.read_u64(2);
    let out_len = sp.read_u64(3);
    if out_len != 32 {
        eprintln!("Go attempting to read inbox message with out len {out_len}");
        sp.write_u64(5, 0);
        return Ok(());
    }

    let message = [0; 64];
      
    sp.write_slice(out_ptr, &message);
    sp.write_u64(5, message.len() as u64);
    Ok(())
}

pub fn read_delayed_inbox_message(env: &WasmEnvArc, sp: u32) -> MaybeEscape {
    let (sp, mut env) = GoStack::new(sp, env);
    Escape::hostio("unimplemented")
}

pub fn resolve_preimage(env: &WasmEnvArc, sp: u32) -> MaybeEscape {
    let (sp, mut env) = GoStack::new(sp, env);
    Escape::hostio("unimplemented")
}
