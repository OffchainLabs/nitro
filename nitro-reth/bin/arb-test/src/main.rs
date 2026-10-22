mod fixture;
mod genesis_capture;
mod sepolia_import;

use clap::{Parser, Subcommand};

#[derive(Debug, Parser)]
#[command(
    name = "arb-test",
    version,
    about = "Unified arbreth testing CLI.",
    long_about = None,
)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Debug, Subcommand)]
enum Command {
    /// Operate on execution fixtures: record, verify, compare, promote, triage.
    #[command(subcommand)]
    Fixture(fixture::FixtureCommand),

    /// Capture an Arbitrum genesis state into a geth-format JSON file.
    GenesisCapture(genesis_capture::GenesisCaptureArgs),

    /// Sepolia archive helpers.
    #[command(subcommand)]
    SepoliaImport(sepolia_import::SepoliaImportCommand),
}

fn main() -> anyhow::Result<()> {
    let args = Cli::parse();
    match args.command {
        Command::Fixture(cmd) => fixture::run(cmd),
        Command::GenesisCapture(a) => genesis_capture::run(a),
        Command::SepoliaImport(cmd) => sepolia_import::run(cmd),
    }
}
