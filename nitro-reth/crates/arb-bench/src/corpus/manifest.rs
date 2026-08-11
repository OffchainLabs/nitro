use std::path::{Path, PathBuf};

use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Manifest {
    pub name: String,
    pub category: WorkloadCategory,
    pub scale: ScaleTier,
    pub arbos_version: u64,
    #[serde(default = "default_chain_id")]
    pub chain_id: u64,
    pub corpus_version: String,
    #[serde(default)]
    pub description: String,
    pub messages: MessageSource,
    #[serde(default)]
    pub metrics: MetricsSpec,
    #[serde(default)]
    pub regression: RegressionSpec,
    #[serde(default)]
    pub prewarm: Option<PrewarmSpec>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrewarmSpec {
    pub accounts: u64,
    #[serde(default = "default_prewarm_seed")]
    pub seed: u64,
    /// Each prewarm account gets this balance (hex-string or decimal).
    #[serde(default = "default_prewarm_balance_wei")]
    pub balance_wei: String,
}

fn default_prewarm_seed() -> u64 {
    0x9E37_79B1_85EB_CA87
}

fn default_prewarm_balance_wei() -> String {
    "0x0".to_string()
}

fn default_chain_id() -> u64 {
    421614
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum WorkloadCategory {
    Baseline,
    HeavyBlocks,
    Stylus,
    RetryableChurn,
    DepositBurst,
    FeeEscalation,
    PrecompileHeavy,
    ContractDeployHeavy,
    MixedRealistic,
    MaxCalldata,
    ThousandTxBlock,
    StylusDeepCallStack,
    StylusColdCache,
    RetryableTimeoutSweep,
    PrecompileFanout,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum ScaleTier {
    Micro,
    Short,
    Medium,
    Long,
    Endurance,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "source", rename_all = "snake_case")]
pub enum MessageSource {
    SyntheticGenerator {
        generator: String,
        #[serde(default)]
        params: serde_json::Value,
    },
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsSpec {
    /// Width of the rolling window used for long-run reporting (in blocks).
    #[serde(default = "default_window")]
    pub rolling_window_blocks: usize,
}

fn default_window() -> usize {
    500
}

impl Default for MetricsSpec {
    fn default() -> Self {
        Self {
            rolling_window_blocks: default_window(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegressionSpec {
    /// Whether this manifest is a PR-gate blocker.
    #[serde(default = "default_gate")]
    pub gate: bool,
    /// Maximum allowed regression in percent before the gate fails.
    #[serde(default = "default_tolerance")]
    pub tolerance_pct: f64,
    /// Statistical method used for the verdict.
    #[serde(default)]
    pub paired_stat: PairedStat,
    /// Bootstrap iteration count (used when `paired_stat == bootstrap_95ci`).
    #[serde(default = "default_bootstrap_iters")]
    pub bootstrap_iters: usize,
}

fn default_gate() -> bool {
    true
}
fn default_tolerance() -> f64 {
    5.0
}
fn default_bootstrap_iters() -> usize {
    10_000
}

impl Default for RegressionSpec {
    fn default() -> Self {
        Self {
            gate: true,
            tolerance_pct: 5.0,
            paired_stat: PairedStat::default(),
            bootstrap_iters: 10_000,
        }
    }
}

#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PairedStat {
    #[default]
    #[serde(rename = "bootstrap_95ci")]
    Bootstrap95Ci,
    /// Median-of-pairs comparison; cheaper but less informative.
    PairedMedian,
}

impl Manifest {
    /// Load a manifest from disk, resolving any `genesis_ref` path relative to
    /// the manifest file's directory if present.
    pub fn from_path(path: &Path) -> eyre::Result<Self> {
        let bytes = std::fs::read(path)
            .map_err(|e| eyre::eyre!("read manifest {}: {e}", path.display()))?;
        let manifest: Self = serde_json::from_slice(&bytes)
            .map_err(|e| eyre::eyre!("parse manifest {}: {e}", path.display()))?;
        manifest.validate()?;
        Ok(manifest)
    }

    /// Discover all manifests under a directory tree. Skips directories whose
    /// name starts with `_` (used for shared assets like genesis files).
    pub fn discover(root: &Path) -> eyre::Result<Vec<(PathBuf, Self)>> {
        let mut out = Vec::new();
        let mut stack = vec![root.to_path_buf()];
        while let Some(dir) = stack.pop() {
            let entries = std::fs::read_dir(&dir)
                .map_err(|e| eyre::eyre!("read dir {}: {e}", dir.display()))?;
            for entry in entries {
                let entry = entry?;
                let path = entry.path();
                let ft = entry.file_type()?;
                if ft.is_dir() {
                    let name = path.file_name().and_then(|s| s.to_str()).unwrap_or("");
                    if name.starts_with('_') {
                        continue;
                    }
                    stack.push(path);
                } else if path.extension().and_then(|s| s.to_str()) == Some("json") {
                    let manifest = Self::from_path(&path)?;
                    out.push((path, manifest));
                }
            }
        }
        out.sort_by(|a, b| a.0.cmp(&b.0));
        Ok(out)
    }

    pub fn validate(&self) -> eyre::Result<()> {
        if self.name.is_empty() {
            eyre::bail!("manifest name is empty");
        }
        if !(1..=200).contains(&self.arbos_version) {
            eyre::bail!("arbos_version {} out of range", self.arbos_version);
        }
        if self.regression.tolerance_pct <= 0.0 {
            eyre::bail!("regression.tolerance_pct must be > 0");
        }
        if self.metrics.rolling_window_blocks == 0 {
            eyre::bail!("metrics.rolling_window_blocks must be > 0");
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn synthetic_manifest_roundtrip() {
        let m = Manifest {
            name: "synthetic/thousand-tx-block/short".into(),
            category: WorkloadCategory::ThousandTxBlock,
            scale: ScaleTier::Short,
            arbos_version: 60,
            chain_id: 421614,
            corpus_version: "1.0.0".into(),
            description: "1000-tx blocks".into(),
            messages: MessageSource::SyntheticGenerator {
                generator: "thousand_tx_block".into(),
                params: serde_json::json!({ "block_count": 50, "txs_per_block": 1000 }),
            },
            metrics: MetricsSpec::default(),
            regression: RegressionSpec::default(),
            prewarm: None,
        };
        let s = serde_json::to_string_pretty(&m).unwrap();
        let back: Manifest = serde_json::from_str(&s).unwrap();
        assert_eq!(back.name, m.name);
        assert_eq!(back.category, WorkloadCategory::ThousandTxBlock);
        assert_eq!(back.scale, ScaleTier::Short);
    }

    #[test]
    fn invalid_arbos_rejected() {
        let mut m = Manifest {
            name: "x".into(),
            category: WorkloadCategory::Baseline,
            scale: ScaleTier::Short,
            arbos_version: 0,
            chain_id: 421614,
            corpus_version: "1.0.0".into(),
            description: String::new(),
            messages: MessageSource::SyntheticGenerator {
                generator: "noop".into(),
                params: serde_json::json!({}),
            },
            metrics: MetricsSpec::default(),
            regression: RegressionSpec::default(),
            prewarm: None,
        };
        assert!(m.validate().is_err());
        m.arbos_version = 30;
        assert!(m.validate().is_ok());
    }
}
