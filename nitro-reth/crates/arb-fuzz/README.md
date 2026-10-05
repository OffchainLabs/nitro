# Fuzz targets

Two families of targets live here:

- **Decoder targets** — fast, in-process; no external dependencies.
- **Differential targets** — drive `arbreth` and the Nitro reference container side-by-side via the `arb-test-harness` `DualExec`. The first iteration spawns a Docker container, a mock L1, and a local arbreth process; subsequent iterations reuse them.

## Prerequisites

```bash
rustup toolchain install nightly
cargo install cargo-fuzz
```

The differential targets additionally require a runnable `arbreth` binary on `$PATH` (the harness reads `ARB_BINARY` to locate it) and Docker to host the Nitro reference. Builds do not need either.

## Targets

| Target | Family | Surface |
|---|---|---|
| `parse_l2_transactions` | decode | L1->L2 message parser (all message kinds) |
| `decode_arb_receipt` | decode | EIP-2718 ArbReceipt decode |
| `decode_arb_tx_envelope` | decode | ArbTxEnvelope (all tx variants) decode |
| `decode_internal_tx` | decode | startBlock internal-tx ABI decode |
| `diff_precompile` | differential | precompile invocations (0x64..=0x74) across ArbOS versions |
| `diff_tx` | differential | single arbitrary user tx vs reference |
| `diff_stylus_wasm` | differential | wasm-smith program deploy + invoke |
| `property_invariants` | differential | seed-driven multi-tx mix |

## Build

`cargo-fuzz` expects `fuzz/Cargo.toml` by default; this crate IS the fuzz crate, so pass `--fuzz-dir .`:

```bash
cd crates/arb-fuzz

# All targets:
cargo +nightly fuzz build --fuzz-dir .

# Single target:
cargo +nightly fuzz build diff_precompile --fuzz-dir .
```

## Run

60-second smoke budgets for each target:

```bash
cd crates/arb-fuzz

cargo +nightly fuzz run parse_l2_transactions    --fuzz-dir . -- -max_total_time=60
cargo +nightly fuzz run decode_arb_receipt       --fuzz-dir . -- -max_total_time=60
cargo +nightly fuzz run decode_arb_tx_envelope   --fuzz-dir . -- -max_total_time=60
cargo +nightly fuzz run decode_internal_tx       --fuzz-dir . -- -max_total_time=60

cargo +nightly fuzz run diff_precompile          --fuzz-dir . -- -max_total_time=60
cargo +nightly fuzz run diff_tx                  --fuzz-dir . -- -max_total_time=60
cargo +nightly fuzz run diff_stylus_wasm         --fuzz-dir . -- -max_total_time=60
cargo +nightly fuzz run property_invariants      --fuzz-dir . -- -max_total_time=60
```

With a corpus directory:

```bash
cargo +nightly fuzz run parse_l2_transactions --fuzz-dir . corpus/parse_l2_transactions
```

Reproduce a crash from a saved input:

```bash
cargo +nightly fuzz run parse_l2_transactions --fuzz-dir . corpus/parse_l2_transactions/<crash-input>
```

## Crash output

Differential targets, on a non-clean `DiffReport`, write the `(input, report)` pair as JSON under `crates/arb-spec-tests/fixtures/_captured/fuzz_crash_<hash>.json` so it can be replayed as a regression fixture. `dump_crash_as_fixture` falls back to `std::env::temp_dir()` if writing under the workspace fails.

## Build artifacts

Output lands in the workspace `target/` (gitignored). Nothing here links into the production node binary.
