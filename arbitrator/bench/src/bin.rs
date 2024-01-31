use std::path::PathBuf;

use bench::prepare::*;
use clap::Parser;
use eyre::bail;
use prover::machine::MachineStatus;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Path to a preimages text file
    #[arg(short, long)]
    preimages_path: PathBuf,

    /// Path to a machine.wavm.br
    #[arg(short, long)]
    machine_path: PathBuf,
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();
    let step_sizes = [1 << 24, 1 << 26];
    for step_size in step_sizes {
        let mut machine = prepare_machine(args.preimages_path.clone(), args.machine_path.clone())?;
        let machine_hash = machine.hash();
        println!(
            "Starting to execute machine at position {} => {:?}",
            machine.get_steps(),
            hex::encode(&machine_hash)
        );

        println!("Stepping...");
        let num_iters = 10;
        for i in 0..num_iters {
            let start = std::time::Instant::now();
            machine.step_n(step_size)?;
            let step_end_time = start.elapsed();
            match machine.get_status() {
                MachineStatus::Errored => {
                    let start = std::time::Instant::now();
                    let machine_hash = machine.hash();
                    let hash_end_time = start.elapsed();
                    println!(
                    "hash time {:?}, step time {:?}, step size {}, num_iters {} machine hash at position {} => {:?}",
                    hash_end_time,
                    step_end_time,
                    step_size,
                    i,
                    machine.get_steps(),
                    hex::encode(&machine_hash)
                );
                    bail!("Machine errored => position {}", machine.get_steps())
                }
                MachineStatus::TooFar => {
                    bail!("Machine too far => position {}", machine.get_steps())
                }
                MachineStatus::Running => {}
                MachineStatus::Finished => return Ok(()),
            }
            let start = std::time::Instant::now();
            let machine_hash = machine.hash();
            let hash_end_time = start.elapsed();
            println!(
                "hash time {:?}, step time {:?}, step size {}, num_iters {} machine hash at position {} => {:?}",
                hash_end_time,
                step_end_time,
                step_size,
                i,
                machine.get_steps(),
                hex::encode(&machine_hash)
            );
        }
        println!("=================");
    }
    Ok(())
}
