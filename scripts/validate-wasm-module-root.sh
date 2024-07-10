#!/usr/bin/env bash
set -e

MACHINES_DIR=$1
PROVER=$2

for machine in "$MACHINES_DIR"/*/ ; do
    if [ -d "$machine" ]; then
        expectedWasmModuleRoot=$(cat "$machine/module-root.txt")
        actualWasmModuleRoot=$(cd "$machine" && "$PROVER" machine.wavm.br --print-wasmmoduleroot)
        if [ "$expectedWasmModuleRoot" != "$actualWasmModuleRoot" ]; then
            echo "Error: Expected module root $expectedWasmModuleRoot but found $actualWasmModuleRoot in $machine"
            exit 1
        fi
    fi
done