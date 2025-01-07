// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::scenarios::{call, call_indirect, global_get, global_set, i32_add, i32_eqz, i32_xor, if_op, select};
use clap::ValueEnum;
use std::fs::File;
use std::io::Write;
use std::path::PathBuf;

#[derive(ValueEnum, Copy, Clone, PartialEq, Eq, Debug)]
#[clap(rename_all = "PascalCase")]
pub enum Scenario {
    I32Add,
    I32Eqz,
    I32Xor,
    Call,
    CallIndirect,
    GlobalGet,
    GlobalSet,
    If,
    Select,
}

trait ScenarioWatGenerator {
    fn write_specific_wat_beginning(&self, wat: &mut Vec<u8>);
    fn write_wat_ops(&self, wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize);
}

impl ScenarioWatGenerator for Scenario {
    fn write_specific_wat_beginning(&self, wat: &mut Vec<u8>) {
        match self {
            Scenario::Call => call::write_specific_wat_beginning(wat),
            Scenario::CallIndirect => call_indirect::write_specific_wat_beginning(wat),
            Scenario::GlobalGet => global_get::write_specific_wat_beginning(wat),
            Scenario::GlobalSet => global_set::write_specific_wat_beginning(wat),
            Scenario::I32Add => i32_add::write_specific_wat_beginning(wat),
            Scenario::I32Eqz => i32_eqz::write_specific_wat_beginning(wat),
            Scenario::I32Xor => i32_xor::write_specific_wat_beginning(wat),
            Scenario::If => if_op::write_specific_wat_beginning(wat),
            Scenario::Select => select::write_specific_wat_beginning(wat),
        }
    }

    fn write_wat_ops(&self, wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize) {
        match self {
            Scenario::Call => call::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::CallIndirect => {
                call_indirect::write_wat_ops(wat, number_of_ops_per_loop_iteration)
            }
            Scenario::GlobalGet => global_get::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::GlobalSet => global_set::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Add => i32_add::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Eqz => i32_eqz::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Xor => i32_xor::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::If => if_op::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::Select => select::write_wat_ops(wat, number_of_ops_per_loop_iteration),
        }
    }
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

pub fn generate_wat(scenario: Scenario, output_wat_dir_path: Option<PathBuf>) -> Vec<u8> {
    let number_of_loop_iterations = 200_000;
    let number_of_ops_per_loop_iteration = 2000;

    let mut wat = Vec::new();

    write_common_wat_beginning(&mut wat);
    scenario.write_specific_wat_beginning(&mut wat);

    write_exported_func_beginning(&mut wat);

    scenario.write_wat_ops(&mut wat, number_of_ops_per_loop_iteration);

    write_wat_end(
        &mut wat,
        number_of_loop_iterations,
        number_of_ops_per_loop_iteration,
    );

    // print wat to file if needed
    if let Some(output_wat_dir_path) = output_wat_dir_path {
        let mut output_wat_path = output_wat_dir_path;
        output_wat_path.push(format!("{:?}.wat", scenario));
        let mut file = File::create(output_wat_path).unwrap();
        file.write_all(&wat).unwrap();
    }

    wat
}
