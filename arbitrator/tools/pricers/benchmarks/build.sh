#!/usr/bin/env bash
set -eu
cd "$(dirname "$0")"
mkdir -p wasm-benchmarks
for file in src/bin/*.rs; do
	test="$(basename "$file")"
	test="${test%.rs}"
	cargo run --bin "$test" > "wasm-benchmarks/$test.wat"
	#wat2wasm "wasm-benchmarks/$test.wat" -o "wasm-benchmarks/$test.wasm"
done
