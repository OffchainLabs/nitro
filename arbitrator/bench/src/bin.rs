use bench::prepare::*;
use eyre::bail;
use prover::machine::MachineStatus;

fn main() -> eyre::Result<()> {
    let mut machine = prepare_machine()?;
    let start = std::time::Instant::now();
    let machine_hash = machine.hash();
    println!(
        "Took {:?}, Machine hash at position {} => {:?}",
        start.elapsed(),
        machine.get_steps(),
        hex::encode(&machine_hash)
    );

    println!("Stepping in increments of 1024 steps at a time, 256 iterations");
    let step_size = 16_384;
    let num_iters = 100;
    for i in 0..num_iters {
        let start = std::time::Instant::now();
        machine.step_n(step_size)?;
        match machine.get_status() {
            MachineStatus::Errored => bail!("Machine errored => position {}", machine.get_steps()),
            MachineStatus::TooFar => bail!("Machine too far => position {}", machine.get_steps()),
            MachineStatus::Running => {}
            MachineStatus::Finished => return Ok(()),
        }
        let machine_hash = machine.hash();
        println!(
            "Took {:?}, step size {}, num_iters {} machine hash at position {} => {:?}",
            start.elapsed(),
            step_size,
            i,
            machine.get_steps(),
            hex::encode(&machine_hash)
        );
    }
    Ok(())
}
