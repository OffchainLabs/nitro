use rand::{distributions::Uniform, Rng, SeedableRng};
use rand_chacha::ChaCha8Rng;
use wasm_benchmarks::*;

const LOCALS: usize = 4_000;
const ITERS: usize = 8_000;

fn main() {
    let mut rng = ChaCha8Rng::seed_from_u64(0);
    begin_start(std::iter::repeat("i64").take(LOCALS), 0);
    fill_locals(LOCALS, &mut rng);
    println!("    (call $timer)");
    let local = Uniform::from(0..LOCALS);
    for _ in 0..ITERS {
        println!("    (local.get {})", rng.sample(&local));
        println!("    (i64.and (i64.const 1))");
        println!("    (i64.eqz)");
        println!("    (if (then");
        println!("        (local.get {})", rng.sample(&local));
        println!("        (local.get {})", rng.sample(&local));
        println!("        (i64.add)");
        println!("        (local.set {})", rng.sample(&local));
        println!("    ))");
    }
    println!("    (call $timer)");
    use_variables("local", LOCALS);
    println!(")");
}
