// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const LOCALS: usize = 4096;
const STACK: usize = 4096;
const REPETITIONS: usize = 2;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::repeat("i64").take(LOCALS), 0);
    fill_locals(LOCALS, &mut rng);
    println!("    (call $timer)");
    let local = Uniform::from(0..LOCALS);
    for _ in 0..REPETITIONS {
        for _ in 0..STACK {
            println!("    (local.get {})", rng.sample(local));
        }
        for _ in 0..STACK {
            println!("    (local.set {})", rng.sample(local));
        }
    }
    println!("    (call $timer)");
    require_locals_not_max(LOCALS);
    println!(")");
}
