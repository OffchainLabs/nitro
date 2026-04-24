pub mod clock;
pub mod memory;
pub mod rolling;

use serde::{Deserialize, Serialize};

use rolling::WindowMetric;

/// Per-block measurement captured by the runner.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockMetric {
    pub block_number: u64,
    pub wall_clock_ns: u64,
    pub cpu_ns: u64,
    pub gas_used: u64,
    pub tx_count: usize,
    pub success_count: usize,
    pub rss_bytes: u64,
}

impl BlockMetric {
    pub fn gas_per_sec(&self) -> f64 {
        if self.wall_clock_ns == 0 {
            return 0.0;
        }
        (self.gas_used as f64) * 1_000_000_000.0 / (self.wall_clock_ns as f64)
    }
}

/// Aggregate result of one run of one workload.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunResult {
    pub manifest_name: String,
    pub blocks: Vec<BlockMetric>,
    pub windows: Vec<WindowMetric>,
    pub summary: SummaryMetrics,
    pub host: HostInfo,
}

/// Summary statistics across an entire run.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct SummaryMetrics {
    pub block_count: usize,
    pub total_wall_clock_ns: u64,
    pub total_cpu_ns: u64,
    pub total_gas: u64,
    pub gas_per_sec_mean: f64,
    pub gas_per_sec_median: f64,
    pub wall_clock_ns_p50: u64,
    pub wall_clock_ns_p95: u64,
    pub wall_clock_ns_p99: u64,
    pub peak_rss_bytes: u64,
    pub rss_growth_per_10k_blocks_bytes: i64,
    pub monotonic_slowdown_detected: bool,
    pub worst_window_gas_per_sec: f64,
    pub steady_state_gas_per_sec: f64,
}

impl SummaryMetrics {
    pub fn from_blocks(blocks: &[BlockMetric], windows: &[WindowMetric]) -> Self {
        if blocks.is_empty() {
            return Self::default();
        }
        let total_wall_clock_ns: u64 = blocks.iter().map(|b| b.wall_clock_ns).sum();
        let total_cpu_ns: u64 = blocks.iter().map(|b| b.cpu_ns).sum();
        let total_gas: u64 = blocks.iter().map(|b| b.gas_used).sum();

        let mut wcs: Vec<u64> = blocks.iter().map(|b| b.wall_clock_ns).collect();
        wcs.sort_unstable();
        let p50 = percentile_u64(&wcs, 50.0);
        let p95 = percentile_u64(&wcs, 95.0);
        let p99 = percentile_u64(&wcs, 99.0);

        let gas_per_sec_mean = if total_wall_clock_ns > 0 {
            (total_gas as f64) * 1_000_000_000.0 / (total_wall_clock_ns as f64)
        } else {
            0.0
        };
        let mut per_block_gps: Vec<f64> = blocks.iter().map(BlockMetric::gas_per_sec).collect();
        per_block_gps.sort_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
        let gas_per_sec_median = percentile_f64(&per_block_gps, 50.0);

        let peak_rss_bytes = blocks.iter().map(|b| b.rss_bytes).max().unwrap_or(0);

        let (rss_growth_per_10k, monotonic_slowdown_detected) = analyze_trend(blocks);
        let (worst_window_gas_per_sec, steady_state_gas_per_sec) = window_extremes(windows);

        Self {
            block_count: blocks.len(),
            total_wall_clock_ns,
            total_cpu_ns,
            total_gas,
            gas_per_sec_mean,
            gas_per_sec_median,
            wall_clock_ns_p50: p50,
            wall_clock_ns_p95: p95,
            wall_clock_ns_p99: p99,
            peak_rss_bytes,
            rss_growth_per_10k_blocks_bytes: rss_growth_per_10k,
            monotonic_slowdown_detected,
            worst_window_gas_per_sec,
            steady_state_gas_per_sec,
        }
    }
}

fn percentile_u64(sorted: &[u64], p: f64) -> u64 {
    if sorted.is_empty() {
        return 0;
    }
    let idx = ((p / 100.0) * (sorted.len() as f64 - 1.0)).round() as usize;
    sorted[idx.min(sorted.len() - 1)]
}

fn percentile_f64(sorted: &[f64], p: f64) -> f64 {
    if sorted.is_empty() {
        return 0.0;
    }
    let idx = ((p / 100.0) * (sorted.len() as f64 - 1.0)).round() as usize;
    sorted[idx.min(sorted.len() - 1)]
}

/// Detect monotonic slowdown + RSS growth.
fn analyze_trend(blocks: &[BlockMetric]) -> (i64, bool) {
    if blocks.len() < 30 {
        return (0, false);
    }
    let third = blocks.len() / 3;
    let mut first: Vec<f64> = blocks[..third]
        .iter()
        .map(BlockMetric::gas_per_sec)
        .collect();
    let mut last: Vec<f64> = blocks[blocks.len() - third..]
        .iter()
        .map(BlockMetric::gas_per_sec)
        .collect();
    first.sort_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
    last.sort_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
    let first_med = percentile_f64(&first, 50.0);
    let last_med = percentile_f64(&last, 50.0);
    let monotonic_slowdown = first_med > 0.0 && last_med < first_med * 0.85;

    // Linear regression of RSS over block index.
    let n = blocks.len() as f64;
    let mean_x = (n - 1.0) / 2.0;
    let mean_y = blocks.iter().map(|b| b.rss_bytes as f64).sum::<f64>() / n;
    let mut num = 0.0f64;
    let mut den = 0.0f64;
    for (i, b) in blocks.iter().enumerate() {
        let dx = i as f64 - mean_x;
        num += dx * (b.rss_bytes as f64 - mean_y);
        den += dx * dx;
    }
    let slope = if den > 0.0 { num / den } else { 0.0 };
    let per_10k = (slope * 10_000.0) as i64;
    (per_10k, monotonic_slowdown)
}

fn window_extremes(windows: &[WindowMetric]) -> (f64, f64) {
    if windows.is_empty() {
        return (0.0, 0.0);
    }
    let worst = windows
        .iter()
        .map(|w| w.gas_per_sec)
        .fold(f64::INFINITY, f64::min);
    let mut sorted: Vec<f64> = windows.iter().map(|w| w.gas_per_sec).collect();
    sorted.sort_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
    let steady = percentile_f64(&sorted, 50.0);
    (worst.max(0.0), steady)
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct HostInfo {
    pub os: String,
    pub cpu: String,
    pub memory_total_bytes: u64,
    pub git_sha: String,
    pub captured_at: String,
}

impl HostInfo {
    pub fn collect() -> Self {
        use sysinfo::System;
        let mut sys = System::new();
        sys.refresh_memory();
        sys.refresh_cpu_all();
        let cpu = sys
            .cpus()
            .first()
            .map(|c| format!("{} ({} cores)", c.brand(), sys.cpus().len()))
            .unwrap_or_else(|| "unknown".into());
        let captured_at = chrono::Utc::now().to_rfc3339();
        let git_sha = std::process::Command::new("git")
            .args(["rev-parse", "--short=12", "HEAD"])
            .output()
            .ok()
            .and_then(|o| {
                if o.status.success() {
                    Some(String::from_utf8_lossy(&o.stdout).trim().to_string())
                } else {
                    None
                }
            })
            .unwrap_or_default();
        Self {
            os: format!(
                "{} {}",
                System::name().unwrap_or_default(),
                System::os_version().unwrap_or_default()
            ),
            cpu,
            memory_total_bytes: sys.total_memory(),
            git_sha,
            captured_at,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::metrics::rolling::WindowMetric;

    fn block(n: u64, ns: u64, gas: u64, rss: u64) -> BlockMetric {
        BlockMetric {
            block_number: n,
            wall_clock_ns: ns,
            cpu_ns: ns,
            gas_used: gas,
            tx_count: 1,
            success_count: 1,
            rss_bytes: rss,
        }
    }

    #[test]
    fn summary_reports_basic_stats() {
        let blocks: Vec<_> = (0..100u64)
            .map(|i| block(i, 1_000_000, 1_000_000, 100_000_000 + i * 1_000))
            .collect();
        let win = WindowMetric {
            window_index: 0,
            block_range: (0, 100),
            gas_per_sec: 1.0e9,
            wall_clock_p95_ns: 1_000_000,
            peak_rss_bytes: 100_000_000,
            block_count: 100,
        };
        let s = SummaryMetrics::from_blocks(&blocks, &[win]);
        assert_eq!(s.block_count, 100);
        assert!(s.gas_per_sec_mean > 0.0);
        assert_eq!(s.peak_rss_bytes, 100_099_000);
    }

    #[test]
    fn slowdown_detected() {
        let mut blocks: Vec<_> = (0..30u64)
            .map(|i| block(i, 1_000_000, 1_000_000, 0))
            .collect();
        for b in &mut blocks[20..] {
            b.wall_clock_ns = 10_000_000;
        }
        let s = SummaryMetrics::from_blocks(&blocks, &[]);
        assert!(s.monotonic_slowdown_detected);
    }
}
