// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::scenarios::{
    call, call_indirect, global_get, global_set, i32_eqz, i32_popcnt, i32_wrap_i64, if_op, i32_ctz, i32_clz,
    instruction_with_2_args_1_return, select,
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
            Scenario::I32Add => {}
            Scenario::I32And => {}
            Scenario::I32Clz => i32_clz::write_specific_wat_beginning(wat),
            Scenario::I32Ctz => i32_ctz::write_specific_wat_beginning(wat),
            Scenario::I32DivS => {}
            Scenario::I32DivU => {}
            Scenario::I32Eq => {}
            Scenario::I32Eqz => i32_eqz::write_specific_wat_beginning(wat),
            Scenario::I32GeS => {}
            Scenario::I32GeU => {}
            Scenario::I32GtU => {}
            Scenario::I32GtS => {}
            Scenario::I32LeU => {}
            Scenario::I32LeS => {}
            Scenario::I32LtU => {}
            Scenario::I32LtS => {}
            Scenario::I32Mul => {}
            Scenario::I32Ne => {}
            Scenario::I32Or => {}
            Scenario::I32Popcnt => i32_popcnt::write_specific_wat_beginning(wat),
            Scenario::I32RemS => {}
            Scenario::I32RemU => {}
            Scenario::I32Rotl => {}
            Scenario::I32Rotr => {}
            Scenario::I32Shl => {}
            Scenario::I32ShrS => {}
            Scenario::I32ShrU => {}
            Scenario::I32Sub => {}
            Scenario::I32WrapI64 => i32_wrap_i64::write_specific_wat_beginning(wat),
            Scenario::I32Xor => {}
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
            Scenario::I32Add => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.add",
                0,
                1,
            ),
            Scenario::I32And => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.and",
                0,
                1,
            ),
            Scenario::I32Clz => i32_clz::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32Ctz => i32_ctz::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32DivS => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.div_s",
                1,
                1,
            ),
            Scenario::I32DivU => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.div_u",
                1,
                1,
            ),
            Scenario::I32Eq => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.eq",
                0,
                1,
            ),
            Scenario::I32Eqz => i32_eqz::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32GeS => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.ge_s",
                0,
                1,
            ),
            Scenario::I32GeU => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.ge_u",
                0,
                1,
            ),
            Scenario::I32GtU => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.gt_u",
                0,
                1,
            ),
            Scenario::I32GtS => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.gt_s",
                0,
                1,
            ),
            Scenario::I32LeU => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.le_u",
                0,
                1,
            ),
            Scenario::I32LeS => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.le_s",
                0,
                1,
            ),
            Scenario::I32LtU => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.lt_u",
                0,
                1,
            ),
            Scenario::I32LtS => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.lt_s",
                0,
                1,
            ),
            Scenario::I32Mul => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.mul",
                0,
                1,
            ),
            Scenario::I32Ne => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.ne",
                0,
                1,
            ),
            Scenario::I32Or => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.or",
                0,
                1,
            ),
            Scenario::I32Popcnt => i32_popcnt::write_wat_ops(wat, number_of_ops_per_loop_iteration),
            Scenario::I32RemS => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.rem_s",
                1,
                1,
            ),
            Scenario::I32RemU => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.rem_u",
                1,
                1,
            ),
            Scenario::I32Rotl => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.rotl",
                11231,
                1,
            ),
            Scenario::I32Rotr => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.rotr",
                11231,
                1,
            ),
            Scenario::I32Shl => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.shl",
                11231,
                1,
            ),
            Scenario::I32ShrS => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.shr_s",
                11231,
                1,
            ),
            Scenario::I32ShrU => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.shr_u",
                11231,
                1,
            ),
            Scenario::I32Sub => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.sub",
                11231,
                1,
            ),
            Scenario::I32WrapI64 => {
                i32_wrap_i64::write_wat_ops(wat, number_of_ops_per_loop_iteration)
            }
            Scenario::I32Xor => instruction_with_2_args_1_return::write_wat_ops(
                wat,
                number_of_ops_per_loop_iteration,
                "i32.xor",
                11231,
                13242,
            ),
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
