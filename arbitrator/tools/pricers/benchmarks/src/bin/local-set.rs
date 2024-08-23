// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const LOCALS: usize = 4095;
const SETS: usize = 10_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::repeat("i64").take(LOCALS), 0);
    println!(" (call $timer)");
    let local = Uniform::from(0..LOCALS);
    // Don't let this generate u64::MAX
    let value = Uniform::from(0..u64::MAX);
    for _ in 0..SETS {
        println!(" (i64.const {})", rng.sample(value) as i64);
        println!(" (local.set {})", rng.sample(local));
    }
    println!(" (call $timer)");
    require_locals_not_max(LOCALS);
    println!(")");
}
