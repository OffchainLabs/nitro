use clap::Args;

/// Arbitrum rollup-specific CLI arguments.
#[derive(Debug, Clone, Default, Args)]
#[command(next_help_heading = "Rollup")]
pub struct RollupArgs {
    /// Enable sequencer mode.
    #[arg(long = "rollup.sequencer", default_value_t = false)]
    pub sequencer: bool,
}
