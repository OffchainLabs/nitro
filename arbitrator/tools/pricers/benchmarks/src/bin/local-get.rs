// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const LOCALS: usize = 4095;
const GETS: usize = 10_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::repeat("i64").take(LOCALS), 0);
    println!("(local $check i64)");
    fill_locals(LOCALS, &mut rng);

    println!("    call $timer");
    println!("    i64.const 0");
    let local = Uniform::from(0..LOCALS);
    for _ in 0..GETS {
        println!("    (local.get {})", rng.sample(local));
        println!("    (i64.add)");
    }
    println!("    call $timer");
    expect_equal("i64", 100);
    println!(")");
}
