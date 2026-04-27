use clap::{Parser, Subcommand};

mod commands;

#[derive(Parser)]
#[command(name = "xtask", about = "Arbreth workspace task runner")]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Subcommand)]
enum Command {
    /// Assert each ArbOS version constant has a baseline fixture.
    CheckCoverage,
    /// Regenerate `.plan/gate-inventory.md` from current source.
    RegenInventory,
    /// Print a Markdown table of fixture counts grouped by category.
    CountFixtures,
}

fn main() -> std::process::ExitCode {
    let cli = Cli::parse();
    let result = match cli.command {
        Command::CheckCoverage => commands::check_coverage::run(),
        Command::RegenInventory => commands::regen_inventory::run(),
        Command::CountFixtures => commands::count_fixtures::run(),
    };
    match result {
        Ok(code) => code,
        Err(err) => {
            eprintln!("error: {err:#}");
            std::process::ExitCode::from(2)
        }
    }
}
