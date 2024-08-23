use rand::{Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const OPS: usize = 10_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::once("$check i64"), 0);
    println!("    (call $timer)");
    let mut expected_sum: i64 = 0;
    let value = rng.gen::<i64>();
    expected_sum = expected_sum.wrapping_add(value);
    println!("    (i64.const {})", value);
    for _ in 0..OPS {
        let value = rng.gen::<i64>();
        expected_sum = expected_sum.wrapping_add(value);
        println!("    (i64.const {})", value);
        println!("    (i32.const {})", rng.gen_bool(0.5) as i32);
        println!("    (select)");
    }
    println!("    (call $timer)");
    // Require that the result is not equal to 0 (which would trap)
    println!("    (local.set $check)");
    println!("    (i64.const 1)");
    println!("    (local.get $check)");
    println!("    (i64.div_u)");
    println!("    (drop)");
    println!(")");
}
