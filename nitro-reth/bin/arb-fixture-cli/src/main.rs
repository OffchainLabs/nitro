mod cli;
mod commands;

use clap::Parser;

use crate::cli::{Cli, Command};

fn main() -> anyhow::Result<()> {
    let args = Cli::parse();
    match args.command {
        Command::Record(a) => commands::record::run(a),
        Command::Verify(a) => commands::verify::run(a),
        Command::Compare(a) => commands::compare::run(a),
        Command::Promote(a) => commands::promote::run(a),
        Command::Triage(a) => commands::triage::run(a),
    }
}
