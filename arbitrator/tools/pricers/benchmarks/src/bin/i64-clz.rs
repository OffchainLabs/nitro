use rand::{Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const OPS: usize = 20_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::once("$check i64"), 0);
    println!("    (call $timer)");
    println!("    (i64.const 0)");
    for _ in 0..OPS {
        let value = rng.gen::<i64>();
        println!("    (i64.const {})", value);
        println!("    (i64.add)");
        println!("    (i64.clz)");
    }
    println!("    (call $timer)");
    // Require that the clz is not equal to -1 (which would trap)
    println!("    (i64.add (i64.const 1))");
    println!("    (local.set $check)");
    println!("    (i64.const 1)");
    println!("    (local.get $check)");
    println!("    (i64.div_u)");
    println!("    (drop)");
    println!(")");
}
