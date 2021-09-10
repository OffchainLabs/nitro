mod binary;
mod lir;
mod machine;
mod memory;
mod merkle;
mod utils;
mod value;

use crate::machine::Machine;
use eyre::Result;
use serde::Serialize;
use std::{collections::HashSet, fs::File, io::Read, path::PathBuf, process};
use structopt::StructOpt;

#[derive(StructOpt)]
#[structopt(name = "arbitrator-prover")]
struct Opts {
    binary: PathBuf,
    #[structopt(short, long)]
    output: Option<PathBuf>,
}

#[derive(Serialize)]
struct ProofInfo {
    before: String,
    proof: String,
    after: String,
}

fn main() -> Result<()> {
    let opts = Opts::from_args();
    let mut f = File::open(opts.binary)?;
    let mut buf = Vec::new();
    f.read_to_end(&mut buf)?;

    let bin = match binary::parse(&buf) {
        Ok(bin) => bin,
        Err(err) => {
            eprintln!("Parsing error:");
            for (input, kind) in err.errors {
                eprintln!("Got {:?} while parsing {}", kind, hex::encode(input));
            }
            process::exit(1);
        }
    };

    let out = opts.output.map(File::create).transpose()?;

    let mut proofs = Vec::new();
    let mut mach = Machine::from_binary(bin)?;
    println!("Starting machine hash: {}", mach.hash());

    let mut seen_states = HashSet::new();
    while !mach.is_halted() {
        let before = mach.hash();
        if !seen_states.insert(before) {
            break;
        }
        println!("Machine stack: {:?}", mach.get_data_stack());
        println!(
            "Generating proof #{} of opcode {:?}",
            proofs.len(),
            mach.get_next_instruction().unwrap().opcode
        );
        let proof = mach.serialize_proof();
        mach.step();
        let after = mach.hash();
        proofs.push(ProofInfo {
            before: before.to_string(),
            proof: hex::encode(proof),
            after: after.to_string(),
        });
    }

    println!("End machine hash: {}", mach.hash());
    if let Some(out) = out {
        serde_json::to_writer_pretty(out, &proofs)?;
    }

    Ok(())
}
