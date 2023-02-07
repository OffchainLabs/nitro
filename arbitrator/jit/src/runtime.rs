// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    gostack::{GoStack, TimeoutInfo},
    machine::{Escape, MaybeEscape, WasmEnvMut},
};

use rand::RngCore;

use std::io::Write;

pub fn go_debug(x: u32) {
    println!("go debug: {x}")
}

pub fn reset_memory_data_view(_: u32) {}

pub fn wasm_exit(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let (sp, _) = GoStack::new(sp, &mut env);
    Escape::exit(sp.read_u32(0))
}

pub fn wasm_write(mut env: WasmEnvMut, sp: u32) {
    let (sp, _) = GoStack::new(sp, &mut env);
    let fd = sp.read_u64(0);
    let ptr = sp.read_u64(1);
    let len = sp.read_u32(2);
    let buf = sp.read_slice(ptr, len.into());
    if fd == 2 {
        let stderr = std::io::stderr();
        let mut stderr = stderr.lock();
        stderr.write_all(&buf).unwrap();
    } else {
        let stdout = std::io::stdout();
        let mut stdout = stdout.lock();
        stdout.write_all(&buf).unwrap();
    }
}

pub fn nanotime1(mut env: WasmEnvMut, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, &mut env);
    env.go_state.time += env.go_state.time_interval;
    sp.write_u64(0, env.go_state.time);
}

pub fn walltime(mut env: WasmEnvMut, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, &mut env);
    env.go_state.time += env.go_state.time_interval;
    sp.write_u64(0, env.go_state.time / 1_000_000_000);
    sp.write_u32(1, (env.go_state.time % 1_000_000_000) as u32);
}

pub fn walltime1(mut env: WasmEnvMut, sp: u32) {
    let (sp, mut env) = GoStack::new(sp, &mut env);
    env.go_state.time += env.go_state.time_interval;
    sp.write_u64(0, env.go_state.time / 1_000_000_000);
    sp.write_u64(1, env.go_state.time % 1_000_000_000);
}

pub fn schedule_timeout_event(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);
    let mut time = sp.read_u64(0);
    time = time.saturating_mul(1_000_000); // milliseconds to nanoseconds
    time = time.saturating_add(env.go_state.time); // add the current time to the delay

    let timeouts = &mut env.go_state.timeouts;
    let id = timeouts.next_id;
    timeouts.next_id += 1;
    timeouts.times.push(TimeoutInfo { time, id });
    timeouts.pending_ids.insert(id);

    sp.write_u32(1, id);
}

pub fn clear_timeout_event(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);

    let id = sp.read_u32(0);
    if !env.go_state.timeouts.pending_ids.remove(&id) {
        eprintln!("Go attempting to clear not pending timeout event {id}");
    }
}

pub fn get_random_data(mut env: WasmEnvMut, sp: u32) {
    let (sp, env) = GoStack::new(sp, &mut env);

    let mut ptr = u32::try_from(sp.read_u64(0)).expect("Go getRandomData pointer not a u32");
    let mut len = sp.read_u64(1);
    while len >= 4 {
        let next = env.go_state.rng.next_u32();
        sp.write_u32_ptr(ptr, next);
        ptr += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = env.go_state.rng.next_u32();
        for _ in 0..len {
            sp.write_u8_ptr(ptr, rem as u8);
            ptr += 1;
            rem >>= 8;
        }
    }
}
