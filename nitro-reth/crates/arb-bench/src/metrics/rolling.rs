use serde::{Deserialize, Serialize};

use super::BlockMetric;

/// Per-window aggregated metrics.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WindowMetric {
    pub window_index: usize,
    pub block_range: (u64, u64),
    pub gas_per_sec: f64,
    pub wall_clock_p95_ns: u64,
    pub peak_rss_bytes: u64,
    pub block_count: usize,
}

/// Group block metrics into fixed-width windows.
pub fn build_windows(blocks: &[BlockMetric], window_size: usize) -> Vec<WindowMetric> {
    if window_size == 0 || blocks.is_empty() {
        return Vec::new();
    }
    let mut out = Vec::new();
    for (i, chunk) in blocks.chunks(window_size).enumerate() {
        let total_wall: u64 = chunk.iter().map(|b| b.wall_clock_ns).sum();
        let total_gas: u64 = chunk.iter().map(|b| b.gas_used).sum();
        let mut sorted_wall: Vec<u64> = chunk.iter().map(|b| b.wall_clock_ns).collect();
        sorted_wall.sort_unstable();
        let p95_idx = ((sorted_wall.len() as f64 - 1.0) * 0.95).round() as usize;
        let p95 = sorted_wall.get(p95_idx).copied().unwrap_or(0);
        let peak_rss = chunk.iter().map(|b| b.rss_bytes).max().unwrap_or(0);
        let gps = if total_wall > 0 {
            (total_gas as f64) * 1_000_000_000.0 / (total_wall as f64)
        } else {
            0.0
        };
        let first = chunk.first().map(|b| b.block_number).unwrap_or(0);
        let last = chunk.last().map(|b| b.block_number).unwrap_or(0);
        out.push(WindowMetric {
            window_index: i,
            block_range: (first, last),
            gas_per_sec: gps,
            wall_clock_p95_ns: p95,
            peak_rss_bytes: peak_rss,
            block_count: chunk.len(),
        });
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

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
    fn windows_split_evenly() {
        let blocks: Vec<_> = (0..10u64).map(|i| block(i, 1_000, 1_000, 100)).collect();
        let w = build_windows(&blocks, 5);
        assert_eq!(w.len(), 2);
        assert_eq!(w[0].block_count, 5);
        assert_eq!(w[1].block_range, (5, 9));
    }

    #[test]
    fn empty_returns_empty() {
        assert!(build_windows(&[], 10).is_empty());
        assert!(build_windows(&[block(0, 1, 1, 1)], 0).is_empty());
    }
}
