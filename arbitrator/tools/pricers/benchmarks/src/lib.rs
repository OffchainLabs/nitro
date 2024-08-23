// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::Rng;
use std::fmt::Display;

pub const PAGE_SIZE: usize = 64 * 1024;

pub fn begin_start<T: Display>(locals: impl Iterator<Item = T>, memory_size: usize) {
    println!("(import \"pricer\" \"toggle_timer\" (func $timer))");
    println!("(start $test)");
    memory(memory_size);
    entrypoint_stub();

    print!("(func $test (local");
    for local in locals {
        print!(" {local}");
    }
    println!(")");
}

/// Stubs the entrypoint so that our WASMs pass Stylus validation.
/// The actual function is never invoked.
pub fn entrypoint_stub() {
    println!(r#"(func (export "user_entrypoint") (param i32) (result i32) unreachable)"#);
}

pub fn memory(size: usize) {
    println!(r#"(memory (export "memory") {size} {size})"#);
}

pub fn fill_locals(locals: usize, rng: &mut impl Rng) {
    for i in 0..locals {
        let value: i64 = rng.gen();
        println!("    (local.set {i} (i64.const {value}))");
    }
}

pub fn require_locals_not_max(locals: usize) {
    // Force the use of each local by trapping if it's u64::MAX
    println!("    (block");
    for i in 0..locals {
        println!("        (local.get {})", i);
        println!("        (i64.const {})", u64::MAX as i64);
        println!("        (i64.eq)");
        println!("        (br_if 0)");
    }
    println!("        (return)");
    println!("    )");
    println!("    (unreachable)");
}

pub fn use_variables(ty: &str, count: usize) {
    println!("    (i64.const 1)");
    println!("    (i64.const 0)");
    for i in 0..count {
        println!(" ({ty}.get {})", i);
        println!(" (i64.add)");
    }
    println!("    (i64.div_u)");
    println!("    (drop)");
}

// Assumes that the first local is an i64
pub fn expect_equal<T: Display, V: Display>(ty: T, to: V) {
    println!("    local.set $check");
    println!("    (block");
    println!("        local.get $check");
    println!("        {ty}.const {to}");
    println!("        {ty}.eq");
    println!("        br_if 0");
    println!("        unreachable");
    println!("    )");
}
