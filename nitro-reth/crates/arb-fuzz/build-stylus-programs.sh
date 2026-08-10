#!/usr/bin/env bash
# Regenerate the prebuilt Stylus initcode hex files from the Rust sources
# under `stylus-programs/`. Requires `cargo-stylus` (`cargo install
# cargo-stylus`) and the `wasm32-unknown-unknown` rustup target.
#
# Run from the arb-fuzz crate root:
#   ./build-stylus-programs.sh
#
# Each program lives in its own Cargo project under `stylus-programs/<name>/`.
# The crates are intentionally outside the main workspace because they target
# `wasm32-unknown-unknown` and shouldn't pollute `cargo check` on the host.

set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
PROGRAMS=(counter erc20_mini sol_caller storage_stress)

mkdir -p "$HERE/prebuilt"

for name in "${PROGRAMS[@]}"; do
    src="$HERE/stylus-programs/$name"
    out="$HERE/prebuilt/$name.hex"
    echo "[stylus-programs] building $name -> $out"
    pushd "$src" >/dev/null
    cargo stylus get-initcode --output "$out"
    popd >/dev/null
done

echo "[stylus-programs] done. Initcode hex files in $HERE/prebuilt/"
