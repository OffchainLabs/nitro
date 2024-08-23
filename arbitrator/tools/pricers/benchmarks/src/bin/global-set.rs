use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const GLOBALS: usize = 10_000;
const OPS: usize = 6_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    println!(r#"(import "pricer" "toggle_timer" (func $timer))"#);
    for _ in 0..GLOBALS {
        println!("(global (mut i64) (i64.const {}))", rng.gen::<i64>());
    }

    memory(0);
    entrypoint_stub();

    println!("(start $test)");
    println!("(func $test (local $check i64)");
    println!("    call $timer");
    let global = Uniform::from(0..GLOBALS);
    for _ in 0..OPS {
        println!(
            "    (global.set {} (i64.const {}))",
            rng.sample(&global),
            rng.gen::<i64>(),
        );
    }
    println!("    call $timer");
    use_variables("global", GLOBALS);
    println!(")");
}
