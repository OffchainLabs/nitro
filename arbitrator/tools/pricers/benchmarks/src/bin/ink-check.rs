// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const COUNT: usize = 7_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    
    println!("(import \"pricer\" \"toggle_timer\" (func $timer))");
    println!("(start $test)");
    memory(1);
    entrypoint_stub();

    println!("(global $gas (mut i64) (i64.const {}))", rng.gen::<i64>());
    println!("(global $status (mut i32) (i32.const {}))", 0);

    println!("(func $test");
    println!("    call $timer");
    for _ in 0..COUNT {
        let cost = rng.gen::<u64>() % 10_000;
        println!("    global.get $gas");
        println!("    i64.const {cost}");
        println!("    i64.lt_u");
        println!("    (if (then");
        println!("        i32.const 1");
        println!("        global.set $status");
        println!("        unreachable");
        println!("    ))");
        println!("    global.get $gas");
        println!("    i64.const {cost}");
        println!("    i64.sub");
        println!("    global.set $gas");
    }
    println!("    call $timer");
    println!(")");
}
