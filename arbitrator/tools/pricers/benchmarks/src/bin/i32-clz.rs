use rand::{Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const OPS: usize = 30_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::once("$check i32"), 0);
    println!("    (call $timer)");
    println!("    (i32.const 0)");
    for _ in 0..OPS {
        let value = rng.gen::<i32>();
        println!("    (i32.const {})", value);
        println!("    (i32.add)");
        println!("    (i32.clz)");
    }
    println!("    (call $timer)");
    // Require that the clz is not equal to -1 (which would trap)
    println!("    (i32.add (i32.const 1))");
    println!("    (local.set $check)");
    println!("    (i32.const 1)");
    println!("    (local.get $check)");
    println!("    (i32.div_u)");
    println!("    (drop)");
    println!(")");
}
