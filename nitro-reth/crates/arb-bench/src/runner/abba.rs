use serde::{Deserialize, Serialize};

use super::{in_process::InProcessRunner, RunnerConfig, Workload};
use crate::{
    metrics::{BlockMetric, RunResult},
    report::compare::{bootstrap_paired_delta, BootstrapDelta, MetricKey, Verdict},
};

/// Configuration for the ABBA scheduler.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AbbaConfig {
    /// Number of A-B-B-A iterations. Total runs per side = `2 * iterations`.
    pub iterations: usize,
    /// Bootstrap iteration count for the paired CI computation.
    pub bootstrap_iters: usize,
    /// Allowed regression in percent before the verdict turns to `Regression`.
    pub tolerance_pct: f64,
    /// PRNG seed for the bootstrap.
    pub seed: u64,
    pub runner: RunnerConfig,
}

impl Default for AbbaConfig {
    fn default() -> Self {
        Self {
            iterations: 3,
            bootstrap_iters: 10_000,
            tolerance_pct: 5.0,
            seed: 0xC0FF_EE12_3456_789A,
            runner: RunnerConfig::default(),
        }
    }
}

/// Paired baseline + feature runs on the same workload.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PairedSample {
    pub iter_index: usize,
    pub baseline: RunResult,
    pub feature: RunResult,
}

/// Output of an ABBA-driven comparison.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AbbaResult {
    pub manifest_name: String,
    pub iterations: usize,
    pub samples: Vec<PairedSample>,
    pub deltas: Vec<(MetricKey, BootstrapDelta)>,
    pub verdict: Verdict,
}

/// Run an A-B-B-A interleaved comparison. Each factory is called `2 * iterations` times.
pub fn run_abba<FB, FF>(
    config: &AbbaConfig,
    manifest_name: &str,
    mut build_baseline: FB,
    mut build_feature: FF,
) -> eyre::Result<AbbaResult>
where
    FB: FnMut() -> eyre::Result<Workload>,
    FF: FnMut() -> eyre::Result<Workload>,
{
    let mut samples = Vec::with_capacity(config.iterations);
    for i in 0..config.iterations {
        let order = if i % 2 == 0 {
            [Side::Baseline, Side::Feature, Side::Feature, Side::Baseline]
        } else {
            [Side::Feature, Side::Baseline, Side::Baseline, Side::Feature]
        };

        let mut runs: [Option<RunResult>; 4] = Default::default();
        for (slot, side) in order.iter().enumerate() {
            let workload = match side {
                Side::Baseline => build_baseline()?,
                Side::Feature => build_feature()?,
            };
            let mut runner = InProcessRunner::new(config.runner.clone());
            runs[slot] = Some(runner.run(workload)?);
        }

        let mut baseline_runs = Vec::new();
        let mut feature_runs = Vec::new();
        for (slot, side) in order.iter().enumerate() {
            let r = runs[slot].take().unwrap();
            match side {
                Side::Baseline => baseline_runs.push(r),
                Side::Feature => feature_runs.push(r),
            }
        }
        let baseline = average_runs(&baseline_runs)?;
        let feature = average_runs(&feature_runs)?;

        samples.push(PairedSample {
            iter_index: i,
            baseline,
            feature,
        });
    }

    let deltas = compute_paired_deltas(&samples, config);
    let verdict = decide_verdict(&deltas, config.tolerance_pct);

    Ok(AbbaResult {
        manifest_name: manifest_name.to_string(),
        iterations: config.iterations,
        samples,
        deltas,
        verdict,
    })
}

#[derive(Debug, Clone, Copy)]
enum Side {
    Baseline,
    Feature,
}

/// Combine N runs by averaging per-block metrics.
fn average_runs(runs: &[RunResult]) -> eyre::Result<RunResult> {
    if runs.is_empty() {
        eyre::bail!("average_runs: empty");
    }
    if runs.len() == 1 {
        return Ok(runs[0].clone());
    }
    let n = runs[0].blocks.len();
    if !runs.iter().all(|r| r.blocks.len() == n) {
        eyre::bail!("average_runs: differing block counts");
    }
    let mut blocks: Vec<BlockMetric> = Vec::with_capacity(n);
    for i in 0..n {
        let wall: u64 =
            runs.iter().map(|r| r.blocks[i].wall_clock_ns).sum::<u64>() / runs.len() as u64;
        let cpu: u64 = runs.iter().map(|r| r.blocks[i].cpu_ns).sum::<u64>() / runs.len() as u64;
        let rss: u64 = runs.iter().map(|r| r.blocks[i].rss_bytes).sum::<u64>() / runs.len() as u64;
        blocks.push(BlockMetric {
            block_number: runs[0].blocks[i].block_number,
            wall_clock_ns: wall,
            cpu_ns: cpu,
            gas_used: runs[0].blocks[i].gas_used,
            tx_count: runs[0].blocks[i].tx_count,
            success_count: runs[0].blocks[i].success_count,
            rss_bytes: rss,
        });
    }
    let windows = crate::metrics::rolling::build_windows(&blocks, 500);
    let summary = crate::metrics::SummaryMetrics::from_blocks(&blocks, &windows);
    Ok(RunResult {
        manifest_name: runs[0].manifest_name.clone(),
        blocks,
        windows,
        summary,
        host: runs[0].host.clone(),
    })
}

/// Bootstrap paired deltas per metric.
fn compute_paired_deltas(
    samples: &[PairedSample],
    config: &AbbaConfig,
) -> Vec<(MetricKey, BootstrapDelta)> {
    let mut out = Vec::new();
    let metrics = [
        MetricKey::WallClockNs,
        MetricKey::GasPerSec,
        MetricKey::CpuNs,
        MetricKey::RssBytes,
    ];
    for m in metrics {
        let mut paired = Vec::new();
        for s in samples {
            let n = s.baseline.blocks.len().min(s.feature.blocks.len());
            for i in 0..n {
                let b = metric_value(&s.baseline.blocks[i], m);
                let f = metric_value(&s.feature.blocks[i], m);
                paired.push((b, f));
            }
        }
        if paired.is_empty() {
            continue;
        }
        let delta = bootstrap_paired_delta(&paired, config.bootstrap_iters, config.seed);
        out.push((m, delta));
    }
    out
}

fn metric_value(b: &BlockMetric, m: MetricKey) -> f64 {
    match m {
        MetricKey::WallClockNs => b.wall_clock_ns as f64,
        MetricKey::CpuNs => b.cpu_ns as f64,
        MetricKey::GasPerSec => b.gas_per_sec(),
        MetricKey::RssBytes => b.rss_bytes as f64,
    }
}

fn decide_verdict(deltas: &[(MetricKey, BootstrapDelta)], tolerance_pct: f64) -> Verdict {
    let mut worst: Option<(MetricKey, f64)> = None;
    for (k, d) in deltas {
        let baseline_mean = d.baseline_mean.max(1e-9);
        let pct = match k {
            MetricKey::GasPerSec => -d.mean / baseline_mean * 100.0,
            _ => d.mean / baseline_mean * 100.0,
        };
        let ci_pct = match k {
            MetricKey::GasPerSec => -d.ci_low_95 / baseline_mean * 100.0,
            _ => d.ci_low_95 / baseline_mean * 100.0,
        };
        if ci_pct > tolerance_pct {
            match worst {
                Some((_, w)) if w >= pct => {}
                _ => worst = Some((*k, pct)),
            }
        }
    }
    if let Some((metric, pct)) = worst {
        return Verdict::Regression {
            metric: format!("{metric:?}"),
            delta_pct: pct,
        };
    }

    let mut any_improvement = false;
    for (k, d) in deltas {
        let baseline_mean = d.baseline_mean.max(1e-9);
        let pct_high = match k {
            MetricKey::GasPerSec => -d.ci_high_95 / baseline_mean * 100.0,
            _ => d.ci_high_95 / baseline_mean * 100.0,
        };
        if pct_high < 0.0 {
            any_improvement = true;
        }
    }
    if any_improvement {
        Verdict::Improvement
    } else {
        Verdict::Neutral
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::capture::synthetic::generate;

    #[test]
    fn abba_smoke_runs_and_yields_neutral_for_identical_runs() {
        let cfg = AbbaConfig {
            iterations: 1,
            bootstrap_iters: 200,
            tolerance_pct: 50.0,
            seed: 1,
            runner: RunnerConfig {
                rolling_window_blocks: 2,
                abort_on_block_error: false,
            },
        };
        let result = run_abba(
            &cfg,
            "test/abba",
            || {
                generate(
                    "test/abba",
                    421614,
                    30,
                    "transfer_train",
                    &serde_json::json!({ "block_count": 2, "txs_per_block": 2 }),
                )
            },
            || {
                generate(
                    "test/abba",
                    421614,
                    30,
                    "transfer_train",
                    &serde_json::json!({ "block_count": 2, "txs_per_block": 2 }),
                )
            },
        )
        .unwrap();
        assert_eq!(result.iterations, 1);
        assert!(!result.deltas.is_empty());
        // Verdict for identical workloads under wide tolerance is neutral or improvement;
        // never regression.
        assert!(!matches!(result.verdict, Verdict::Regression { .. }));
    }
}
