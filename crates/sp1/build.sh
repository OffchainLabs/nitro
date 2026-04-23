#!/usr/bin/env bash

set -ex

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
export TOP=$SCRIPT_DIR/../..

cd "$TOP"

# On macOS, point secp256k1-sys (and any other cc-rs crate) at the SP1-provided
# RISC-V toolchain. Without these, the host ar/ranlib produce an empty
# libsecp256k1.a (ELF objects can't be indexed by macOS ranlib) and linking
# the SP1 program fails with undefined rustsecp256k1_v0_10_0_* symbols. Linux
# hosts handle the cross-compile archive fine and don't need these overrides.
if [ "$(uname -s)" = "Darwin" ]; then
    export RISCV_GNU_TOOLCHAIN="$HOME/.sp1/riscv"
    export AR_riscv64im_unknown_none_elf="$HOME/.sp1/riscv/bin/riscv64-unknown-elf-ar"
    export RANLIB_riscv64im_unknown_none_elf="$HOME/.sp1/riscv/bin/riscv64-unknown-elf-ranlib"
fi

make -C "$SCRIPT_DIR" brotli
make -C "$SCRIPT_DIR" nitro-deps

rm -rf target/sp1
mkdir -p target/sp1
export OUTPUT_DIR=$TOP/target/sp1

cd "$SCRIPT_DIR"
# Bump SP1's maximum heap memory size
export SP1_ZKVM_MAX_MEMORY=1099511627776
# Build SP1 program and run bootloading process
cargo run --release -p sp1-builder -- --replay-wasm "$TOP/target/machines/latest/replay.wasm" --output-folder "$OUTPUT_DIR"
# Build the SP1 runner
cargo build --release -p sp1-runner

# Copy relevant files to target folder
cp "$TOP/target/elf-compilation/riscv64im-succinct-zkvm-elf/release/stylus-compiler-program" "$OUTPUT_DIR"
cp "$TOP/target/release/sp1-runner" "$OUTPUT_DIR"

echo "SP1 runner is successfully built!"
