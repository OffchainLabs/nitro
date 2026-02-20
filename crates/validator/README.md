# Validator

A Rust-based validation server for Arbitrum Nitro that validates block state transitions using JIT-compiled WASM execution. It exposes a JSON-RPC 2.0 API compatible with the Go validation client.

## Generating test block inputs

Record block inputs from a system test to use as validation input:

```bash
go test -v -run "TestProgramStorage$" ./system_tests/... -count 1 -- \
  --recordBlockInputs.enable=true \
  --recordBlockInputs.WithBaseDir=target/ \
  --recordBlockInputs.WithTimestampDirEnabled=false \
  --recordBlockInputs.WithBlockIdInFileNameEnabled=false
```

This produces JSON files (e.g. `system_tests/target/TestProgramStorage/block_inputs.json`) containing `ValidationInput` data.

## Validation modes

The server supports two validation modes:

- **Native** (default) -- Runs the JIT machine in-process via the `jit` crate. Each validation request spawns a JIT execution inline.
- **Continuous** -- Spawns long-running JIT machine subprocesses (one per module root) at server startup. Validation requests are fed to these persistent processes over a TCP-based IPC channel. This avoids per-request process startup overhead.

## Building and running the server

The preferred way to build the validator is via the Makefile target:

```bash
make build-validation-server
```

This produces the binary at `target/bin/validator`. Run it with:

```bash
# Native mode (default)
RUST_LOG=tower_http=debug,info target/bin/validator

# Continuous mode
RUST_LOG=tower_http=debug,info target/bin/validator --mode continuous

# All options
RUST_LOG=tower_http=debug,info target/bin/validator \
  --mode native|continuous \
  --address 0.0.0.0:4141 \
  --workers 8 \
  --root-path /path/to/machines \
  --logging-format text|json
```

The `RUST_LOG` environment variable controls log verbosity via [tracing](https://docs.rs/tracing-subscriber/latest/tracing_subscriber/filter/struct.EnvFilter.html). `tower_http=debug` enables HTTP request/response logging, and `info` sets the default level for all other modules. Other useful values:

- `RUST_LOG=debug` -- verbose logging for all modules
- `RUST_LOG=validator=debug,info` -- debug logging for the validator crate only
- `RUST_LOG=tower_http=trace,debug` -- trace-level HTTP logging (includes headers and bodies)

Alternatively, during development you can build and run in one step:

```bash
cargo run --release -p validator -- --mode continuous
```

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `native` | Validation mode: `native` or `continuous` |
| `--address` | `0.0.0.0:4141` | Socket address to listen on |
| `--workers` | auto-detected | Number of worker threads |
| `--root-path` | auto-discovered | Root path to machine directories |
| `--logging-format` | `text` | Log format: `text` or `json` |

### Machine directory discovery

When `--root-path` is not provided, the server searches for machine directories in this order:

1. `<crate_dir>/../../target/machines`
2. `<cwd>/machines`
3. `<cwd>/target/machines`
4. `<binary_dir>/../machines`

Each machine directory must contain a `module-root.txt` file. A directory named `latest` is treated as the latest module root. Directories named `0x<hex>` are identified by their hex module root value.

## API endpoints

### GET `/validation_name`

Returns the validator name.

```bash
curl http://localhost:4141/validation_name
# Rust JIT validator
```

### GET `/validation_capacity`

Returns the number of available worker threads.

```bash
curl http://localhost:4141/validation_capacity
# 4
```

### GET `/validation_wasmModuleRoots`

Returns the list of available WASM module roots.

```bash
curl http://localhost:4141/validation_wasmModuleRoots
# [0xabcd..., 0x1234...]
```

### GET `/validation_stylusArchs`

Returns the supported Stylus architecture target (`arm64`, `amd64`, or `host`).

```bash
curl http://localhost:4141/validation_stylusArchs
# host
```

### POST `/validation_validate`

Performs block validation. Accepts a JSON-RPC 2.0 request where `params` is an array containing:

1. A `ValidationInput` object (required)
2. A module root hex string (optional)

Without module root (uses `latest`)
```bash
curl -s -X POST http://localhost:4141/validation_validate \
  -H "Content-Type: application/json" \
  -d "$(jq -n --slurpfile input system_tests/target/TestProgramStorage/block_inputs.json \
    '{jsonrpc: "2.0", id: 1, method: "validation_validate", params: [$input[0]]}')"
```

With explicit module root
```bash
curl -s -X POST http://localhost:4141/validation_validate \
  -H "Content-Type: application/json" \
  -d "$(jq -n --slurpfile input system_tests/target/TestProgramStorage/block_inputs.json \
    '{jsonrpc: "2.0", id: 1, method: "validation_validate", params: [$input[0], "0xYourModuleRootHere"]}')"
```

**Success response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "BlockHash": "0x...",
    "SendRoot": "0x...",
    "Batch": 1,
    "PosInBatch": 0
  }
}
```

**Error response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32000,
    "message": "error description"
  }
}
```

### Module root behavior

When the module root is **omitted** from the request, the server falls back to the latest available module root discovered at startup. When **provided**, it looks up the corresponding machine directory by hex value and returns an error if not found.
