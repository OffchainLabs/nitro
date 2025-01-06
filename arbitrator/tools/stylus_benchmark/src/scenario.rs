// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use clap::ValueEnum;
use std::fs::File;
use std::io::Write;
use std::path::PathBuf;

#[derive(ValueEnum, Copy, Clone, PartialEq, Eq, Debug)]
#[clap(rename_all = "PascalCase")]
pub enum Scenario {
    AddI32,
    XorI32,
    CallIndirect,
}

// Programs to be benchmarked have a loop in which several similar operations are executed.
// The number of operations per loop is chosen to be large enough so the overhead related to the loop is negligible,
// but not too large to avoid a big program size.
// Keeping a small program size is important to better use CPU cache, trying to keep the code in the cache.

fn write_common_wat_beginning(wat: &mut Vec<u8>) {
    wat.write_all(b"(module\n").unwrap();
    wat.write_all(b"    (import \"debug\" \"start_benchmark\" (func $start_benchmark))\n")
        .unwrap();
    wat.write_all(b"    (import \"debug\" \"end_benchmark\" (func $end_benchmark))\n")
        .unwrap();
    wat.write_all(b"    (memory (export \"memory\") 0 0)\n")
        .unwrap();
    wat.write_all(b"    (global $ops_counter (mut i32) (i32.const 0))\n")
        .unwrap();
}

fn write_exported_func_beginning(wat: &mut Vec<u8>) {
    wat.write_all(b"    (func (export \"user_entrypoint\") (param i32) (result i32)\n")
        .unwrap();

    wat.write_all(b"        call $start_benchmark\n").unwrap();

    wat.write_all(b"        (loop $loop\n").unwrap();
    wat.write_all(b"            call $start_benchmark\n")
        .unwrap();
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

    wat.write_all(b"        call $end_benchmark\n").unwrap();

    wat.write_all(b"        i32.const 0)\n").unwrap();
    wat.write_all(b")").unwrap();
}

fn wat(
    write_specific_wat_beginning: fn(&mut Vec<u8>),
    write_wat_ops: fn(&mut Vec<u8>, usize),
) -> Vec<u8> {
    let number_of_loop_iterations = 200_000;
    let number_of_ops_per_loop_iteration = 2000;

    let mut wat = Vec::new();

    write_common_wat_beginning(&mut wat);
    write_specific_wat_beginning(&mut wat);

    write_exported_func_beginning(&mut wat);

    write_wat_ops(&mut wat, number_of_ops_per_loop_iteration);

    write_wat_end(
        &mut wat,
        number_of_loop_iterations,
        number_of_ops_per_loop_iteration,
    );

    wat.to_vec()
}

fn write_add_i32_wat_ops(wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize) {
    wat.write_all(b"            i32.const 0\n").unwrap();
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(b"            i32.const 1\n").unwrap();
        wat.write_all(b"            i32.add\n").unwrap();
    }
    wat.write_all(b"            drop\n").unwrap();
}

fn write_xor_i32_wat_ops(wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize) {
    wat.write_all(b"            i32.const 1231\n").unwrap();
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(b"            i32.const 12312313\n").unwrap();
        wat.write_all(b"            i32.xor\n").unwrap();
    }
    wat.write_all(b"            drop\n").unwrap();
}

fn write_call_indirect_wat_beginning(wat: &mut Vec<u8>) {
    wat.write_all(b"        (type $nop_func_type (func))\n")
        .unwrap();
    wat.write_all(b"        (func $nop nop)\n").unwrap();
    wat.write_all(b"        (table 1 funcref)\n").unwrap();
    wat.write_all(b"        (elem (i32.const 0) $nop)\n")
        .unwrap();
}

fn write_call_indirect_wat_ops(wat: &mut Vec<u8>, number_of_loop_iterations: usize) {
    for _ in 0..number_of_loop_iterations {
        wat.write_all(b"            i32.const 0\n").unwrap();
        wat.write_all(b"            call_indirect (type $nop_func_type)\n")
            .unwrap();
    }
}

fn noop_write_specific_wat_beginning(_: &mut Vec<u8>) {}

pub fn generate_wat(scenario: Scenario, output_wat_dir_path: Option<PathBuf>) -> Vec<u8> {
    let wat = match scenario {
        Scenario::AddI32 => wat(noop_write_specific_wat_beginning, write_add_i32_wat_ops),
        Scenario::XorI32 => wat(noop_write_specific_wat_beginning, write_xor_i32_wat_ops),
        Scenario::CallIndirect => wat(
            write_call_indirect_wat_beginning,
            write_call_indirect_wat_ops,
        ),
    };

    // print wat to file if needed
    if let Some(output_wat_dir_path) = output_wat_dir_path {
        let mut output_wat_path = output_wat_dir_path;
        output_wat_path.push(format!("{:?}.wat", scenario));
        let mut file = File::create(output_wat_path).unwrap();
        file.write_all(&wat).unwrap();
    }

    wat
}
