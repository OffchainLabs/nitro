use std::path::PathBuf;

use clap::Subcommand;

mod compare;
mod promote;
mod record;
mod triage;
mod verify;

#[derive(Debug, Subcommand)]
pub enum FixtureCommand {
    /// Capture truth from a reference node into the fixture's `expected` block.
    Record(RecordArgs),
    /// Run the fixture against arbreth and assert against `expected`.
    Verify(VerifyArgs),
    /// Run the fixture against both nodes and emit a structural diff.
    Compare(CompareArgs),
    /// Move a captured fixture into a committed location.
    Promote(PromoteArgs),
    /// Re-run every fixture in compare mode and classify the result.
    Triage(TriageArgs),
}

#[derive(Debug, clap::Args)]
pub struct RecordArgs {
    /// Path to the fixture JSON to record into.
    pub fixture: PathBuf,
    /// Reference node RPC URL.
    #[arg(long = "nitro-rpc", value_name = "URL")]
    pub nitro_rpc: String,
    /// Optional output path. When unset the fixture is rewritten in-place.
    #[arg(long = "out", value_name = "PATH")]
    pub out: Option<PathBuf>,
}

#[derive(Debug, clap::Args)]
pub struct VerifyArgs {
    /// Path to the fixture JSON to verify.
    pub fixture: PathBuf,
    /// arbreth RPC URL.
    #[arg(long = "rpc", value_name = "URL")]
    pub rpc: String,
}

#[derive(Debug, clap::Args)]
pub struct CompareArgs {
    /// Path to the fixture JSON to compare.
    pub fixture: PathBuf,
    /// Reference node RPC URL.
    #[arg(long = "nitro-rpc", value_name = "URL")]
    pub nitro_rpc: String,
    /// arbreth RPC URL.
    #[arg(long = "arbreth-rpc", value_name = "URL")]
    pub arbreth_rpc: String,
}

#[derive(Debug, clap::Args)]
pub struct PromoteArgs {
    /// Source path under `fixtures/_captured/`.
    pub captured: PathBuf,
    /// Destination path inside the committed fixtures tree.
    pub committed: PathBuf,
    /// Optional arbreth RPC URL; when set the moved fixture is re-verified after promotion.
    #[arg(long = "rpc", value_name = "URL")]
    pub rpc: Option<String>,
}

#[derive(Debug, clap::Args)]
pub struct TriageArgs {
    /// Directory to walk for `*.json` fixtures.
    #[arg(long = "fixtures-dir", value_name = "DIR")]
    pub fixtures_dir: PathBuf,
    /// Reference node RPC URL.
    #[arg(long = "nitro-rpc", value_name = "URL")]
    pub nitro_rpc: String,
    /// arbreth RPC URL.
    #[arg(long = "arbreth-rpc", value_name = "URL")]
    pub arbreth_rpc: String,
}

pub fn run(cmd: FixtureCommand) -> anyhow::Result<()> {
    match cmd {
        FixtureCommand::Record(a) => record::run(a),
        FixtureCommand::Verify(a) => verify::run(a),
        FixtureCommand::Compare(a) => compare::run(a),
        FixtureCommand::Promote(a) => promote::run(a),
        FixtureCommand::Triage(a) => triage::run(a),
    }
}
