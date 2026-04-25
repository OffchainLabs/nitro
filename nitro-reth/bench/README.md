# bench/

Performance corpus for `arb-bench`. See `crates/arb-bench/README.md` for usage.

## Layout

- `corpus/synthetic/` — Manifests generated in-process by `arbreth-bench`.
  Self-contained; no network access required. Used by the local, PR-gate, and
  nightly flows. Each category has `short.json` / `medium.json` / etc. scales.
- `baselines/master/` — Recorded run results per master commit, auto-committed
  by `bench-nightly.yml`. Used for trend tracking and as the `compare` baseline.
