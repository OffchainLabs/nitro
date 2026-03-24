#!/usr/bin/env bash

set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export TOP=$SCRIPT_DIR/../..

cd "$TOP"

# Download RISC-V C toolchain if needed
if [ ! -d "$HOME/.sp1/riscv" ]; then
    # Force reinstallation of sp1up, so we can pickup latest sp1up updates for rv64im toolchain
    curl -L https://sp1up.succinct.xyz | bash
    "$HOME"/.sp1/bin/sp1up -c

    echo "Testing riscv64-unknown-elf-gcc..."
    "$HOME"/.sp1/riscv/bin/riscv64-unknown-elf-gcc --version
fi

# Build brotli for SP1
cp crates/sp1/brotli_cmake_patch.txt brotli/CMakeLists.txt
rm -rf target/build-sp1/brotli target/lib-sp1
mkdir -p target/build-sp1/brotli
cd target/build-sp1/brotli
cmake -DCMAKE_POLICY_VERSION_MINIMUM=3.5 \
  -DCMAKE_TRY_COMPILE_TARGET_TYPE=STATIC_LIBRARY \
  -DCMAKE_SYSTEM_NAME=Generic \
  -DCMAKE_C_COMPILER="$HOME"/.sp1/riscv/bin/riscv64-unknown-elf-gcc \
  -DCMAKE_C_FLAGS="-march=rv64im -mabi=lp64 -DBROTLI_BUILD_PORTABLE -mcmodel=medany -ffunction-sections -fdata-sections -fPIC" \
  -DCMAKE_AR="$HOME"/.sp1/riscv/bin/riscv64-unknown-elf-ar \
  -DCMAKE_RANLIB="$HOME"/.sp1/riscv/bin/riscv64-unknown-elf-ranlib \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INSTALL_PREFIX="$TOP"/target/lib-sp1 \
  -DBROTLI_DISABLE_TESTS=ON \
  "$TOP"/brotli
make
make install
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
