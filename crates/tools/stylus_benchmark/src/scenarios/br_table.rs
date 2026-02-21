// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::io::Write;

fn rm_indentation(s: &mut String) {
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
        let mut indentation = String::from("            ");
        for _ in 0..table_size {
            wat.write_all(format!("{}(block\n", indentation).as_bytes())
                .unwrap();
            indentation.push_str("    ");
        }
        // it will jump to the end of the first block
        wat.write_all(format!("{}i32.const {}\n", indentation, table_size - 1).as_bytes())
            .unwrap();

        let mut br_table = String::from("br_table");
        for i in 0..table_size {
            br_table.push_str(&format!(" {}", i));
        }
        wat.write_all(format!("{}{}\n", indentation, br_table).as_bytes())
            .unwrap();

        for _ in 0..table_size {
            rm_indentation(&mut indentation);
            wat.write_all(format!("{})\n", indentation).as_bytes())
                .unwrap();
        }
    }
}
