// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::scenarios::{
    call, call_indirect, global_get, global_set, i32_add, i32_and, i32_clz, i32_ctz, i32_div_s,
    i32_div_u, i32_eq, i32_eqz, i32_ge_s, i32_ge_u, i32_gt_s, i32_gt_u, i32_le_s, i32_le_u,
    i32_lt_s, i32_lt_u, i32_mul, i32_ne, i32_or, i32_popcnt, i32_rem_s, i32_rem_u, i32_rotl,
    i32_rotr, i32_shl, i32_shr_s, i32_shr_u, i32_sub, i32_wrap_i64, i32_xor, if_op, select,
};
use clap::ValueEnum;
use std::fs::File;
use std::io::Write;
use std::path::PathBuf;

#[derive(ValueEnum, Copy, Clone, PartialEq, Eq, Debug)]
#[clap(rename_all = "PascalCase")]
pub enum Scenario {
    I32Add,
    I32And,
    I32Clz,
    I32Ctz,
    I32DivS,
    I32DivU,
    I32Eq,
    I32Eqz,
    I32GeS,
    I32GeU,
    I32GtU,
    I32GtS,
    I32LeU,
    I32LeS,
    I32LtU,
    I32LtS,
    I32Mul,
    I32Ne,
    I32Or,
    I32Popcnt,
    I32RemS,
    I32RemU,
    I32Rotl,
    I32Rotr,
    I32Shl,
    I32ShrS,
    I32ShrU,
    I32Sub,
    I32WrapI64,
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
            Scenario::I32And => i32_and::write_specific_wat_beginning(wat),
            Scenario::I32Clz => i32_clz::write_specific_wat_beginning(wat),
            Scenario::I32Ctz => i32_ctz::write_specific_wat_beginning(wat),
            Scenario::I32DivS => i32_div_s::write_specific_wat_beginning(wat),
            Scenario::I32DivU => i32_div_u::write_specific_wat_beginning(wat),
            Scenario::I32Eq => i32_eq::write_specific_wat_beginning(wat),
            Scenario::I32Eqz => i32_eqz::write_specific_wat_beginning(wat),
            Scenario::I32GeS => i32_ge_s::write_specific_wat_beginning(wat),
            Scenario::I32GeU => i32_ge_u::write_specific_wat_beginning(wat),
            Scenario::I32GtU => i32_gt_u::write_specific_wat_beginning(wat),
            Scenario::I32GtS => i32_gt_s::write_specific_wat_beginning(wat),
            Scenario::I32LeU => i32_le_u::write_specific_wat_beginning(wat),
            Scenario::I32LeS => i32_le_s::write_specific_wat_beginning(wat),
            Scenario::I32LtU => i32_lt_u::write_specific_wat_beginning(wat),
            Scenario::I32LtS => i32_lt_s::write_specific_wat_beginning(wat),
            Scenario::I32Mul => i32_mul::write_specific_wat_beginning(wat),
            Scenario::I32Ne => i32_ne::write_specific_wat_beginning(wat),
            Scenario::I32Or => i32_or::write_specific_wat_beginning(wat),
            Scenario::I32Popcnt => i32_popcnt::write_specific_wat_beginning(wat),
            Scenario::I32RemS => i32_rem_s::write_specific_wat_beginning(wat),
            Scenario::I32RemU => i32_rem_u::write_specific_wat_beginning(wat),
            Scenario::I32Rotl => i32_rotl::write_specific_wat_beginning(wat),
            Scenario::I32Rotr => i32_rotr::write_specific_wat_beginning(wat),
            Scenario::I32Shl => i32_shl::write_specific_wat_beginning(wat),
            Scenario::I32ShrS => i32_shr_s::write_specific_wat_beginning(wat),
            Scenario::I32ShrU => i32_shr_u::write_specific_wat_beginning(wat),
            Scenario::I32Sub => i32_sub::write_specific_wat_beginning(wat),
            Scenario::I32WrapI64 => i32_wrap_i64::write_specific_wat_beginning(wat),
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
            Scenario::I32And => i32_and::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Clz => i32_clz::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Ctz => i32_ctz::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32DivS => i32_div_s::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32DivU => i32_div_u::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Eq => i32_eq::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Eqz => i32_eqz::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32GeS => i32_ge_s::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32GeU => i32_ge_u::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32GtU => i32_gt_u::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32GtS => i32_gt_s::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32LeU => i32_le_u::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32LeS => i32_le_s::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32LtU => i32_lt_u::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32LtS => i32_lt_s::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Mul => i32_mul::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Ne => i32_ne::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Or => i32_or::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Popcnt => i32_popcnt::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32RemS => i32_rem_s::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32RemU => i32_rem_u::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Rotl => i32_rotl::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Rotr => i32_rotr::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Shl => i32_shl::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32ShrS => i32_shr_s::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32ShrU => i32_shr_u::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Sub => i32_sub::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32WrapI64 => i32_wrap_i64::write_wat_ops(wat, number_of_ops_per_loop_iteration),
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
