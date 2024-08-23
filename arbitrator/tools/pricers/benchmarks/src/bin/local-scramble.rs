// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const LOCALS: usize = 4096;
const ADDS: usize = 10_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::repeat("i64").take(LOCALS), 0);
    println!("    (call $timer)");
    fill_locals(LOCALS, &mut rng);
    let local = Uniform::from(0..LOCALS);
    for _ in 0..ADDS {
        println!("    (local.get {})", rng.sample(local));
        println!("    (local.get {})", rng.sample(local));
        println!("    (i64.add)");
        println!("    (local.set {})", rng.sample(local));
    }
    println!("    (call $timer)");
    require_locals_not_max(LOCALS);
    println!(")");
}
