#!/usr/bin/env bash
# Checks that every checked-in Cargo.lock still matches its manifests.

set -euo pipefail

workspaces=(
    "."
    "crates/sp1"
    "crates/stylus/tests"
    "crates/tools/module_roots"
    "crates/tools/stylus_benchmark"
    "crates/wasm-testsuite"
    "crates/prover/test-cases/rust"
    "nitro-reth"
)

status=0
for workspace in "${workspaces[@]}"; do
    manifest="${workspace}/Cargo.toml"
    if cargo metadata --locked --format-version 1 --manifest-path "$manifest" >/dev/null; then
        echo "ok       ${workspace}/Cargo.lock"
    else
        echo "outdated ${workspace}/Cargo.lock"
        echo "         regenerate with: cargo metadata --manifest-path ${manifest} >/dev/null"
        status=1
    fi
done

if [ "$status" -ne 0 ]; then
    echo
    echo "Some Cargo.lock files are out of date. Regenerate them as shown above and commit the result."
fi
exit "$status"
