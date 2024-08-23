use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const PAGES: usize = 128;
const OPS: usize = 12_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    println!(r#"(import "pricer" "toggle_timer" (func $timer))"#);
    memory(PAGES);
    entrypoint_stub();

    println!("(start $test)");
    println!("(func $test (local $test i32)");

    let memory = Uniform::from(0..=(PAGES * PAGE_SIZE - 8));

    // prep the memory
    for _ in 0..(OPS / 6) {
        println!(
            "    (i32.store (i32.const {}) (i32.const {}))",
            rng.sample(&memory),
            rng.gen::<i32>(),
        );
    }

    println!("    call $timer");

    for _ in 0..OPS {
        println!(
            "    (i32.store (i32.const {}) (i32.const {}))",
            rng.sample(&memory),
            rng.gen::<i32>(),
        );
    }
    println!("    call $timer");
    println!("    (loop");
    println!("        (local.get $test)");
    println!("        (i32.add (i32.const 8))");
    println!("        (local.tee $test)");
    println!("        (i32.load)");
    println!("        (i32.eqz)");
    println!("        (br_if 0)");
    println!("    )");
    println!(")");
}
