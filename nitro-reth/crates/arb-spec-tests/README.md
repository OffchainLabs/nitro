# arb-spec-tests

Consensus-canary fixtures for arbreth. Each JSON file under `fixtures/`
encodes either a pure ArbOS state mutation (e.g. add a chain owner, set
an L2 pricing parameter and assert the post-state) or a full L1->L2
message stream that must produce identical receipts, state diffs, and
storage values when replayed against arbreth and a reference Nitro
node.

## Layout

- `fixtures/pricing/`, `state_transitions/`, `retryables/`,
  `l1_pricing_dynamics/`, `address_handling/`, `merkle/`,
  `version_transitions/` — pure ArbOS state fixtures. Run in-process
  against the `arbos` crate without any external process.
- `fixtures/execution/`, `arbos/`, `stylus/`, `retryables/*.json`
  containing `"messages"` — execution fixtures. Need either:
  - a running arbreth process the harness can talk to over RPC, or
  - a release `arb-reth` binary the harness spawns per fixture against
    the inline genesis declared in the JSON.

## What `cargo test -p arb-spec-tests` runs by default

Only the pure-ArbOS-state fixtures (~24 fixtures, no external process).
Everything that needs to drive an arbreth node is feature-gated and
shows up as `ignored` in the test output. From the workspace root:

```
cargo test --workspace
```

prints a non-zero `ignored` count for the binary-driven tests so they
are visible (no silent skip).

## Running the full suite (binary-driven fixtures)

Build the release binary and run with `--features spec-binary` plus
`ARB_SPEC_BINARY`:

```
cargo build --release -p arb-reth --bin arb-reth
ARB_SPEC_BINARY=$PWD/target/release/arb-reth \
    cargo test -p arb-spec-tests --features spec-binary
```

If the path in `ARB_SPEC_BINARY` does not exist, or the spawn fails,
the affected test panics with a clear message instead of silently
skipping.

## Environment variables

- `ARB_SPEC_BINARY` — path to a release `arb-reth` binary. Required
  for fixtures that ship inline genesis (the harness spawns a fresh
  node per fixture against a free port).
- `ARB_SPEC_RPC_URL` — JSON-RPC URL of an already-running node. Used
  for fixtures that have no inline genesis and just need to replay
  against a static chain (e.g. a Sepolia archive).
- `ARB_SPEC_REQUIRE_BINARY` — when set to any value, panics if the
  crate was built without `--features spec-binary`. CI uses this to
  guarantee the binary-driven coverage actually runs rather than being
  silently filtered out.
- `ARB_SPEC_MODE` — `verify` (default), `record`, or `compare`. The
  `record` mode rewrites the fixture JSON from a fresh capture.
- `ARB_SPEC_FILTER` — substring filter; only fixture paths containing
  this string are exercised.
- `ARB_SPEC_INCLUDE_PENDING` — when set, also run `pending_*.json`
  fixtures that reproduce known-unsolved divergences.
- `ARB_SPEC_RUST_LOG`, `STYLUS_HOSTIO_TRACE`, `STYLUS_DEBUG` —
  forwarded to spawned arbreth processes for debugging.
- `ARB_SPEC_STARTUP_TIMEOUT` — seconds to wait for the spawned node's
  RPC to come up. Defaults to 90.
- `ARB_SPEC_KEEP_WORKDIR` — when set, the per-fixture temp datadir is
  left in place after the test for inspection.

## CI

`.github/workflows/spec-tests.yml` builds `arb-reth --release`,
uploads it as an artifact, and invokes the spec suite with both
`--features spec-binary` and `ARB_SPEC_REQUIRE_BINARY=1` so that any
missing-binary regression fails loudly instead of pruning coverage.
