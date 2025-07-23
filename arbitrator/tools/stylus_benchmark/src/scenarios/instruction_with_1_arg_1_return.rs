// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::scenarios::data_type::{DataType, Rand};
use std::io::Write;

pub fn write_wat_ops(
    wat: &mut Vec<u8>,
    number_of_ops_per_loop_iteration: usize,
    data_type: DataType,
    instruction: &str,
) {
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(format!("            {}.const {}\n", data_type, data_type.gen()).as_bytes())
            .unwrap();
        wat.write_all(format!("            {}.{}\n", data_type, instruction).as_bytes())
            .unwrap();
        wat.write_all(b"            drop\n").unwrap();
    }
}
