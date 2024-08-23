// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use wasm_benchmarks::*;

const FUNCS: usize = 250;
const ITERS: usize = 100;

fn main() {
    println!("(import \"pricer\" \"toggle_timer\" (func $timer))");
    println!("(global $check (mut i64) (i64.const 0))");
    for f in 0..FUNCS {
        print!("(func $func{f} (param");
        for _ in 0..f {
            print!(" i32");
        }
        println!(")");
        for i in 0..f {
            print!("    (local.get {i})");
        }
        if f == 0 {
            print!("    (global.set $check (i64.const 1))");
        } else {
            print!("    (call_indirect (param");
            for _ in 0..f - 1 {
                print!(" i32");
            }
            println!("))");
        }
        println!(")");
    }

    memory(0);
    entrypoint_stub();

    println!("(start $test)");
    println!("(table {FUNCS} funcref)");
    print!("(elem (i32.const 0)");
    for i in 0..FUNCS {
        print!(" $func{i}");
    }
    println!(")");
    println!("(func $test");
    println!("    (call $timer)");
    for _ in 0..ITERS {
        for i in 0..(FUNCS - 1) {
            println!("    (i32.const {i})");
        }
        println!("    (call $func{})", FUNCS - 1);
    }
    println!("    (call $timer)");
    // Require that the global is not equal to 0 (which would trap)
    println!("    (i64.const 1)");
    println!("    (global.get $check)");
    println!("    (i64.div_u)");
    println!("    (drop)");
    println!(")");
}
