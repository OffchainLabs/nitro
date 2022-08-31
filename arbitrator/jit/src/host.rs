// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::{GoStack, TimeoutInfo, WasmEnvArc};

use rand::RngCore;

use std::io::Write;

pub fn go_debug(x: u32) {
    println!("go debug: {x}")
}

pub fn runtime_reset_memory_data_view(_: u32) {}

pub fn runtime_wasm_exit(env: &WasmEnvArc, sp: u32) {
    let sp = GoStack::new(sp, env);
    std::process::exit(sp.read_u32(0) as i32);
}

pub fn runtime_wasm_write(env: &WasmEnvArc, sp: u32) {
    let sp = GoStack::new(sp, env);
    let fd = sp.read_u64(0);
    let ptr = sp.read_u64(1);
    let len = sp.read_u32(2);
    let buf = sp.read_slice(ptr, len);
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

pub fn runtime_nanotime1(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new_with_env(sp, env);
    env.time += env.time_interval;
    sp.write_u64(0, env.time);
}

pub fn runtime_walltime(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new_with_env(sp, env);
    env.time += env.time_interval;
    sp.write_u64(0, env.time / 1_000_000_000);
    sp.write_u32(1, (env.time % 1_000_000_000) as u32);
}

pub fn runtime_schedule_timeout_event(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new_with_env(sp, env);
    let mut time = sp.read_u64(0);
    time = time.saturating_mul(1_000_000); // milliseconds to nanoseconds
    time = time.saturating_add(env.time); // add the current time to the delay

    let timeouts = &mut env.timeouts;
    let id = timeouts.next_id;
    timeouts.next_id += 1;
    timeouts.times.push(TimeoutInfo { time, id });
    timeouts.pending_ids.insert(id);

    sp.write_u32(1, id);
}

pub fn runtime_clear_timeout_event(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new_with_env(sp, env);

    let id = sp.read_u32(0);
    if !env.timeouts.pending_ids.remove(&id) {
        eprintln!("Go attempting to clear not pending timeout event {id}");
    }
}

pub fn runtime_get_random_data(env: &WasmEnvArc, sp: u32) {
    let (sp, mut env) = GoStack::new_with_env(sp, env);

    let mut ptr = u32::try_from(sp.read_u64(0)).expect("Go getRandomData pointer not a u32");
    let mut len = sp.read_u64(1);
    while len >= 4 {
        sp.write_u32_ptr(ptr, env.rng.next_u32());
        ptr += 4;
        len -= 4;
    }
    if len > 0 {
        let mut rem = env.rng.next_u32();
        for _ in 0..len {
            sp.write_u8_ptr(ptr, rem as u8);
            ptr += 1;
            rem >>= 8;
        }
    }
}
