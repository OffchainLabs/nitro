# arb-bench

Performance benchmarking framework for arbreth. Measures whether a code change
makes the node faster, slower, or neither — locally during development and
authoritatively in CI.

## Quick start

### Local microbenches (~30s)

```bash
cargo bench -p arb-bench --bench block_executor
cargo bench -p arb-bench --bench precompile_dispatch
cargo bench -p arb-bench --bench tx_decode
```

### Local ABBA (within-run baseline-vs-feature, ~2 min)

```bash
cargo run -p arb-bench --release --bin arbreth-bench -- abba \
  --preset local --iterations 2
```

`abba` runs your current branch as the "feature" and the last commit on
`master` as the "baseline" by alternating in an A-B-B-A pattern on identical
workloads. Output is a paired bootstrap 95% CI on the perf delta — robust to
laptop noise.

### Single workload run

```bash
cargo run -p arb-bench --release --bin arbreth-bench -- run \
  bench/corpus/synthetic/precompile-fanout/short.json
```

Writes `bench/baselines/local/synthetic__precompile-fanout__short.json`
(per-block + summary metrics) and matching CSVs.

### Compare two saved runs

```bash
arbreth-bench compare \
  --baseline bench/baselines/master/2026-04-24-abc123.json \
  --feature  bench/baselines/local/synthetic__precompile-fanout__short.json
```

### Capture from Sepolia

```bash
arbreth-bench capture \
  --rpc https://sepolia-rollup.arbitrum.io/rpc \
  --from 50000000 --to 50001000 \
  --out raw.json
arbreth-bench curate --input raw.json --out staging
```

## Architecture

- **Workload sources** — synthetic generators (in-process, no network).
- **Runner** — `InProcessRunner` drives `ArbBlockExecutor` directly. Engine-RPC
  helpers are provided for nightly validation against a running node.
- **ABBA scheduler** — `runner::abba::run_abba()` interleaves baseline and
  feature on identical workloads.
- **Bootstrap CI** — `report::compare::bootstrap_paired_delta()` produces
  paired-bootstrap 95% confidence intervals; verdict logic in
  `report::compare::compare()` and `runner::abba::decide_verdict`.
- **Rolling windows** — `metrics::rolling::build_windows()` for long-run flush /
  pruner / fragmentation visibility.

## Manifest schema

```json
{
  "name": "synthetic/thousand-tx-block/short",
  "category": "thousand-tx-block",
  "scale": "short",
  "arbos_version": 60,
  "chain_id": 421614,
  "corpus_version": "1.0.0",
  "messages": {
    "source": "synthetic_generator",
    "generator": "thousand_tx_block",
    "params": { "block_count": 5, "txs_per_block": 1000 }
  },
  "metrics": { "rolling_window_blocks": 1 },
  "regression": { "gate": true, "tolerance_pct": 5.0 }
}
```

`source` may be `synthetic_generator`, `frozen` (tarball), or `fresh_rpc`.

## CI workflows

- `.github/workflows/bench-pr.yml` — PR gate, ABBA × 3 over `pr-gate` preset
- `.github/workflows/bench-nightly.yml` — full short+medium matrix, daily
- `.github/workflows/bench-soak.yml` — weekly endurance with rolling-window
  monotonic-slowdown / RSS-growth detection
- `.github/workflows/bench-release-gate.yml` — manual full-matrix sweep before
  releases
