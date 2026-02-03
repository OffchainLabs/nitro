#!/usr/bin/env bash

set -euo pipefail

rm -rf target/machines/latest/replay.*
rm -rf target/machines/latest/machine*

make build-prover-bin > /dev/null 2>&1
make build-replay-env > /dev/null 2>&1

for test in TestGenerateLightBlock TestGenerateHeavyBlock; do
  gotestsum --format short-verbose -- -run ^$test$ ./system_tests/... --count 1 -- \
    --recordBlockInputs.enable=true --recordBlockInputs.WithBaseDir=../target \
    --recordBlockInputs.WithTimestampDirEnabled=false --recordBlockInputs.WithBlockIdInFileNameEnabled=false \
    > /dev/null 2>&1
done

for test in TestGenerateLightBlock TestGenerateHeavyBlock; do
  cargo run --release --bin prover -- target/machines/latest/machine.wavm.br -p --profile-sum-opcodes --profile-sum-funcs \
    --json-inputs="./target/$test/block_inputs.json" 2> /dev/null | grep -E "WASM says|Total cycles"
done
