// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use eyre::{Result, WrapErr};
use prover::{machine::GlobalState, utils::file_bytes, Machine};
use std::{collections::HashMap, fmt::Display, path::PathBuf, sync::Arc};
use structopt::StructOpt;

#[derive(StructOpt)]
#[structopt(name = "module-roots")]
struct Opts {
    #[structopt(long)]
    binary: PathBuf,
    #[structopt(long)]
    stylus_modules: Vec<PathBuf>,
}

fn main() -> Result<()> {
    let mut opts = Opts::from_args();

    macro_rules! relocate {
        ($file:expr) => {
            let mut path = PathBuf::from("../../../");
            path.push(&$file);
            *$file = path;
        };
    }
    relocate!(&mut opts.binary);

    let mut mach = Machine::from_paths(
        &[],
        &opts.binary,
        true,
        true,
        true,
        true,
        true,
        GlobalState::default(),
        HashMap::default(),
        Arc::new(|_, _| panic!("tried to read preimage")),
    )?;

    let mut stylus = vec![];
    for module in &mut opts.stylus_modules {
        relocate!(module);
        let error = || format!("failed to read module at {}", module.to_string_lossy());
        let wasm = file_bytes(&module).wrap_err_with(error)?;
        let hash = mach.add_program(&wasm, 1, true).wrap_err_with(error)?;
        let name = module.file_stem().unwrap().to_string_lossy();
        stylus.push((name.to_owned(), hash));
        println!("{} {}", name, hash);
    }

    let mut segment = 0;
    for (name, root) in stylus {
        println!("    (data (i32.const 0x{:03x})", segment);
        println!("        \"{}\") ;; {}", pairs(root), name);
        segment += 32;
    }
    Ok(())
}

fn pairs<D: Display>(text: D) -> String {
    let mut out = String::new();
    let text = format!("{text}");
    let mut chars = text.chars();
    while let Some(a) = chars.next() {
        let b = chars.next().unwrap();
        out += &format!("\\{a}{b}")
    }
    out
}
