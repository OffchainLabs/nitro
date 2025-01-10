// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::io::Write;

fn rm_identation(s: &mut String) {
    for _ in 0..4 {
        s.pop();
    }
}

pub fn write_wat_ops(
    wat: &mut Vec<u8>,
    number_of_ops_per_loop_iteration: usize,
    table_size: usize,
) {
    for _ in 0..number_of_ops_per_loop_iteration {
        let mut identation = String::from("            ");
        for _ in 0..table_size {
            wat.write_all(format!("{}(block\n", identation).as_bytes())
                .unwrap();
            identation.push_str("    ");
        }
        // it will jump to the end of the first block
        wat.write_all(format!("{}i32.const {}\n", identation, table_size - 1).as_bytes())
            .unwrap();

        let mut br_table = String::from("br_table");
        for i in 0..table_size {
            br_table.push_str(&format!(" {}", i));
        }
        wat.write_all(format!("{}{}\n", identation, br_table).as_bytes())
            .unwrap();

        for _ in 0..table_size {
            rm_identation(&mut identation);
            wat.write_all(format!("{})\n", identation).as_bytes())
                .unwrap();
        }
    }
}
