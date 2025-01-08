// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::io::Write;

pub fn write_wat_ops(wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize, instruction: &str, arg: usize) {
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(format!("            i32.const {}\n", arg).as_bytes()).unwrap();
        wat.write_all(format!("            {}\n", instruction).as_bytes()).unwrap();
        wat.write_all(b"            drop\n").unwrap();
    }
}
