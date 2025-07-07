use std::path::PathBuf;

use clap::Parser;
use eyre::bail;

use prover::machine::MachineStatus;

mod prepare;

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    /// Path to a preimages json file
    #[arg(short, long)]
    json_inputs: PathBuf,

    /// Path to a mel_machine.wavm.br
    #[arg(short, long)]
    mel_machine_binary: PathBuf,
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();

    let mut machine =
        prepare::build_machine(args.json_inputs.clone(), args.mel_machine_binary.clone())?;
    let _ = machine.hash();

    let mut total = 0;
    loop {
        machine.step_n(1)?;
        total += 1;
        if total % 10_000 == 0 {
            println!("Total steps: {}", total);
        }
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
                println!("Machine finished!");
                break;
            }
        }
    }
    let gs = machine.get_global_state();
    println!(
        "Global state final mel state root: {}",
        hex::encode(gs.bytes32_vals[2])
    );
    Ok(())
}
