// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::io::Write;

pub fn write_specific_wat_beginning(wat: &mut Vec<u8>) {
    wat.write_all(b"        (global $var (mut i32) (i32.const 0))\n")
        .unwrap();
}

pub fn write_wat_ops(wat: &mut Vec<u8>, number_of_loop_iterations: usize) {
    for _ in 0..number_of_loop_iterations {
        wat.write_all(b"            i32.const 10\n").unwrap();
        wat.write_all(b"            global.set $var\n").unwrap();
    }
}
