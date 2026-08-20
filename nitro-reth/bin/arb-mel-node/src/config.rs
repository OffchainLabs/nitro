use std::{num::NonZero, path::PathBuf, time::Duration};

use alloy_primitives::Address;
use arb_mel_runner::{MessageExtractionConfig, ReadMode};
use clap::Parser;

/// CLI arguments for the MEL runner binary.
#[derive(Debug, Parser)]
pub struct MelNodeConfig {
    /// Directory for the consensus database.
    #[arg(long)]
    pub datadir: PathBuf,

    /// Fsync every DB commit to disk before the write is marked as completed.
    #[arg(long = "db.sync-mode", default_value_t = false)]
    pub db_sync_mode: bool,

    /// Parent-chain RPC endpoint.
    #[arg(long)]
    pub parent_chain_url: String,

    /// Bridge contract address (delayed-message posting target).
    #[arg(long)]
    pub bridge: Address,

    /// Inbox contract address.
    #[arg(long)]
    pub inbox: Address,

    /// Sequencer-inbox contract address (batch posting target).
    #[arg(long)]
    pub sequencer_inbox: Address,

    /// Rollup contract address.
    #[arg(long)]
    pub rollup: Address,

    /// Parent-chain block the rollup was deployed at (seeds the initial MEL state).
    #[arg(long)]
    pub deployed_at: u64,

    /// Wait time before retrying after a failed FSM tick.
    #[arg(long, default_value = "500ms", value_parser = humantime::parse_duration)]
    pub retry_interval: Duration,

    /// Number of parent-chain blocks to prefetch logs for at once.
    #[arg(long, default_value_t = 499)]
    pub blocks_to_prefetch: u64,

    /// Which parent-chain blocks to read: latest, safe, or finalized.
    #[arg(long, default_value = "latest")]
    pub read_mode: ReadMode,

    /// Max times the FSM may be stuck at the same state before an error is logged.
    #[arg(long, default_value_t = 10)]
    pub stall_tolerance: u64,

    /// How often (in blocks processed) to log extraction status.
    #[arg(long, default_value = "100")]
    pub log_extraction_status_frequency_blocks: NonZero<u64>,
}

impl MelNodeConfig {
    /// The runner tunables (defaults mirror nitro's `DefaultMessageExtractionConfig`).
    /// `enable` is hardcoded: running this binary is enabling.
    pub fn extraction_config(&self) -> MessageExtractionConfig {
        MessageExtractionConfig {
            enable: true,
            retry_interval: self.retry_interval,
            blocks_to_prefetch: self.blocks_to_prefetch,
            read_mode: self.read_mode,
            stall_tolerance: self.stall_tolerance,
            log_extraction_status_frequency_blocks: self.log_extraction_status_frequency_blocks,
        }
    }
}
