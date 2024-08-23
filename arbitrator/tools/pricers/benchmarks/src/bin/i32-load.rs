// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const PAGES: usize = 128;
const OPS: usize = 10_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    println!(r#"(import "pricer" "toggle_timer" (func $timer))"#);
    memory(PAGES);
    entrypoint_stub();

    println!("(start $test)");
    println!("(func $test (local $check i32)");
    fill_memory(PAGES, &mut rng);

    println!("    (call $timer)");
    println!("    (i32.const 0)");
    let memory = Uniform::from(0..=(PAGES * PAGE_SIZE - 8));
    for _ in 0..OPS {
        println!("    (i32.load (i32.const {}))", rng.sample(&memory));
        println!("    (i32.add)");
    }
    println!("    (call $timer)");
    println!("    (local.set $check)");
    println!("    (block");
    println!("        (local.get $check)");
    println!("        (i32.eqz)");
    println!("        (br_if 0)");
    println!("        (return)");
    println!("    )");
    println!("    (unreachable)");
    println!(")");
}

pub fn fill_memory(pages: usize, rng: &mut impl Rng) {
    let limit = pages * PAGE_SIZE - 8;
    let mut step = 0;
    while step < limit {
        let value: i32 = rng.gen();
        println!("    (i32.store (i32.const {step}) (i32.const {value}))");
        step += rng.gen::<usize>() % 2048;
    }
}
