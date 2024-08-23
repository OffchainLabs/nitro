// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const FUNCS: usize = 9_000;
const OPS: usize = 10_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    println!(r#"(import "pricer" "toggle_timer" (func $timer))"#);
    println!("(global $check (mut i64) (i64.const 0))");
    for _ in 0..FUNCS {
        println!("(func");
        println!("    (global.set $check (i64.const {}))", rng.gen::<i64>());
        println!(")");
    }

    memory(0);
    entrypoint_stub();

    println!("(start $test)");
    println!("(table {FUNCS} funcref)");
    print!("(elem (i32.const 0)");
    for i in 0..FUNCS {
        print!(" {}", i + 1);
    }
    println!(")");
    println!("(func $test");
    println!("    (call $timer)");
    let funcs = Uniform::from(0..FUNCS);
    for _ in 0..OPS / 2 {
        println!("    (call_indirect (i32.const {}))", rng.sample(&funcs));
    }
    println!("    (call $timer)");
    // Require that the global is not equal to 0 (which would trap)
    println!("    (i64.const 1)");
    println!("    (global.get $check)");
    println!("    (i64.div_u)");
    println!("    (drop)");
    println!(")");
}
