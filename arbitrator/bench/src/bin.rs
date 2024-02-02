use std::{path::PathBuf, time::Duration};

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
    let step_sizes = [1 << 15];
    for step_size in step_sizes {
        let mut machine = prepare_machine(args.preimages_path.clone(), args.machine_path.clone())?;
        let machine_hash = machine.hash();
        println!(
            "Starting to execute machine at position {} => {:?}",
            machine.get_steps(),
            hex::encode(&machine_hash)
        );

        println!("Stepping...");
        let mut hash_times = vec![];
        let mut step_times = vec![];
        let mut num_iters = 0;
        loop {
            let start = std::time::Instant::now();
            machine.step_n(step_size)?;
            let step_end_time = start.elapsed();
            step_times.push(step_end_time);
            match machine.get_status() {
                MachineStatus::Errored => {
                    println!("Errored");
                    break;
                    // bail!("Machine errored => position {}", machine.get_steps())
                }
                MachineStatus::TooFar => {
                    bail!("Machine too far => position {}", machine.get_steps())
                }
                MachineStatus::Running => {}
                MachineStatus::Finished => return Ok(()),
            }
            let start = std::time::Instant::now();
            let _ = machine.hash();
            let hash_end_time = start.elapsed();
            hash_times.push(hash_end_time);
            num_iters += 1;
            println!("Num iters {}", num_iters)
        }
        println!(
            "avg hash time {:?}, avg step time {:?}, step size {}, num_iters {} machine hash at position {} => {:?}",
            average(&hash_times),
            average(&step_times),
            step_size,
            num_iters,
            machine.get_steps(),
            hex::encode(&machine_hash)
        );
    }
    Ok(())
}

fn average(numbers: &[Duration]) -> Duration {
    let sum: Duration = numbers.iter().sum();
    let sum: u64 = sum.as_nanos().try_into().unwrap();
    Duration::from_nanos(sum / numbers.len() as u64)
}
