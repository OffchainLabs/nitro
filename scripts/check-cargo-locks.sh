#!/usr/bin/env bash
# Checks that every checked-in Cargo.lock still matches its manifests.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

status=0
checked=0

# git ls-files rather than find: it lists only tracked lockfiles, without submodules
while IFS= read -r lockfile; do
    manifest="${lockfile%Cargo.lock}Cargo.toml"
    if [ ! -f "$manifest" ]; then
        echo "no manifest for ${lockfile}; either delete the lockfile or add ${manifest}"
        status=1
        continue
    fi
    checked=$((checked + 1))
    if cargo metadata --locked --format-version 1 --manifest-path "$manifest" >/dev/null; then
        echo "ok       ${lockfile}"
    else
        echo "outdated ${lockfile}"
        echo "         regenerate with: cargo metadata --manifest-path ${manifest} >/dev/null"
        status=1
    fi
done < <(git ls-files '*Cargo.lock')

# Guards against the discovery silently finding nothing, which would otherwise
# look identical to every lockfile being in sync.
if [ "$checked" -eq 0 ]; then
    echo "found no Cargo.lock files to check, which should never happen"
    exit 1
fi

if [ "$status" -ne 0 ]; then
    echo
    echo "Some Cargo.lock files are out of date. Regenerate them as shown above and commit the result."
fi
exit "$status"
