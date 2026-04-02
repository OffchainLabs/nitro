#!/usr/bin/env bash

set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export TOP=$SCRIPT_DIR/../..

cd "$TOP"

# Install sp1up if not present
if [ ! -f "$HOME/.sp1/bin/sp1up" ]; then
    curl -L https://sp1up.succinct.xyz | bash
fi

# Install SP1 Rust toolchain (succinct) if not present
if ! rustup toolchain list 2>/dev/null | grep -q succinct; then
    "$HOME"/.sp1/bin/sp1up -v v6.0.0
fi

# Download RISC-V C toolchain if needed
if [ ! -d "$HOME/.sp1/riscv" ]; then
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

# Later on, there will be much more to do. For now it's enough to build just the stylus compiler runner.
cargo build --release -p stylus-compiler-runner
