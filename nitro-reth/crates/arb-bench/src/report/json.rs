use std::path::Path;

use crate::{metrics::RunResult, report::compare::ComparisonReport, runner::abba::AbbaResult};

pub fn write_run_result(result: &RunResult, path: &Path) -> eyre::Result<()> {
    write_json(result, path)
}

pub fn write_abba_result(result: &AbbaResult, path: &Path) -> eyre::Result<()> {
    write_json(result, path)
}

pub fn write_comparison(report: &ComparisonReport, path: &Path) -> eyre::Result<()> {
    write_json(report, path)
}

pub fn read_run_result(path: &Path) -> eyre::Result<RunResult> {
    let bytes = std::fs::read(path)?;
    Ok(serde_json::from_slice(&bytes)?)
}

pub fn read_abba_result(path: &Path) -> eyre::Result<AbbaResult> {
    let bytes = std::fs::read(path)?;
    Ok(serde_json::from_slice(&bytes)?)
}

fn write_json<T: serde::Serialize>(value: &T, path: &Path) -> eyre::Result<()> {
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)?;
    }
    let mut bytes = serde_json::to_vec_pretty(value)?;
    bytes.push(b'\n');
    std::fs::write(path, bytes)?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::metrics::{BlockMetric, HostInfo, RunResult, SummaryMetrics};

    #[test]
    fn round_trip_run_result() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("r.json");
        let r = RunResult {
            manifest_name: "t".into(),
            blocks: vec![BlockMetric {
                block_number: 1,
                wall_clock_ns: 100,
                cpu_ns: 80,
                gas_used: 21000,
                tx_count: 1,
                success_count: 1,
                rss_bytes: 1_000_000,
            }],
            windows: vec![],
            summary: SummaryMetrics::default(),
            host: HostInfo::default(),
        };
        write_run_result(&r, &path).unwrap();
        let r2 = read_run_result(&path).unwrap();
        assert_eq!(r2.blocks.len(), 1);
        assert_eq!(r2.manifest_name, "t");
    }
}
