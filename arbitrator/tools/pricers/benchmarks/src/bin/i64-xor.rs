// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const OPS: usize = 20_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::once("$check i64"), 0);
    println!("    call $timer");

    let mut expected: i64 = 0;
    let value: i64 = rng.gen();
    expected = expected.wrapping_add(value);
    println!("    i64.const {}", value);

    for _ in 0..OPS {
        let value = rng.gen::<i64>();
        expected = expected.wrapping_add(value);
        print!("    (i64.xor");
        println!(" (i64.const {}))", value);
    }
    println!("    call $timer");
    expect_equal("i64", expected);
    println!(")");
}
