use std::{fs::File, io::BufReader, path::PathBuf, time::Duration};

use bench::{parse_nova_data::FileData, prepare::*};
use clap::Parser;
use eyre::bail;
use prover::machine::MachineStatus;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Path to a preimages text file
    #[arg(short, long)]
    data_path: PathBuf,
    /// Path to a machine.wavm.br
    #[arg(short, long)]
    machine_path: PathBuf,
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();
    println!("Setting up machine, this might take some time...");
    let mut machine = prepare_machine(args.data_path.clone(), args.machine_path.clone())?;
    println!("Machine setup complete, now stepping...");
    let start = std::time::Instant::now();
    machine.step_n(1 << 43)?;
    let step_end_time = start.elapsed();
    match machine.get_status() {
        MachineStatus::Errored => {
            bail!("Machine errored => position {}", machine.get_steps())
        }
        MachineStatus::TooFar => {
            bail!("Machine too far => position {}", machine.get_steps())
        }
        MachineStatus::Running => {}
        MachineStatus::Finished => return Ok(()),
    }
    println!(
        "took {:?} to execute fully, and reached took total number of machine steps = {}",
        step_end_time,
        machine.get_steps()
    );
    // let step_sizes = [1, 1 << 10, 1 << 15, 1 << 20, 1 << 26];
    // for step_size in step_sizes {
    //     let mut machine = prepare_machine(args.preimages_path.clone(), args.machine_path.clone())?;
    //     let _ = machine.hash();
    //     let mut hash_times = vec![];
    //     let mut step_times = vec![];
    //     let mut num_iters = 0;
    //     loop {
    //         let start = std::time::Instant::now();
    //         machine.step_n(step_size)?;
    //         let step_end_time = start.elapsed();
    //         step_times.push(step_end_time);
    //         match machine.get_status() {
    //             MachineStatus::Errored => {
    //                 println!("Errored");
    //                 break;
    //                 // bail!("Machine errored => position {}", machine.get_steps())
    //             }
    //             MachineStatus::TooFar => {
    //                 bail!("Machine too far => position {}", machine.get_steps())
    //             }
    //             MachineStatus::Running => {}
    //             MachineStatus::Finished => return Ok(()),
    //         }
    //         let start = std::time::Instant::now();
    //         let _ = machine.hash();
    //         let hash_end_time = start.elapsed();
    //         hash_times.push(hash_end_time);
    //         num_iters += 1;
    //         if num_iters == 100 {
    //             break;
    //         }
    //     }
    //     println!(
    //         "avg hash time {:?}, avg step time {:?}, step size {}, num_iters {}",
    //         average(&hash_times),
    //         average(&step_times),
    //         step_size,
    //         num_iters,
    //     );
    // }
    Ok(())
}

fn average(numbers: &[Duration]) -> Duration {
    let sum: Duration = numbers.iter().sum();
    let sum: u64 = sum.as_nanos().try_into().unwrap();
    Duration::from_nanos(sum / numbers.len() as u64)
}
