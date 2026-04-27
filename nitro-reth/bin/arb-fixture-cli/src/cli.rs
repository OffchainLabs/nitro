use std::path::PathBuf;

use clap::{Parser, Subcommand};

/// Operator CLI for arbreth execution fixtures.
#[derive(Debug, Parser)]
#[command(
    name = "arb-fixture-cli",
    version,
    about = "Record, verify, compare, promote, and triage arbreth fixtures.",
    long_about = None,
)]
pub struct Cli {
    #[command(subcommand)]
    pub command: Command,
}

#[derive(Debug, Subcommand)]
pub enum Command {
    /// Capture truth from Nitro into the fixture's `expected` block.
    Record(RecordArgs),
    /// Run the fixture against arbreth and assert against `expected`.
    Verify(VerifyArgs),
    /// Run the fixture against both nodes and emit a structural diff.
    Compare(CompareArgs),
    /// Move a captured fixture into a committed location.
    Promote(PromoteArgs),
    /// Re-run every fixture in compare mode and classify the result.
    Triage(TriageArgs),
    /// Print the gate inventory from `.plan/gate-inventory.md`.
    ListGates(ListGatesArgs),
}

#[derive(Debug, clap::Args)]
pub struct RecordArgs {
    /// Path to the fixture JSON to record into.
    pub fixture: PathBuf,
    /// Nitro reference RPC URL.
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
    /// Nitro reference RPC URL.
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
    /// Nitro reference RPC URL.
    #[arg(long = "nitro-rpc", value_name = "URL")]
    pub nitro_rpc: String,
    /// arbreth RPC URL.
    #[arg(long = "arbreth-rpc", value_name = "URL")]
    pub arbreth_rpc: String,
}

#[derive(Debug, clap::Args)]
pub struct ListGatesArgs {
    /// Override the default `.plan/gate-inventory.md` path.
    #[arg(long = "inventory", value_name = "PATH")]
    pub inventory: Option<PathBuf>,
}
