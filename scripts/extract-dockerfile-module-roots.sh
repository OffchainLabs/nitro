#!/usr/bin/env bash
# Emits one "NAME HASH" pair per uncommented `RUN ./download-machine.sh NAME HASH ...`
# line found in the given Dockerfile (default: ./Dockerfile).
#
# Exits non-zero if no active invocations are found, so a silently-weakened CI
# job (e.g. caused by a Dockerfile format change) fails loudly instead of
# validating against zero module roots.

set -euo pipefail

DOCKERFILE="${1:-Dockerfile}"

if [ ! -f "$DOCKERFILE" ]; then
    echo "Dockerfile not found: $DOCKERFILE" >&2
    exit 1
fi

matches=$(grep -E '^RUN[[:space:]]+\./download-machine\.sh[[:space:]]+' "$DOCKERFILE" || true)
if [ -z "$matches" ]; then
    echo "no active download-machine.sh invocations found in $DOCKERFILE" >&2
    exit 1
fi

echo "$matches" | awk '{print $3, $4}'
