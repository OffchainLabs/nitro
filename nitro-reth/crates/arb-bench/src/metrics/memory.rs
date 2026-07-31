use sysinfo::{Pid, ProcessRefreshKind, RefreshKind, System};

/// Cached system handle for cheap repeated RSS reads.
pub struct RssMonitor {
    sys: System,
    pid: Pid,
    peak: u64,
}

impl RssMonitor {
    /// Track this process's RSS.
    pub fn new() -> Self {
        Self::for_pid(std::process::id())
    }

    /// Track another process's RSS, e.g. a spawned node whose memory — not the
    /// harness's — is the thing under measurement.
    pub fn for_pid(pid: u32) -> Self {
        let pid = Pid::from_u32(pid);
        let sys = System::new_with_specifics(
            RefreshKind::new().with_processes(ProcessRefreshKind::new().with_memory()),
        );
        Self { sys, pid, peak: 0 }
    }

    /// Refresh and return current RSS in bytes. Updates the running peak.
    pub fn current_rss(&mut self) -> u64 {
        self.sys.refresh_processes_specifics(
            sysinfo::ProcessesToUpdate::Some(&[self.pid]),
            true,
            ProcessRefreshKind::new().with_memory(),
        );
        let rss = self.sys.process(self.pid).map(|p| p.memory()).unwrap_or(0);
        if rss > self.peak {
            self.peak = rss;
        }
        rss
    }

    pub fn peak_rss(&self) -> u64 {
        self.peak
    }
}

impl Default for RssMonitor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rss_returns_nonzero_for_self() {
        let mut m = RssMonitor::new();
        let r = m.current_rss();
        assert!(r > 0, "expected nonzero RSS for current process");
        assert!(m.peak_rss() >= r);
    }

    /// The subprocess runner reports the node's memory, not the harness's.
    #[test]
    fn rss_tracks_another_process() {
        let mut child = std::process::Command::new("sleep")
            .arg("30")
            .spawn()
            .expect("spawn sleep");
        let mut m = RssMonitor::for_pid(child.id());
        let r = m.current_rss();
        let _ = child.kill();
        let _ = child.wait();
        assert!(r > 0, "expected nonzero RSS for the child process");
    }
}
