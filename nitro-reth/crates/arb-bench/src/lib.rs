//! Performance benchmarking framework for arbreth.

pub mod capture;
pub mod corpus;
pub mod metrics;
pub mod report;
pub mod runner;

pub use corpus::manifest::{Manifest, MessageSource, RegressionSpec, ScaleTier, WorkloadCategory};
pub use metrics::{rolling::WindowMetric, BlockMetric, RunResult, SummaryMetrics};
pub use report::compare::{BootstrapDelta, ComparisonReport, Verdict};
pub use runner::{
    abba::{AbbaConfig, AbbaResult, PairedSample},
    in_process::InProcessRunner,
};
