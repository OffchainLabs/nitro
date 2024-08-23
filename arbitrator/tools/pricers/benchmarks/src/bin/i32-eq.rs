// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const OPS: usize = 30_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::once("$check i32"), 0);
    println!("    call $timer");

    let mut expected: i32 = 0;
    let value: i32 = rng.gen();
    expected = expected.wrapping_add(value);
    println!("    i32.const {}", value);

    for _ in 0..OPS {
        let value = rng.gen::<i32>();
        expected = expected.wrapping_add(value);
        print!("    (i32.eq");
        println!(" (i32.const {}))", value);
    }
    println!("    call $timer");
    expect_equal("i32", expected);
    println!(")");
}
