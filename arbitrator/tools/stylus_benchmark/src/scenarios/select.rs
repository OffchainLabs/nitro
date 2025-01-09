// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::io::Write;

pub fn write_wat_ops(wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize) {
    wat.write_all(b"            i32.const 10\n").unwrap();
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(b"            i32.const 20\n").unwrap();
        wat.write_all(b"            i32.const 0\n").unwrap();
        wat.write_all(b"            select\n").unwrap();
    }
    wat.write_all(b"            drop\n").unwrap();
}
