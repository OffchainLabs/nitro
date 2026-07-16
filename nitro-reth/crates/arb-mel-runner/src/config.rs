//! Configuration for the message extraction service.
//!
//! Ports `MessageExtractionConfig` and its defaults/validation from nitro's
//! `arbnode/mel/runner/mel.go`.

use std::time::Duration;

use crate::{MelError, Result};

/// Tunables for the [`crate::MessageExtractor`].
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MessageExtractionConfig {
    /// Whether the message extraction service is enabled.
    pub enable: bool,
    /// Wait time before retrying after a failed FSM tick.
    pub retry_interval: Duration,
    /// Number of parent-chain blocks to prefetch logs for at once.
    pub blocks_to_prefetch: u64,
    /// Which parent-chain blocks to read: `latest`, `safe`, or `finalized`.
    pub read_mode: String,
    /// Max times the FSM may be stuck at the same state before an error is logged.
    pub stall_tolerance: u64,
    /// How often (in blocks processed) to log extraction status.
    pub log_extraction_status_frequency_blocks: u64,
}

impl MessageExtractionConfig {
    /// Normalizes and validates the config, mirroring `(*MessageExtractionConfig).Validate`.
    ///
    /// Lower-cases `read_mode` in place, requires it to be one of
    /// `latest`/`safe`/`finalized`, and requires a non-zero log frequency.
    pub fn validate(&mut self) -> Result<()> {
        self.read_mode = self.read_mode.to_lowercase();
        if !matches!(self.read_mode.as_str(), "latest" | "safe" | "finalized") {
            return Err(MelError::Config(format!(
                "inbox reader read-mode is invalid, want: latest or safe or finalized, got: {}",
                self.read_mode
            )));
        }
        if self.log_extraction_status_frequency_blocks == 0 {
            return Err(MelError::Config(
                "log-extraction-status-frequency-blocks must be greater than 0".to_string(),
            ));
        }
        Ok(())
    }

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
            read_mode: "latest".to_string(),
            stall_tolerance: 10,
            log_extraction_status_frequency_blocks: 100,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn valid_read_modes_pass() {
        for mode in ["latest", "safe", "finalized"] {
            let mut cfg = MessageExtractionConfig {
                read_mode: mode.to_string(),
                ..Default::default()
            };
            assert!(cfg.validate().is_ok());
        }
    }

    #[test]
    fn read_mode_is_normalized_to_lowercase() {
        let mut cfg = MessageExtractionConfig {
            read_mode: "LATEST".to_string(),
            ..Default::default()
        };
        assert!(cfg.validate().is_ok());
        assert_eq!(cfg.read_mode, "latest");
    }

    #[test]
    fn invalid_read_mode_is_rejected() {
        let mut cfg = MessageExtractionConfig {
            read_mode: "unsafe".to_string(),
            ..Default::default()
        };
        let err = cfg.validate().unwrap_err();
        assert!(err.to_string().contains("invalid"));
    }

    #[test]
    fn zero_log_frequency_is_rejected() {
        let mut cfg = MessageExtractionConfig {
            log_extraction_status_frequency_blocks: 0,
            ..Default::default()
        };
        let err = cfg.validate().unwrap_err();
        assert!(err.to_string().contains("must be greater than 0"));
    }

    #[test]
    fn test_config_uses_short_retry() {
        assert_eq!(
            MessageExtractionConfig::test().retry_interval,
            Duration::from_millis(10)
        );
    }
}
