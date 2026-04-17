#!/usr/bin/env bash
# Runs the Rust validation server against the block input JSON in Native and
# Continuous modes, validating against every module root listed in the
# Dockerfile plus the `latest` built in target/machines/.
#
# Usage: validation.sh <INPUT_FILE> <JIT_HASH>
#   INPUT_FILE — path to the block inputs JSON produced earlier in the job.
#   JIT_HASH   — expected block hash from the JIT prover run on the same JSON.

set -eo pipefail

INPUT_FILE=$1
JIT_HASH=$2

if [ -z "$INPUT_FILE" ] || [ -z "$JIT_HASH" ]; then
  echo "usage: $0 <INPUT_FILE> <JIT_HASH>" >&2
  exit 1
fi

# --- Download every module root the Dockerfile currently supports ----------
# The validation server discovers all of them under target/machines/ at
# startup and we can address each one by hash below.
# Capture into a variable (not process substitution) so a non-zero exit from
MODULE_ROOTS=$(scripts/extract-dockerfile-module-roots.sh Dockerfile)
NAMES=()
HASHES=()
while IFS=' ' read -r NAME HASH; do
  echo "Downloading $NAME ($HASH)..."
  scripts/download-machine.sh "$NAME" "$HASH"
  mv "$HASH" target/machines/
  # download-machine.sh creates a `latest` symlink in cwd; remove it so the
  # target/machines/latest built by `make build-replay-env` stays intact.
  rm -f latest
  NAMES+=("$NAME")
  HASHES+=("$HASH")
done <<< "$MODULE_ROOTS"

# Refuse to run a degraded validation that only checks `latest`. If HASHES is
# empty here, something went wrong upstream (e.g. the download step silently
# produced no hashes) and the multi-root coverage this job is supposed to
# provide would be lost.
if [ "${#HASHES[@]}" -eq 0 ]; then
  echo "❌ No module root hashes — refusing to run degraded validation"
  exit 1
fi

# --- Validation helpers ----------------------------------------------------

send_validate() {
  local ID=$1
  local MODULE_ROOT=$2
  if [ -z "$MODULE_ROOT" ]; then
    jq -n --slurpfile input "$INPUT_FILE" --argjson id "$ID" \
      '{jsonrpc: "2.0", id: $id, method: "validation_validate", params: [$input[0]]}'
  else
    jq -n --slurpfile input "$INPUT_FILE" --argjson id "$ID" --arg mr "$MODULE_ROOT" \
      '{jsonrpc: "2.0", id: $id, method: "validation_validate", params: [$input[0], $mr]}'
  fi | curl -s -X POST http://localhost:4141/ \
    -H "Content-Type: application/json" -d @-
}

check_response() {
  local LABEL=$1
  local EXPECTED_ID=$2
  local RESPONSE=$3

  local RESP_ID SERVER_HASH
  RESP_ID=$(echo "$RESPONSE" | jq -r '.id')
  if [ "$RESP_ID" != "$EXPECTED_ID" ]; then
    echo "❌ $LABEL JSON-RPC id mismatch: expected $EXPECTED_ID, got $RESP_ID"
    echo "   response: $RESPONSE"
    return 1
  fi

  SERVER_HASH=$(echo "$RESPONSE" | jq -r '.result.BlockHash')
  if [ "$JIT_HASH" != "$SERVER_HASH" ]; then
    echo "❌ $LABEL hash mismatch: JIT=$JIT_HASH server=$SERVER_HASH"
    echo "   response: $RESPONSE"
    return 1
  fi

  echo "✅ $LABEL matches JIT ($SERVER_HASH)"
}

validate_mode() {
  local MODE_NAME=$1
  local SERVER_ARGS=$2

  echo "::group::Testing Mode: $MODE_NAME"

  # 1. Start server in background, capturing output so a crash surfaces the
  # actual error instead of just a hash/id mismatch.
  local LOG_FILE="/tmp/validator-$MODE_NAME.log"
  echo "Starting server ($MODE_NAME)..."
  target/bin/validator $SERVER_ARGS > "$LOG_FILE" 2>&1 &
  SERVER_PID=$!

  # 2. Wait for server to respond (up to 2 minutes)
  echo "Waiting for validator ($MODE_NAME) to start..."
  if ! timeout 120s bash -c 'until curl -s localhost:4141 > /dev/null; do sleep 1; done'; then
    echo "❌ Server ($MODE_NAME) failed to start within 2 min timeout"
    echo "::group::Validator log ($MODE_NAME)"
    cat "$LOG_FILE" || true
    echo "::endgroup::"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
  fi

  # 3. Validate once against the default (latest) module root, then once per
  # Dockerfile-listed module root.
  local EXIT_CODE=0
  local ID=1
  local RESPONSE
  RESPONSE=$(send_validate "$ID" "")
  check_response "$MODE_NAME latest" "$ID" "$RESPONSE" || EXIT_CODE=1

  for i in "${!HASHES[@]}"; do
    local MODULE_NAME="${NAMES[$i]}"
    local MODULE_ROOT="${HASHES[$i]}"
    ID=$((ID + 1))
    RESPONSE=$(send_validate "$ID" "$MODULE_ROOT")
    check_response "$MODULE_NAME $MODE_NAME $MODULE_ROOT" "$ID" "$RESPONSE" || EXIT_CODE=1
  done

  # 4. Stop the server
  kill $SERVER_PID
  wait $SERVER_PID 2>/dev/null || true

  if [ "$EXIT_CODE" -ne 0 ]; then
    echo "::group::Validator log ($MODE_NAME)"
    cat "$LOG_FILE" || true
    echo "::endgroup::"
    echo "❌ $MODE_NAME Validation failed"
    exit "$EXIT_CODE"
  fi
  echo "✅ $MODE_NAME Validation successful"
  echo "::endgroup::"
}

# --- Run tests -------------------------------------------------------------

validate_mode "Native" ""
validate_mode "Continuous" "--mode continuous"
