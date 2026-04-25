use rand::{rngs::StdRng, RngCore, SeedableRng};
use serde::{Deserialize, Serialize};

use crate::metrics::RunResult;

/// Metric identifier used in delta tables and verdict logic.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum MetricKey {
    WallClockNs,
    CpuNs,
    GasPerSec,
    RssBytes,
}

/// Mean delta + 95% CI from a paired bootstrap.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BootstrapDelta {
    pub n: usize,
    pub baseline_mean: f64,
    pub feature_mean: f64,
    pub mean: f64,
    pub ci_low_95: f64,
    pub ci_high_95: f64,
}

/// Verdict aggregated across all metrics for a single manifest comparison.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum Verdict {
    Improvement,
    Neutral,
    Regression { metric: String, delta_pct: f64 },
}

impl Verdict {
    pub fn label(&self) -> &'static str {
        match self {
            Self::Improvement => "IMPROVEMENT",
            Self::Neutral => "NEUTRAL",
            Self::Regression { .. } => "REGRESSION",
        }
    }
}

/// Output of `compare` over two RunResults.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComparisonReport {
    pub manifest_name: String,
    pub baseline_summary: crate::metrics::SummaryMetrics,
    pub feature_summary: crate::metrics::SummaryMetrics,
    pub deltas: Vec<(MetricKey, BootstrapDelta)>,
    pub verdict: Verdict,
}

/// Compare two single runs (no within-run pairing).
pub fn compare(
    baseline: &RunResult,
    feature: &RunResult,
    bootstrap_iters: usize,
    tolerance_pct: f64,
    seed: u64,
) -> ComparisonReport {
    let n = baseline.blocks.len().min(feature.blocks.len());
    let mut deltas = Vec::new();
    for m in [
        MetricKey::WallClockNs,
        MetricKey::CpuNs,
        MetricKey::GasPerSec,
        MetricKey::RssBytes,
    ] {
        let mut paired: Vec<(f64, f64)> = Vec::with_capacity(n);
        for i in 0..n {
            let b = match m {
                MetricKey::WallClockNs => baseline.blocks[i].wall_clock_ns as f64,
                MetricKey::CpuNs => baseline.blocks[i].cpu_ns as f64,
                MetricKey::GasPerSec => baseline.blocks[i].gas_per_sec(),
                MetricKey::RssBytes => baseline.blocks[i].rss_bytes as f64,
            };
            let f = match m {
                MetricKey::WallClockNs => feature.blocks[i].wall_clock_ns as f64,
                MetricKey::CpuNs => feature.blocks[i].cpu_ns as f64,
                MetricKey::GasPerSec => feature.blocks[i].gas_per_sec(),
                MetricKey::RssBytes => feature.blocks[i].rss_bytes as f64,
            };
            paired.push((b, f));
        }
        deltas.push((m, bootstrap_paired_delta(&paired, bootstrap_iters, seed)));
    }
    let verdict = decide_verdict_for_compare(&deltas, tolerance_pct);
    ComparisonReport {
        manifest_name: baseline.manifest_name.clone(),
        baseline_summary: baseline.summary.clone(),
        feature_summary: feature.summary.clone(),
        deltas,
        verdict,
    }
}

fn decide_verdict_for_compare(
    deltas: &[(MetricKey, BootstrapDelta)],
    tolerance_pct: f64,
) -> Verdict {
    let mut worst: Option<(MetricKey, f64)> = None;
    for (k, d) in deltas {
        let baseline_mean = d.baseline_mean.max(1e-9);
        let pct = match k {
            MetricKey::GasPerSec => -d.mean / baseline_mean * 100.0,
            _ => d.mean / baseline_mean * 100.0,
        };
        let ci_pct_lower = match k {
            MetricKey::GasPerSec => -d.ci_low_95 / baseline_mean * 100.0,
            _ => d.ci_low_95 / baseline_mean * 100.0,
        };
        if ci_pct_lower > tolerance_pct && worst.map(|(_, w)| pct > w).unwrap_or(true) {
            worst = Some((*k, pct));
        }
    }
    if let Some((m, pct)) = worst {
        return Verdict::Regression {
            metric: format!("{m:?}"),
            delta_pct: pct,
        };
    }

    let mut improved = false;
    for (k, d) in deltas {
        let baseline_mean = d.baseline_mean.max(1e-9);
        let high = match k {
            MetricKey::GasPerSec => -d.ci_high_95 / baseline_mean * 100.0,
            _ => d.ci_high_95 / baseline_mean * 100.0,
        };
        if high < 0.0 {
            improved = true;
        }
    }
    if improved {
        Verdict::Improvement
    } else {
        Verdict::Neutral
    }
}

/// Bootstrap a 95% CI for the mean of `feature - baseline`.
pub fn bootstrap_paired_delta(
    paired: &[(f64, f64)],
    iterations: usize,
    seed: u64,
) -> BootstrapDelta {
    if paired.is_empty() {
        return BootstrapDelta {
            n: 0,
            baseline_mean: 0.0,
            feature_mean: 0.0,
            mean: 0.0,
            ci_low_95: 0.0,
            ci_high_95: 0.0,
        };
    }
    let baseline_mean = paired.iter().map(|p| p.0).sum::<f64>() / paired.len() as f64;
    let feature_mean = paired.iter().map(|p| p.1).sum::<f64>() / paired.len() as f64;
    let mean_delta = feature_mean - baseline_mean;

    let mut rng = StdRng::seed_from_u64(seed);
    let n = paired.len();
    let mut sample_means = Vec::with_capacity(iterations);
    for _ in 0..iterations {
        let mut sum = 0.0;
        for _ in 0..n {
            let idx = (rng.next_u32() as usize) % n;
            let (b, f) = paired[idx];
            sum += f - b;
        }
        sample_means.push(sum / n as f64);
    }
    sample_means.sort_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
    let lo_idx = ((iterations as f64) * 0.025).floor() as usize;
    let hi_idx = ((iterations as f64) * 0.975).ceil() as usize - 1;
    let ci_low = sample_means[lo_idx.min(sample_means.len() - 1)];
    let ci_high = sample_means[hi_idx.min(sample_means.len() - 1)];

    BootstrapDelta {
        n,
        baseline_mean,
        feature_mean,
        mean: mean_delta,
        ci_low_95: ci_low,
        ci_high_95: ci_high,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn bootstrap_identical_samples_has_zero_ci() {
        let pairs: Vec<_> = (0..50).map(|i| (i as f64, i as f64)).collect();
        let d = bootstrap_paired_delta(&pairs, 1000, 42);
        assert_eq!(d.n, 50);
        assert!(d.mean.abs() < 1e-9);
        assert!(d.ci_low_95.abs() < 1e-9);
        assert!(d.ci_high_95.abs() < 1e-9);
    }

    #[test]
    fn bootstrap_detects_clear_improvement() {
        let pairs: Vec<_> = (0..50)
            .map(|i| (100.0_f64 + i as f64, 50.0 + i as f64))
            .collect();
        let d = bootstrap_paired_delta(&pairs, 1000, 42);
        assert!(d.mean < 0.0);
        assert!(d.ci_high_95 < 0.0); // Entirely below zero → clear improvement.
    }

    #[test]
    fn verdict_regression_when_metric_strictly_worse() {
        let deltas = vec![(
            MetricKey::WallClockNs,
            BootstrapDelta {
                n: 10,
                baseline_mean: 1000.0,
                feature_mean: 1500.0,
                mean: 500.0,
                ci_low_95: 400.0,
                ci_high_95: 600.0,
            },
        )];
        let v = decide_verdict_for_compare(&deltas, 5.0);
        assert!(matches!(v, Verdict::Regression { .. }));
    }

    #[test]
    fn verdict_improvement_when_gas_per_sec_strictly_higher() {
        let deltas = vec![(
            MetricKey::GasPerSec,
            BootstrapDelta {
                n: 10,
                baseline_mean: 100.0,
                feature_mean: 150.0,
                mean: 50.0,
                ci_low_95: 40.0,
                ci_high_95: 60.0,
            },
        )];
        let v = decide_verdict_for_compare(&deltas, 5.0);
        assert_eq!(v, Verdict::Improvement);
    }
}
