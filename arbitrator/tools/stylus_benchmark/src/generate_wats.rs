// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::fs::File;
use std::io::Write;
use std::path::PathBuf;

fn generate_add_i32_wat(mut out_path: PathBuf) -> eyre::Result<()> {
    let number_of_ops = 20_000_000;

    out_path.push("add_i32.wat");
    println!(
        "Generating {:?}, number_of_ops: {:?}",
        out_path, number_of_ops
    );

    let mut file = File::create(out_path)?;

    file.write_all(b"(module\n")?;
    file.write_all(b"    (import \"debug\" \"toggle_benchmark\" (func $toggle_benchmark))\n")?;
    file.write_all(b"    (memory (export \"memory\") 0 0)\n")?;
    file.write_all(b"    (func (export \"user_entrypoint\") (param i32) (result i32)\n")?;

    file.write_all(b"        call $toggle_benchmark\n")?;

    file.write_all(b"        i32.const 1\n")?;
    for _ in 0..number_of_ops {
        file.write_all(b"        i32.const 1\n")?;
        file.write_all(b"        i32.add\n")?;
    }

    file.write_all(b"        call $toggle_benchmark\n")?;

    file.write_all(b"        drop\n")?;
    file.write_all(b"        i32.const 0)\n")?;
    file.write_all(b")")?;

    Ok(())
}

pub fn generate_wats(out_path: PathBuf) -> eyre::Result<()> {
    return generate_add_i32_wat(out_path);
}
