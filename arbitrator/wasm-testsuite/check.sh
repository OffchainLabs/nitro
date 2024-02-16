#!/usr/bin/bash

# Copyright 2022, Offchain Labs, Inc.
# For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

rm -rf tests ../../contracts/test/prover/spec-proofs
mkdir -p tests/
mkdir -p ../../contracts/test/prover/spec-proofs/

for file in testsuite/*wast; do
    wast="${file##testsuite/}"
    json="tests/${wast%.wast}.json"
    wast2json $file -o $json 2>/dev/null
done

cargo build --release

for file in tests/*.json; do
    base="${file#tests/}"
    name="${base%.wasm}"
    target/release/wasm-testsuite $name &
done

wait
