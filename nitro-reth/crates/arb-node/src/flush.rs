//! Flush scheduling for the block producer.

pub const DEFAULT_FLUSH_INTERVAL: u64 = 128;

/// Fixed-interval flush scheduler with an EMA of commit latency tracked for
/// observability. The interval is set at construction and does not change.
pub struct FlushScheduler {
    interval: u64,
    ema_commit_latency_ms: u64,
}

impl FlushScheduler {
    pub fn new(interval: u64) -> Self {
        Self {
            interval,
            ema_commit_latency_ms: 0,
        }
    }

    pub fn should_flush(&self, since_last: u64) -> bool {
        since_last >= self.interval
    }

    pub fn observe(&mut self, commit_latency_ms: u64) {
        self.ema_commit_latency_ms = (self.ema_commit_latency_ms * 7 + commit_latency_ms * 3) / 10;
    }

    pub fn current_interval(&self) -> u64 {
        self.interval
    }
}

#[cfg(target_os = "linux")]
pub(crate) fn read_dirty_pages_mb() -> Option<u64> {
    let content = std::fs::read_to_string("/proc/meminfo").ok()?;
    for line in content.lines() {
        if let Some(rest) = line.strip_prefix("Dirty:") {
            let kb: u64 = rest.trim().trim_end_matches(" kB").trim().parse().ok()?;
            return Some(kb / 1024);
        }
    }
    None
}

#[cfg(not(target_os = "linux"))]
pub(crate) fn read_dirty_pages_mb() -> Option<u64> {
    None
}
