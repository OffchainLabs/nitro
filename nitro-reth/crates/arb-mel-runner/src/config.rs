//! Configuration for the message extraction service.
//!
//! Ports `MessageExtractionConfig` and its defaults from nitro's
//! `arbnode/mel/runner/mel.go`. Where nitro validates a freshly-parsed config
//! with a separate `Validate` pass, we instead encode those invariants in the
//! field types ([`ReadMode`] and [`NonZero`]) so an invalid config can't be
//! constructed in the first place.

use std::{fmt, num::NonZero, str::FromStr, time::Duration};

use crate::{MelRunnerError, Result};

/// Which parent-chain blocks the extractor reads.
///
/// Replaces nitro's free-form `read-mode` string. Parsing user input through
/// [`ReadMode::from_str`] is the single point where an unknown mode is rejected;
/// past that boundary the mode is always one of these three.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum ReadMode {
    /// Read parent-chain blocks as soon as they reach the latest tip.
    #[default]
    Latest,
    /// Only read up to the parent chain's safe tip.
    Safe,
    /// Only read up to the parent chain's finalized tip.
    Finalized,
}

impl ReadMode {
    /// The lower-case string form, matching the accepted config values.
    pub fn as_str(&self) -> &'static str {
        match self {
            ReadMode::Latest => "latest",
            ReadMode::Safe => "safe",
            ReadMode::Finalized => "finalized",
        }
    }
}

impl fmt::Display for ReadMode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(self.as_str())
    }
}

impl FromStr for ReadMode {
    type Err = MelRunnerError;

    /// Parses a `read-mode` config value case-insensitively, mirroring the
    /// normalization and check nitro's `(*MessageExtractionConfig).Validate` did.
    fn from_str(s: &str) -> Result<Self> {
        match s.to_lowercase().as_str() {
            "latest" => Ok(ReadMode::Latest),
            "safe" => Ok(ReadMode::Safe),
            "finalized" => Ok(ReadMode::Finalized),
            other => Err(MelRunnerError::Config(format!(
                "inbox reader read-mode is invalid, want: latest or safe or finalized, got: {other}"
            ))),
        }
    }
}

/// Tunables for the [`crate::MessageExtractor`].
///
/// Every field is valid by construction: [`read_mode`](Self::read_mode) can only
/// hold a known [`ReadMode`], and
/// [`log_extraction_status_frequency_blocks`](Self::log_extraction_status_frequency_blocks)
/// can only hold a non-zero count.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MessageExtractionConfig {
    /// Whether the message extraction service is enabled.
    pub enable: bool,
    /// Wait time before retrying after a failed FSM tick.
    pub retry_interval: Duration,
    /// Number of parent-chain blocks to prefetch logs for at once.
    pub blocks_to_prefetch: u64,
    /// Which parent-chain blocks to read.
    pub read_mode: ReadMode,
    /// Max times the FSM may be stuck at the same state before an error is logged.
    pub stall_tolerance: u64,
    /// How often (in blocks processed) to log extraction status.
    pub log_extraction_status_frequency_blocks: NonZero<u64>,
}

impl MessageExtractionConfig {
    /// The config used by tests (a short retry interval), mirroring
    /// `TestMessageExtractionConfig`.
    pub fn test() -> Self {
        Self {
            retry_interval: Duration::from_millis(10),
            ..Self::default()
        }
    }
}

impl Default for MessageExtractionConfig {
    /// Mirrors `DefaultMessageExtractionConfig`.
    fn default() -> Self {
        Self {
            enable: false,
            retry_interval: Duration::from_millis(500),
            // 500 is the eth_getLogs block range limit.
            blocks_to_prefetch: 499,
            read_mode: ReadMode::Latest,
            stall_tolerance: 10,
            log_extraction_status_frequency_blocks: NonZero::new(100).expect("100 is non-zero"),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn valid_read_modes_parse() {
        assert_eq!("latest".parse::<ReadMode>().unwrap(), ReadMode::Latest);
        assert_eq!("safe".parse::<ReadMode>().unwrap(), ReadMode::Safe);
        assert_eq!(
            "finalized".parse::<ReadMode>().unwrap(),
            ReadMode::Finalized
        );
    }

    #[test]
    fn read_mode_parse_is_case_insensitive() {
        assert_eq!("LATEST".parse::<ReadMode>().unwrap(), ReadMode::Latest);
        assert_eq!(
            "Finalized".parse::<ReadMode>().unwrap(),
            ReadMode::Finalized
        );
    }

    #[test]
    fn read_mode_round_trips_through_as_str() {
        for mode in [ReadMode::Latest, ReadMode::Safe, ReadMode::Finalized] {
            assert_eq!(mode.as_str().parse::<ReadMode>().unwrap(), mode);
        }
    }

    #[test]
    fn invalid_read_mode_is_rejected() {
        let err = "unsafe".parse::<ReadMode>().unwrap_err();
        assert!(err.to_string().contains("invalid"));
    }

    #[test]
    fn default_read_mode_is_latest() {
        assert_eq!(
            MessageExtractionConfig::default().read_mode,
            ReadMode::Latest
        );
    }

    #[test]
    fn test_config_uses_short_retry() {
        assert_eq!(
            MessageExtractionConfig::test().retry_interval,
            Duration::from_millis(10)
        );
    }
}
