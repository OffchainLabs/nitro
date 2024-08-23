// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const GLOBALS: usize = 10_000;
const OPS: usize = 10_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    println!(r#"(import "pricer" "toggle_timer" (func $timer))"#);
    for _ in 0..GLOBALS {
        println!("(global (mut i64) (i64.const {}))", rng.gen::<i64>());
    }

    memory(0);
    entrypoint_stub();

    println!("(start $test)");
    println!("(func $test (local $check i64)");
    println!("    (call $timer)");
    println!("    (i64.const 0)");
    let global = Uniform::from(0..GLOBALS);
    for _ in 0..OPS {
        println!("    (global.get {})", rng.sample(&global));
        println!("    (i64.add)");
    }
    println!("    (call $timer)");
    // Require that the sum is not equal to 0 (which would trap)
    println!("    (local.set $check)");
    println!("    (i64.const 1)");
    println!("    (local.get $check)");
    println!("    (i64.div_u)");
    println!("    (drop)");
    println!(")");
}
