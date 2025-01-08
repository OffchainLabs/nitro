// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::io::Write;

pub fn write_specific_wat_beginning(_: &mut Vec<u8>) {}

pub fn write_wat_ops(wat: &mut Vec<u8>, number_of_ops_per_loop_iteration: usize) {
    wat.write_all(b"            i32.const 1\n").unwrap();
    for _ in 0..number_of_ops_per_loop_iteration {
        wat.write_all(b"            i32.const 1\n").unwrap();
        wat.write_all(b"            i32.rem_u\n").unwrap();
    }
    wat.write_all(b"            drop\n").unwrap();
}
