// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::scenarios::data_type::{DataType, Rand};
use std::io::Write;

pub fn write_wat_ops(wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize) {
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(format!("            i32.const {}\n", DataType::I32.gen()).as_bytes())
            .unwrap();
        wat.write_all(format!("            i32.const {}\n", DataType::I32.gen()).as_bytes())
            .unwrap();
        wat.write_all(b"            i32.const 0\n").unwrap();
        wat.write_all(b"            select\n").unwrap();
        wat.write_all(b"            drop\n").unwrap();
    }
}
