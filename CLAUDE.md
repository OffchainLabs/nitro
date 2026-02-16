# Nitro

Arbitrum Nitro is a Layer 2 optimistic rollup for Ethereum. The node is primarily written in Go, with fraud-proof infrastructure (prover, JIT, Stylus) in Rust, and on-chain contracts in Solidity.

## Build

Requires: Go 1.25+, Rust 1.88+, Node/Yarn, wasi-sdk, wabt (wat2wasm), Foundry. After clone: `git submodule update --init --recursive`.

```sh
make build               # Go binaries -> target/bin/
make build-node-deps     # prerequisite for most Go work (prover header/lib, JIT, solgen, cbrotli)
make build-replay-env    # prover, JIT, WASM libs, replay.wasm
make build-solidity      # Hardhat + Foundry contracts
make contracts           # regenerate Go bindings from Solidity ABIs (solgen/go/)
```

## Test

**Prerequisites**: `make test-go-deps` must run first -- it builds WASM artifacts, stylus test wasms, and replay environment. Without it, tests fail with missing file errors.

```sh
make test-go                 # standard Go tests via gotestsum
make test-go-challenge       # --tags challengetest
make test-go-stylus          # --tags stylustest
make test-go-redis           # requires Redis at localhost:6379
make test-rust               # cargo test --release
make tests                   # test-go + test-rust
```

To run a single Go test:

```sh
go test -run TestName ./package/...
```

Unit tests are colocated with source. System/integration tests live in `system_tests/` -- these spin up actual node instances and are heavyweight. Prefer unit tests for logic that doesn't require a running node.

## Pre-push check

```sh
make push    # lint + test-go + fmt -- run before pushing
```

## Lint and format

```sh
make lint      # custom linters (go run ./linters ./...) + golangci-lint + solhint
make fmt       # golangci-lint fmt + cargo fmt + forge fmt
```

### Custom Go linters (`linters/`)

Run via `go run ./linters ./...`. These are project-specific static analyzers:

- **structinit** / **namedfieldsinit** -- structs annotated with `// lint:require-exhaustive-initialization` must have all fields set with named fields at every instantiation site
- **koanf** -- validates koanf config struct usage
- **pointercheck**, **rightshift**, **jsonneverempty** -- additional safety checks (see linter error messages for details)

### License headers

Required on all `.go`, `.rs`, and `Makefile` files: `// Copyright YEAR[-YEAR], Offchain Labs, Inc.` Check with `make check-license-headers`.

## Architecture

Consensus and execution are separated at an RPC boundary. The interface contracts live in `execution/interface.go`, `consensus/interface.go`, and `validator/interface.go` -- these define how the layers communicate and can run either in-process or via RPC.

Data flow: L1 batches -> InboxReader -> TransactionStreamer -> ExecutionEngine -> L2 blocks. The sequencer path: user txs -> Sequencer -> TransactionStreamer -> batch accumulation -> BatchPoster -> L1 SequencerInbox.

- **Consensus layer** (`arbnode/`): InboxReader, TransactionStreamer, BatchPoster, SeqCoordinator, BlockValidator, Staker, BroadcastServer
- **Execution layer** (`execution/gethexec/`): ExecutionEngine, Sequencer, ArbOS (L2 state machine)
- **Validation** (`validator/`, `staker/`): BlockValidator replays blocks through Rust JIT/prover; Staker posts assertions to L1
- **ArbOS** (`arbos/`): L2 operating system -- block processing, fee accounting, retryables, precompiles, Stylus
- **Fraud proofs**: Go replay binary compiled to WASM (`cmd/replay/`), processed by Rust prover (`crates/prover/`) for one-step proof generation

### go-ethereum fork

The `go-ethereum/` submodule is a **fork** of geth, replaced in `go.mod`. The node runs on this fork, not upstream geth. Any changes to geth-level behavior (EVM, p2p, RPC, state) must go in the `go-ethereum/` submodule.

### Go-Rust boundary

- **CGo FFI**: Stylus (`crates/stylus/`) compiles to `libstylus.a`, called from Go via `target/include/arbitrator.h`
- **Process-level**: JIT (`crates/jit/`) and validator (`crates/validator/`) run as separate processes, communicating via sockets

### Submodules

Key submodules: `go-ethereum` (forked geth), `brotli`, `contracts`, `contracts-legacy`, `safe-smart-account`, `nitro-testnode`, Rust SDK/lang crates (`crates/langs/`), `wasmer` fork (`crates/tools/wasmer/`), soft-float, wasm-testsuite.

## Code conventions

### Go

- **Logger**: `github.com/ethereum/go-ethereum/log` (structured key-value: `log.Info("msg", "key", val)`)
- **Config**: koanf + pflag. Each component has a `Config` struct with `koanf:"kebab-case"` tags (map to CLI flags like `--node.sequencer`), `ConfigDefault`, `ConfigAddOptions(prefix, flagset)`, `Validate() error`. Fields tagged `reload:"hot"` support live reloading.
- **Lifecycle**: components embed `util/stopwaiter` and implement `Start(ctx)` / `StopAndWait()`
- **Import order** (enforced by gci): stdlib, third-party, `github.com/ethereum/go-ethereum`, `github.com/offchainlabs`
- **Metrics**: `go-ethereum/metrics` with `arb/` prefix namespace (e.g. `arb/batchposter/wallet/eth`)
- **Errors**: wrap with `fmt.Errorf("context: %w", err)`. Config structs validate via `.Validate()` methods.

### Rust

- Workspace at repo root (`Cargo.toml`), default members: `crates/jit`, `crates/prover`
- Formatting: `cargo fmt -p arbutil -p prover -p jit -p stylus`
- Key crates: `prover` (WAVM emulator), `jit` (Wasmer-based execution), `stylus` (WASM smart contracts), `validator` (validation server)
- WASM libraries in `crates/wasm-libraries/` target `wasm32-wasip1` or `wasm32-unknown-unknown`

### Solidity

- Three contract directories: `contracts/` (current), `contracts-legacy/`, `contracts-local/`
- Tooling: Hardhat + Foundry, Solhint for linting, `forge fmt` for formatting
- Go bindings are auto-generated: edit Solidity -> `make build-solidity` -> `make contracts` -> bindings regenerate in `solgen/go/` (gitignored, never edit directly)

## PR requirements

- Add a changelog fragment to `changelog/` (keepachangelog format: `### Added`, `### Changed`, `### Fixed`). Use `### Ignored` for non-noteworthy changes (CI, deps). Filename convention: `<author>-<ticket>.md`
- CI validates the changelog via `unclog`
- Branch naming: `<author>/<ticket>-<description>`
