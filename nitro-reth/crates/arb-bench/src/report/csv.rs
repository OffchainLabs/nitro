use std::path::Path;

use crate::{metrics::RunResult, runner::abba::AbbaResult};

/// Per-block CSV emit for plotting.
pub fn write_run_blocks_csv(result: &RunResult, path: &Path) -> eyre::Result<()> {
    if let Some(p) = path.parent() {
        std::fs::create_dir_all(p)?;
    }
    let file = std::fs::File::create(path)?;
    let mut w = csv::Writer::from_writer(file);
    w.write_record([
        "manifest",
        "block",
        "wall_clock_ns",
        "cpu_ns",
        "gas_used",
        "tx_count",
        "success_count",
        "rss_bytes",
        "gas_per_sec",
    ])?;
    for b in &result.blocks {
        w.write_record([
            result.manifest_name.as_str(),
            &b.block_number.to_string(),
            &b.wall_clock_ns.to_string(),
            &b.cpu_ns.to_string(),
            &b.gas_used.to_string(),
            &b.tx_count.to_string(),
            &b.success_count.to_string(),
            &b.rss_bytes.to_string(),
            &format!("{:.2}", b.gas_per_sec()),
        ])?;
    }
    w.flush()?;
    Ok(())
}

/// Per-window CSV emit for visualizing flush/pruner cycles.
pub fn write_run_windows_csv(result: &RunResult, path: &Path) -> eyre::Result<()> {
    if let Some(p) = path.parent() {
        std::fs::create_dir_all(p)?;
    }
    let file = std::fs::File::create(path)?;
    let mut w = csv::Writer::from_writer(file);
    w.write_record([
        "manifest",
        "window_index",
        "block_first",
        "block_last",
        "block_count",
        "gas_per_sec",
        "wall_clock_p95_ns",
        "peak_rss_bytes",
    ])?;
    for win in &result.windows {
        w.write_record([
            result.manifest_name.as_str(),
            &win.window_index.to_string(),
            &win.block_range.0.to_string(),
            &win.block_range.1.to_string(),
            &win.block_count.to_string(),
            &format!("{:.2}", win.gas_per_sec),
            &win.wall_clock_p95_ns.to_string(),
            &win.peak_rss_bytes.to_string(),
        ])?;
    }
    w.flush()?;
    Ok(())
}

/// Wide-format ABBA delta table for plotting deltas across iterations.
pub fn write_abba_deltas_csv(result: &AbbaResult, path: &Path) -> eyre::Result<()> {
    if let Some(p) = path.parent() {
        std::fs::create_dir_all(p)?;
    }
    let file = std::fs::File::create(path)?;
    let mut w = csv::Writer::from_writer(file);
    w.write_record([
        "manifest",
        "metric",
        "n",
        "baseline_mean",
        "feature_mean",
        "delta_mean",
        "ci_low_95",
        "ci_high_95",
    ])?;
    for (k, d) in &result.deltas {
        w.write_record([
            result.manifest_name.as_str(),
            &format!("{k:?}"),
            &d.n.to_string(),
            &format!("{:.6}", d.baseline_mean),
            &format!("{:.6}", d.feature_mean),
            &format!("{:.6}", d.mean),
            &format!("{:.6}", d.ci_low_95),
            &format!("{:.6}", d.ci_high_95),
        ])?;
    }
    w.flush()?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::metrics::{BlockMetric, HostInfo, SummaryMetrics};

    #[test]
    fn writes_blocks_csv_with_header_row() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("blocks.csv");
        let r = RunResult {
            manifest_name: "t".into(),
            blocks: vec![BlockMetric {
                block_number: 1,
                wall_clock_ns: 1000,
                cpu_ns: 900,
                gas_used: 21000,
                tx_count: 1,
                success_count: 1,
                rss_bytes: 1_000_000,
            }],
            windows: vec![],
            summary: SummaryMetrics::default(),
            host: HostInfo::default(),
        };
        write_run_blocks_csv(&r, &path).unwrap();
        let s = std::fs::read_to_string(&path).unwrap();
        assert!(s.starts_with("manifest,block,"));
        assert!(s.contains(",21000,"));
    }
}
