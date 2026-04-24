//! arbreth-bench CLI.

use std::path::PathBuf;

use arb_bench::{
    capture::synthetic::generate,
    corpus::manifest::{Manifest, MessageSource},
    metrics::RunResult,
    report::{
        self,
        compare::{compare, ComparisonReport, Verdict},
        markdown,
    },
    runner::{
        abba::{run_abba, AbbaConfig, AbbaResult},
        in_process::InProcessRunner,
        subprocess::{SubprocessConfig, SubprocessRunner},
        RunnerConfig, Workload,
    },
};
use clap::{Parser, Subcommand, ValueEnum};
use tracing_subscriber::{fmt, EnvFilter};

#[derive(Parser, Debug)]
#[command(
    name = "arbreth-bench",
    version,
    about = "arbreth performance benchmarking harness"
)]
struct Cli {
    /// Log level (RUST_LOG style: e.g. `debug`, `arb_bench=info`).
    #[arg(long, global = true, default_value = "info")]
    log_level: String,

    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Execute one or more manifests and emit results.
    Run(RunCommand),
    /// Compare baseline vs feature on the identical workload (within-run ABBA).
    Abba(AbbaCommand),
    /// Compare two saved RunResult JSONs (post-hoc trend tracking).
    Compare(CompareCommand),
    /// Run a manifest under the standard runner, suitable for `cargo flamegraph`.
    Profile(ProfileCommand),
}

#[derive(Parser, Debug)]
struct RunCommand {
    /// Manifest paths (or directories — discovered recursively).
    #[arg(num_args = 1..)]
    manifests: Vec<PathBuf>,
    /// Output directory (one JSON + one CSV per manifest).
    #[arg(long, default_value = "bench/baselines/local")]
    out: PathBuf,
    /// Override the rolling-window size from the manifest.
    #[arg(long)]
    window: Option<usize>,
    /// Stop early on the first block-level error.
    #[arg(long, default_value_t = false)]
    abort_on_error: bool,
    /// Runner mode: `in-process` (default, fast, no real DB) or `subprocess`
    /// (spawn `arb-reth`, real MDBX + Stylus + flush).
    #[arg(long, value_enum, default_value_t = RunnerMode::InProcess)]
    mode: RunnerMode,
    /// Path to the `arb-reth` binary (subprocess mode only).
    #[arg(long, default_value = "target/release/arb-reth")]
    arbreth_binary: PathBuf,
    /// Genesis JSON (subprocess mode only).
    #[arg(long, default_value = "genesis/arbitrum-sepolia.json")]
    genesis: PathBuf,
}

#[derive(Copy, Clone, Debug, ValueEnum)]
enum RunnerMode {
    InProcess,
    Subprocess,
}

#[derive(Parser, Debug)]
struct AbbaCommand {
    /// Manifest paths.
    #[arg(long = "manifests", num_args = 1.., value_delimiter = ' ')]
    manifests: Vec<PathBuf>,
    /// Use a named preset instead of supplying `--manifests`.
    #[arg(long, value_enum)]
    preset: Option<Preset>,
    /// Number of A-B-B-A iterations.
    #[arg(long, default_value_t = 3)]
    iterations: usize,
    /// Tolerance percent for the regression gate.
    #[arg(long, default_value_t = 5.0)]
    tolerance_pct: f64,
    /// Bootstrap iterations for the paired CI.
    #[arg(long, default_value_t = 10_000)]
    bootstrap_iters: usize,
    /// Output directory.
    #[arg(long, default_value = "bench/baselines/abba")]
    out: PathBuf,
    /// Optional path used to record what we ran for CI artifacts.
    #[arg(long)]
    markdown_out: Option<PathBuf>,
    /// Exit non-zero if any manifest verdict is `Regression`.
    #[arg(long, default_value_t = true)]
    fail_on_regression: bool,
}

#[derive(Parser, Debug)]
struct CompareCommand {
    #[arg(long)]
    baseline: PathBuf,
    #[arg(long)]
    feature: PathBuf,
    #[arg(long)]
    out: Option<PathBuf>,
    #[arg(long, default_value_t = 5.0)]
    tolerance_pct: f64,
    #[arg(long, default_value_t = 10_000)]
    bootstrap_iters: usize,
    #[arg(long, default_value_t = 0xC0FF_EE12_3456_789A)]
    seed: u64,
}

#[derive(Parser, Debug)]
struct ProfileCommand {
    #[arg(long)]
    manifest: PathBuf,
    /// Override block count (helpful for short flamegraph runs).
    #[arg(long)]
    block_count_override: Option<usize>,
}

#[derive(Copy, Clone, Debug, ValueEnum)]
enum Preset {
    /// PR-gate slice: 6 categories × short × default.
    PrGate,
    /// Local quickcheck: 3 categories × short.
    Local,
    /// Nightly: 9 × (short+medium).
    Nightly,
}

impl Preset {
    fn manifests(&self) -> Vec<PathBuf> {
        let root = PathBuf::from("bench/corpus");
        let pick =
            |paths: &[&str]| -> Vec<PathBuf> { paths.iter().map(|p| root.join(p)).collect() };
        match self {
            Self::Local => pick(&[
                "synthetic/thousand-tx-block/short.json",
                "synthetic/max-calldata/short.json",
                "synthetic/precompile-fanout/short.json",
            ]),
            Self::PrGate => pick(&[
                "synthetic/thousand-tx-block/short.json",
                "synthetic/max-calldata/short.json",
                "synthetic/precompile-fanout/short.json",
                "synthetic/stylus-deep-call-stack/short.json",
                "synthetic/stylus-cold-cache/short.json",
                "synthetic/retryable-timeout-sweep/short.json",
            ]),
            Self::Nightly => pick(&[
                "synthetic/thousand-tx-block/short.json",
                "synthetic/max-calldata/short.json",
                "synthetic/precompile-fanout/short.json",
                "synthetic/stylus-deep-call-stack/short.json",
                "synthetic/stylus-cold-cache/short.json",
                "synthetic/retryable-timeout-sweep/short.json",
            ]),
        }
    }
}

fn main() -> eyre::Result<()> {
    let cli = Cli::parse();
    init_logging(&cli.log_level);
    match cli.command {
        Commands::Run(c) => run_cmd(c),
        Commands::Abba(c) => abba_cmd(c),
        Commands::Compare(c) => compare_cmd(c),
        Commands::Profile(c) => profile_cmd(c),
    }
}

fn init_logging(level: &str) {
    let filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new(level));
    fmt().with_env_filter(filter).with_target(false).init();
}

fn run_cmd(cmd: RunCommand) -> eyre::Result<()> {
    let manifests = expand_manifests(&cmd.manifests)?;
    if manifests.is_empty() {
        eyre::bail!("no manifests found");
    }
    std::fs::create_dir_all(&cmd.out)?;
    for (path, manifest) in &manifests {
        tracing::info!(manifest = %path.display(), mode = ?cmd.mode, "running");
        let workload = build_workload(manifest)?;
        let runner_cfg = RunnerConfig {
            rolling_window_blocks: cmd.window.unwrap_or(manifest.metrics.rolling_window_blocks),
            abort_on_block_error: cmd.abort_on_error,
        };
        let result = match cmd.mode {
            RunnerMode::InProcess => InProcessRunner::new(runner_cfg).run(workload)?,
            RunnerMode::Subprocess => {
                let safe = manifest.name.replace('/', "__");
                let data_dir = std::env::temp_dir()
                    .join("arbreth-bench-subprocess")
                    .join(format!("{safe}-{}", std::process::id()));
                if data_dir.exists() {
                    std::fs::remove_dir_all(&data_dir)?;
                }
                let sub = SubprocessConfig::new(
                    cmd.arbreth_binary.clone(),
                    cmd.genesis.clone(),
                    data_dir,
                );
                SubprocessRunner::new(runner_cfg, sub).run(workload)?
            }
        };
        let safe_name = manifest.name.replace('/', "__");
        let json_out = cmd.out.join(format!("{safe_name}.json"));
        let csv_out = cmd.out.join(format!("{safe_name}-blocks.csv"));
        let win_out = cmd.out.join(format!("{safe_name}-windows.csv"));
        report::json::write_run_result(&result, &json_out)?;
        report::csv::write_run_blocks_csv(&result, &csv_out)?;
        report::csv::write_run_windows_csv(&result, &win_out)?;
        let summary = markdown::render_run_summary(&result);
        println!("{summary}");
    }
    Ok(())
}

fn abba_cmd(cmd: AbbaCommand) -> eyre::Result<()> {
    let manifests_paths: Vec<PathBuf> = if let Some(p) = cmd.preset {
        p.manifests()
    } else {
        cmd.manifests.clone()
    };
    if manifests_paths.is_empty() {
        eyre::bail!("supply --manifests or --preset");
    }
    let manifests = expand_manifests(&manifests_paths)?;
    std::fs::create_dir_all(&cmd.out)?;
    let mut full_md = String::new();
    let mut any_regression = false;
    for (path, manifest) in &manifests {
        tracing::info!(manifest = %path.display(), "abba");
        let cfg = AbbaConfig {
            iterations: cmd.iterations,
            bootstrap_iters: cmd.bootstrap_iters,
            tolerance_pct: cmd.tolerance_pct,
            seed: 0xC0FF_EE12_3456_789A,
            runner: RunnerConfig {
                rolling_window_blocks: manifest.metrics.rolling_window_blocks,
                abort_on_block_error: false,
            },
        };
        let manifest_clone = manifest.clone();
        let manifest_clone_2 = manifest.clone();
        let result: AbbaResult = run_abba(
            &cfg,
            &manifest.name,
            move || build_workload(&manifest_clone),
            move || build_workload(&manifest_clone_2),
        )?;
        let safe_name = manifest.name.replace('/', "__");
        let json_out = cmd.out.join(format!("{safe_name}-abba.json"));
        let csv_out = cmd.out.join(format!("{safe_name}-abba.csv"));
        report::json::write_abba_result(&result, &json_out)?;
        report::csv::write_abba_deltas_csv(&result, &csv_out)?;
        let md = markdown::render_abba(&result);
        println!("{md}");
        full_md.push_str(&md);
        full_md.push('\n');
        if matches!(result.verdict, Verdict::Regression { .. }) {
            any_regression = true;
        }
    }
    if let Some(out) = cmd.markdown_out {
        if let Some(p) = out.parent() {
            std::fs::create_dir_all(p)?;
        }
        std::fs::write(&out, full_md)?;
    }
    if any_regression && cmd.fail_on_regression {
        eyre::bail!("at least one manifest regressed beyond tolerance");
    }
    Ok(())
}

fn compare_cmd(cmd: CompareCommand) -> eyre::Result<()> {
    let baseline: RunResult = report::json::read_run_result(&cmd.baseline)?;
    let feature: RunResult = report::json::read_run_result(&cmd.feature)?;
    let report: ComparisonReport = compare(
        &baseline,
        &feature,
        cmd.bootstrap_iters,
        cmd.tolerance_pct,
        cmd.seed,
    );
    let md = markdown::render_comparison(&report);
    println!("{md}");
    if let Some(out) = cmd.out {
        report::json::write_comparison(&report, &out)?;
    }
    if matches!(report.verdict, Verdict::Regression { .. }) {
        eyre::bail!("regression detected");
    }
    Ok(())
}

fn profile_cmd(cmd: ProfileCommand) -> eyre::Result<()> {
    let manifest = Manifest::from_path(&cmd.manifest)?;
    let mut workload = build_workload(&manifest)?;
    if let Some(n) = cmd.block_count_override {
        workload.blocks.truncate(n);
    }
    let mut runner = InProcessRunner::new(RunnerConfig::default());
    let result = runner.run(workload)?;
    println!("{}", markdown::render_run_summary(&result));
    Ok(())
}

fn expand_manifests(paths: &[PathBuf]) -> eyre::Result<Vec<(PathBuf, Manifest)>> {
    let mut out = Vec::new();
    for p in paths {
        let meta = std::fs::metadata(p).map_err(|e| eyre::eyre!("stat {}: {e}", p.display()))?;
        if meta.is_dir() {
            out.extend(Manifest::discover(p)?);
        } else {
            let m = Manifest::from_path(p)?;
            out.push((p.clone(), m));
        }
    }
    Ok(out)
}

fn build_workload(manifest: &Manifest) -> eyre::Result<Workload> {
    let mut workload = match &manifest.messages {
        MessageSource::SyntheticGenerator { generator, params } => generate(
            &manifest.name,
            manifest.chain_id,
            manifest.arbos_version,
            generator,
            params,
        )?,
    };
    if let Some(spec) = &manifest.prewarm {
        let balance = parse_wei(&spec.balance_wei)?;
        workload.prewarm_alloc = Some(arb_bench::runner::PrewarmAlloc {
            count: spec.accounts,
            seed: spec.seed,
            balance,
        });
    }
    Ok(workload)
}

fn parse_wei(s: &str) -> eyre::Result<alloy_primitives::U256> {
    if let Some(stripped) = s.strip_prefix("0x") {
        Ok(alloy_primitives::U256::from_str_radix(stripped, 16)?)
    } else {
        Ok(alloy_primitives::U256::from_str_radix(s, 10)?)
    }
}
