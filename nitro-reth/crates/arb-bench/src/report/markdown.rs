use std::fmt::Write as _;

use crate::{
    metrics::RunResult,
    report::compare::{BootstrapDelta, ComparisonReport, MetricKey, Verdict},
    runner::abba::AbbaResult,
};

/// Markdown summary suitable for posting as a PR comment.
pub fn render_abba(result: &AbbaResult) -> String {
    let mut s = String::new();
    let _ = writeln!(s, "## arb-bench: `{}`", result.manifest_name);
    let _ = writeln!(s, "- Iterations: {}", result.iterations);
    let _ = writeln!(s, "- Verdict: **{}**", result.verdict.label());
    if let Verdict::Regression { metric, delta_pct } = &result.verdict {
        let _ = writeln!(s, "- Regressing metric: `{metric}` (Δ {delta_pct:+.2}%)");
    }
    let _ = writeln!(s);
    let _ = writeln!(
        s,
        "| Metric | Baseline | Feature | Δ mean | 95% CI | Verdict |"
    );
    let _ = writeln!(s, "|---|---|---|---|---|---|");
    for (key, d) in &result.deltas {
        let row = render_metric_row(*key, d);
        let _ = writeln!(s, "{row}");
    }
    s
}

pub fn render_comparison(report: &ComparisonReport) -> String {
    let mut s = String::new();
    let _ = writeln!(s, "## arb-bench compare: `{}`", report.manifest_name);
    let _ = writeln!(s, "- Verdict: **{}**", report.verdict.label());
    let _ = writeln!(s);
    let _ = writeln!(s, "| Metric | Baseline | Feature | Δ mean | 95% CI |");
    let _ = writeln!(s, "|---|---|---|---|---|");
    for (key, d) in &report.deltas {
        let row = render_metric_row(*key, d);
        let _ = writeln!(s, "{row}");
    }
    s
}

pub fn render_run_summary(r: &RunResult) -> String {
    let mut s = String::new();
    let _ = writeln!(s, "### `{}`", r.manifest_name);
    let _ = writeln!(s, "- Blocks: {}", r.summary.block_count);
    let _ = writeln!(s, "- Total gas: {}", r.summary.total_gas);
    let _ = writeln!(s, "- Mean gas/sec: {:.0}", r.summary.gas_per_sec_mean);
    let _ = writeln!(
        s,
        "- p50 / p95 / p99 wall (ms): {:.2} / {:.2} / {:.2}",
        r.summary.wall_clock_ns_p50 as f64 / 1.0e6,
        r.summary.wall_clock_ns_p95 as f64 / 1.0e6,
        r.summary.wall_clock_ns_p99 as f64 / 1.0e6
    );
    let _ = writeln!(
        s,
        "- Peak RSS: {:.1} MiB",
        r.summary.peak_rss_bytes as f64 / 1024.0 / 1024.0
    );
    if r.summary.monotonic_slowdown_detected {
        let _ = writeln!(s, "- ⚠️  Monotonic slowdown detected.");
    }
    s
}

fn render_metric_row(key: MetricKey, d: &BootstrapDelta) -> String {
    let (baseline_fmt, feature_fmt, delta_fmt, ci_fmt, verdict_label) = match key {
        MetricKey::WallClockNs => {
            let bl = d.baseline_mean / 1.0e6;
            let ft = d.feature_mean / 1.0e6;
            let pct = if d.baseline_mean > 0.0 {
                d.mean / d.baseline_mean * 100.0
            } else {
                0.0
            };
            let label = direction_label(pct, false);
            (
                format!("{bl:.2} ms"),
                format!("{ft:.2} ms"),
                format!("{:+.2} ms ({pct:+.2}%)", d.mean / 1.0e6),
                format!(
                    "[{:+.2}, {:+.2}] ms",
                    d.ci_low_95 / 1.0e6,
                    d.ci_high_95 / 1.0e6
                ),
                label,
            )
        }
        MetricKey::CpuNs => {
            let bl = d.baseline_mean / 1.0e6;
            let ft = d.feature_mean / 1.0e6;
            let pct = if d.baseline_mean > 0.0 {
                d.mean / d.baseline_mean * 100.0
            } else {
                0.0
            };
            let label = direction_label(pct, false);
            (
                format!("{bl:.2} ms"),
                format!("{ft:.2} ms"),
                format!("{:+.2} ms ({pct:+.2}%)", d.mean / 1.0e6),
                format!(
                    "[{:+.2}, {:+.2}] ms",
                    d.ci_low_95 / 1.0e6,
                    d.ci_high_95 / 1.0e6
                ),
                label,
            )
        }
        MetricKey::GasPerSec => {
            let bl = d.baseline_mean / 1.0e6;
            let ft = d.feature_mean / 1.0e6;
            let pct = if d.baseline_mean > 0.0 {
                d.mean / d.baseline_mean * 100.0
            } else {
                0.0
            };
            let label = direction_label(pct, true);
            (
                format!("{bl:.1} Mgas/s"),
                format!("{ft:.1} Mgas/s"),
                format!("{:+.1} Mgas/s ({pct:+.2}%)", d.mean / 1.0e6),
                format!(
                    "[{:+.1}, {:+.1}] Mgas/s",
                    d.ci_low_95 / 1.0e6,
                    d.ci_high_95 / 1.0e6
                ),
                label,
            )
        }
        MetricKey::RssBytes => {
            let bl = d.baseline_mean / (1024.0 * 1024.0);
            let ft = d.feature_mean / (1024.0 * 1024.0);
            let pct = if d.baseline_mean > 0.0 {
                d.mean / d.baseline_mean * 100.0
            } else {
                0.0
            };
            let label = direction_label(pct, false);
            (
                format!("{bl:.1} MiB"),
                format!("{ft:.1} MiB"),
                format!("{:+.1} MiB ({pct:+.2}%)", d.mean / (1024.0 * 1024.0)),
                format!(
                    "[{:+.1}, {:+.1}] MiB",
                    d.ci_low_95 / (1024.0 * 1024.0),
                    d.ci_high_95 / (1024.0 * 1024.0)
                ),
                label,
            )
        }
    };
    format!(
        "| {key:?} | {baseline_fmt} | {feature_fmt} | {delta_fmt} | {ci_fmt} | {verdict_label} |"
    )
}

fn direction_label(pct: f64, higher_is_better: bool) -> String {
    let improvement = (pct < 0.0 && !higher_is_better) || (pct > 0.0 && higher_is_better);
    let regression = (pct > 0.0 && !higher_is_better) || (pct < 0.0 && higher_is_better);
    if pct.abs() < 0.5 {
        "≈".into()
    } else if improvement {
        "↑ better".into()
    } else if regression {
        "↓ worse".into()
    } else {
        "≈".into()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::metrics::SummaryMetrics;

    #[test]
    fn render_compare_well_formed() {
        let report = ComparisonReport {
            manifest_name: "t".into(),
            baseline_summary: SummaryMetrics::default(),
            feature_summary: SummaryMetrics::default(),
            deltas: vec![(
                MetricKey::WallClockNs,
                BootstrapDelta {
                    n: 10,
                    baseline_mean: 1.0e6,
                    feature_mean: 9.0e5,
                    mean: -1.0e5,
                    ci_low_95: -2.0e5,
                    ci_high_95: -1.0e4,
                },
            )],
            verdict: Verdict::Improvement,
        };
        let s = render_comparison(&report);
        assert!(s.contains("IMPROVEMENT"));
        assert!(s.contains("WallClockNs"));
    }
}
