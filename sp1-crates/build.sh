#!/usr/bin/env bash

set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export TOP=$SCRIPT_DIR/..

cd "$TOP"
# Build nitro dependencies
make build-replay-env test-go-deps
rm -rf target/sp1
mkdir -p target/sp1
export OUTPUT_DIR=$TOP/target/sp1
# Build replay.wasm, but with SP1 optimizations
GOOS=wasip1 GOARCH=wasm go build -tags sp1 -o "$OUTPUT_DIR"/replay.wasm "$TOP"/cmd/replay/...
# Build a sample Arbitrum test block
rm -rf system_tests/test-data
go test -run TestProgramStorage ./system_tests/ -- \
    -recordBlockInputs.WithBaseDir="$(pwd)"/system_tests/test-data \
    -recordBlockInputs.WithTimestampDirEnabled=false \
    -recordBlockInputs.enable=true
cp system_tests/test-data/TestProgramStorage/*.json target/sp1/

cd "$SCRIPT_DIR"
# Bump SP1's maximum heap memory size
export SP1_ZKVM_MAX_MEMORY=1099511627776
# Build SP1 program and run bootloading process
cargo run --release -p sp1-builder -- --replay-wasm "$OUTPUT_DIR"/replay.wasm --output-folder "$OUTPUT_DIR"
# Build the SP1 runner
cargo build --release -p sp1-runner

# Copy relavant files to target folder
cp target/elf-compilation/riscv64im-succinct-zkvm-elf/release/stylus-compiler-program "$OUTPUT_DIR"
cp target/release/sp1-runner "$OUTPUT_DIR"

echo "SP1 runner is successfully built!"
