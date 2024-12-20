// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::fs::File;
use std::io::Write;
use std::path::PathBuf;
use strum_macros::{Display, EnumIter, EnumString};

#[derive(Copy, Clone, PartialEq, Eq, Debug, EnumString, Display, EnumIter)]
pub enum Scenario {
    #[strum(serialize = "add_i32")]
    AddI32,
    #[strum(serialize = "xor_i32")]
    XorI32,
}

fn write_wat_beginning(wat: &mut Vec<u8>) {
    wat.write_all(b"(module\n").unwrap();
    wat.write_all(b"    (import \"debug\" \"start_benchmark\" (func $start_benchmark))\n")
        .unwrap();
    wat.write_all(b"    (import \"debug\" \"end_benchmark\" (func $end_benchmark))\n")
        .unwrap();
    wat.write_all(b"    (memory (export \"memory\") 0 0)\n")
        .unwrap();
    wat.write_all(b"    (global $ops_counter (mut i32) (i32.const 0))\n")
        .unwrap();
    wat.write_all(b"    (func (export \"user_entrypoint\") (param i32) (result i32)\n")
        .unwrap();

    wat.write_all(b"            call $start_benchmark\n")
        .unwrap();

    wat.write_all(b"        (loop $loop\n").unwrap();
}

fn write_wat_end(
    wat: &mut Vec<u8>,
    number_of_loop_iterations: usize,
    number_of_ops_per_loop_iteration: usize,
) {
    let number_of_ops = number_of_loop_iterations * number_of_ops_per_loop_iteration;

    // update ops_counter
    wat.write_all(b"            global.get $ops_counter\n")
        .unwrap();
    wat.write_all(
        format!(
            "            i32.const {}\n",
            number_of_ops_per_loop_iteration
        )
        .as_bytes(),
    )
    .unwrap();
    wat.write_all(b"            i32.add\n").unwrap();
    wat.write_all(b"            global.set $ops_counter\n")
        .unwrap();

    // check if we need to continue looping
    wat.write_all(b"            global.get $ops_counter\n")
        .unwrap();
    wat.write_all(format!("            i32.const {}\n", number_of_ops).as_bytes())
        .unwrap();
    wat.write_all(b"            i32.lt_s\n").unwrap();
    wat.write_all(b"            br_if $loop)\n").unwrap();

    wat.write_all(b"            call $end_benchmark\n").unwrap();

    wat.write_all(b"        i32.const 0)\n").unwrap();
    wat.write_all(b")").unwrap();
}

fn generate_add_i32_wat() -> Vec<u8> {
    let number_of_loop_iterations = 10_000;
    let number_of_ops_per_loop_iteration = 2000;

    let mut wat = Vec::new();

    write_wat_beginning(&mut wat);

    // ops to be benchmarked
    wat.write_all(b"            i32.const 0\n").unwrap();
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(b"            i32.const 1\n").unwrap();
        wat.write_all(b"            i32.add\n").unwrap();
    }
    wat.write_all(b"            drop\n").unwrap();

    write_wat_end(
        &mut wat,
        number_of_loop_iterations,
        number_of_ops_per_loop_iteration,
    );

    wat.to_vec()
}

fn generate_xor_i32_wat() -> Vec<u8> {
    let number_of_loop_iterations = 10_000;
    let number_of_ops_per_loop_iteration = 2000;

    let mut wat = Vec::new();

    write_wat_beginning(&mut wat);

    // ops to be benchmarked
    wat.write_all(b"            i32.const 1231\n").unwrap();
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(b"            i32.const 12312313\n").unwrap();
        wat.write_all(b"            i32.xor\n").unwrap();
    }
    wat.write_all(b"            drop\n").unwrap();

    write_wat_end(
        &mut wat,
        number_of_loop_iterations,
        number_of_ops_per_loop_iteration,
    );

    wat.to_vec()
}

pub fn generate_wat(scenario: Scenario, output_wat_dir_path: Option<PathBuf>) -> Vec<u8> {
    let wat = match scenario {
        Scenario::AddI32 => generate_add_i32_wat(),
        Scenario::XorI32 => generate_xor_i32_wat(),
    };

    // print wat to file if needed
    if let Some(output_wat_dir_path) = output_wat_dir_path {
        let mut output_wat_path = output_wat_dir_path;
        output_wat_path.push(format!("{}.wat", scenario));
        let mut file = File::create(output_wat_path).unwrap();
        file.write_all(&wat).unwrap();
    }

    wat
}
