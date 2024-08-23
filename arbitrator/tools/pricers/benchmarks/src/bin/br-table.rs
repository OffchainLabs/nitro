// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const DEPTH: usize = 2;
const OPS: usize = 1000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    println!(r#"(import "pricer" "toggle_timer" (func $timer))"#);
    println!("(global (mut i64) (i64.const 0))");
    println!("(start $test)");

    memory(0);
    entrypoint_stub();

    println!("(func $noop)");
    println!("(func $test");
    println!("    (call $timer)");
    let depth = Uniform::from(0..DEPTH);
    for _ in 0..OPS {
        for _ in 0..DEPTH {
            println!("    (block");
        }
        println!(
            "        (i32.const {})",
            rng.sample(&depth).saturating_sub(1)
        );
        println!("        (i32.const 1)");
        println!("        (i32.add)");
        println!("        (br_table ");
        for i in 0..DEPTH {
            print!(" {i}");
        }
        println!(")");
        for _ in 0..DEPTH {
            //println!("    (call $noop)");

            println!("    (global.set 0 (i64.const 0))");
            println!("    )");
        }
    }
    println!("    (call $timer)");
    println!(")");
}
