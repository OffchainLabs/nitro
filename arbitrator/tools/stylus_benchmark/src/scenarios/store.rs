// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::scenarios::data_type::{DataType, Rand};
use std::io::Write;

pub fn write_wat_ops(
    wat: &mut Vec<u8>,
    number_of_ops_per_loop_iteration: usize,
    data_type: DataType,
) {
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(format!("            {}.const 0\n", data_type).as_bytes())
            .unwrap();
        wat.write_all(format!("            {}.const {}\n", data_type, data_type.gen()).as_bytes())
            .unwrap();
        wat.write_all(format!("            {}.store\n", data_type).as_bytes())
            .unwrap();
    }
}
