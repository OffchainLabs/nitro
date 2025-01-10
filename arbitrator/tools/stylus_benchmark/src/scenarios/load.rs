// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::scenarios::data_type::DataType;
use std::io::Write;

pub fn write_wat_ops(
    wat: &mut Vec<u8>,
    number_of_ops_per_loop_iteration: usize,
    data_type: DataType,
) {
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(format!("            i32.const 0\n").as_bytes())
            .unwrap();
        wat.write_all(format!("            {}.load\n", data_type).as_bytes())
            .unwrap();
        wat.write_all(b"            drop\n").unwrap();
    }
}
