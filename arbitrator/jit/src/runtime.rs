// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

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

/// go side: λ(code int32)
pub fn wasm_exit(env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let mut sp = GoStack::simple(sp, &env);
    Escape::exit(sp.read_u32())
}

/// go side: λ(fd uintptr, p pointer, len int32)
pub fn wasm_write(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let fd = sp.read_u64();
    let ptr = sp.read_u64();
    let len = sp.read_u32();
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

/// go side: λ() int64
pub fn nanotime1(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    env.go_state.time += env.go_state.time_interval;
    sp.write_u64(env.go_state.time);
}

/// go side: λ() (seconds int64, nanos int32)
pub fn walltime(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    env.go_state.time += env.go_state.time_interval;
    sp.write_u64(env.go_state.time / 1_000_000_000);
    sp.write_u32((env.go_state.time % 1_000_000_000) as u32);
}

/// go side: λ() (seconds int64, nanos int32)
pub fn walltime1(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    env.go_state.time += env.go_state.time_interval;
    sp.write_u64(env.go_state.time / 1_000_000_000);
    sp.write_u64(env.go_state.time % 1_000_000_000);
}

/// go side: λ() (delay int64) int32
pub fn schedule_timeout_event(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);
    let mut time = sp.read_u64();
    time = time.saturating_mul(1_000_000); // milliseconds to nanoseconds
    time = time.saturating_add(env.go_state.time); // add the current time to the delay

    let timeouts = &mut env.go_state.timeouts;
    let id = timeouts.next_id;
    timeouts.next_id += 1;
    timeouts.times.push(TimeoutInfo { time, id });
    timeouts.pending_ids.insert(id);

    sp.write_u32(id);
}

/// go side: λ(id int32)
pub fn clear_timeout_event(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let id = sp.read_u32();
    if !env.go_state.timeouts.pending_ids.remove(&id) {
        eprintln!("Go attempting to clear not pending timeout event {id}");
    }
}

/// go side: λ(dest []byte)
pub fn get_random_data(mut env: WasmEnvMut, sp: u32) {
    let (mut sp, env) = GoStack::new(sp, &mut env);

    let mut ptr = u32::try_from(sp.read_u64()).expect("Go getRandomData pointer not a u32");
    let mut len = sp.read_u64();
    while len >= 4 {
        let next = env.go_state.rng.next_u32();
        sp.write_u32_raw(ptr, next);
        ptr += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = env.go_state.rng.next_u32();
        for _ in 0..len {
            sp.write_u8_raw(ptr, rem as u8);
            ptr += 1;
            rem >>= 8;
        }
    }
}
