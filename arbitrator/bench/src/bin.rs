use std::{path::PathBuf, time::Duration};

use clap::Parser;
use eyre::bail;

#[cfg(feature = "cpuprof")]
use gperftools::profiler::PROFILER;

#[cfg(feature = "heapprof")]
use gperftools::heap_profiler::HEAP_PROFILER;

#[cfg(feature = "counters")]
use prover::{machine, memory, merkle};

use prover::machine::MachineStatus;
use prover::prepare::prepare_machine;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Path to a preimages json file
    #[arg(short, long)]
    json_inputs: PathBuf,

    /// Path to a machine.wavm.br
    #[arg(short, long)]
    binary: PathBuf,
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();
    let step_sizes = [1, 1 << 10, 1 << 15, 1 << 20, 1 << 24, 1 << 26, 1 << 28];

    println!("Running benchmark with always merkleize feature on");
    for step_size in step_sizes {
        let mut machine = prepare_machine(args.json_inputs.clone(), args.binary.clone())?;
        let _ = machine.hash();
        let mut hash_times = vec![];
        let mut step_times = vec![];
        let mut num_iters = 0;

        #[cfg(feature = "cpuprof")]
        PROFILER
            .lock()
            .unwrap()
            .start(format!("./target/bench-{}.prof", step_size))
            .unwrap();

        #[cfg(feature = "heapprof")]
        HEAP_PROFILER
            .lock()
            .unwrap()
            .start(format!("./target/bench-{}.hprof", step_size))
            .unwrap();

        #[cfg(feature = "counters")]
        {
            machine::reset_counters();
            memory::reset_counters();
            merkle::reset_counters();
        }
        let total = std::time::Instant::now();
        loop {
            let start = std::time::Instant::now();
            machine.step_n(step_size)?;
            let step_end_time = start.elapsed();
            step_times.push(step_end_time);
            match machine.get_status() {
                MachineStatus::Errored => {
                    println!("Errored");
                    break;
                }
                MachineStatus::TooFar => {
                    bail!("Machine too far => position {}", machine.get_steps())
                }
                MachineStatus::Running => {}
                MachineStatus::Finished => {
                    break;
                }
            }
            let start = std::time::Instant::now();
            let _ = machine.hash();
            let hash_end_time = start.elapsed();
            hash_times.push(hash_end_time);
            num_iters += 1;
            if num_iters == 200 {
                break;
            }
        }

        #[cfg(feature = "cpuprof")]
        PROFILER.lock().unwrap().stop().unwrap();

        #[cfg(feature = "heapprof")]
        HEAP_PROFILER.lock().unwrap().stop().unwrap();

        let total_end_time = total.elapsed();
        println!(
            "avg hash time {:>11?}, avg step time {:>12?}, step size {:>9}, num_iters {:>3}, total time {:>12?}",
            average(&hash_times),
            average(&step_times),
            step_size,
            num_iters,
            total_end_time,
        );
        #[cfg(feature = "counters")]
        {
            machine::print_counters();
            memory::print_counters();
            merkle::print_counters();
        }
    }
    Ok(())
}

fn average(numbers: &[Duration]) -> Duration {
    let sum: Duration = numbers.iter().sum();
    let sum: u64 = sum.as_nanos().try_into().unwrap();
    Duration::from_nanos(sum / numbers.len() as u64)
}
