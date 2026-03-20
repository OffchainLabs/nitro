#!/bin/bash
set -euo pipefail

# Poll the nitro RPC endpoint, exiting 0 on success.
# Exits early if the testnode process dies, or times out after 30 minutes.
# Dumps testnode logs on any failure.
timeout_time=$(($(date +%s) + 1800))

dump_logs() {
  if [ -f /tmp/testnode.log ]; then
    echo "=== Last 200 lines of testnode.log ==="
    tail -200 /tmp/testnode.log || echo "WARNING: could not read /tmp/testnode.log"
  else
    echo "WARNING: /tmp/testnode.log does not exist -- no logs available"
  fi
}

die() {
  echo "ERROR: $1"
  exit 1
}

if [ ! -f /tmp/testnode.pid ]; then
  die "/tmp/testnode.pid not found -- cannot monitor testnode process"
fi

TESTNODE_PID=$(cat /tmp/testnode.pid)
if ! [[ "$TESTNODE_PID" =~ ^[0-9]+$ ]]; then
  die "/tmp/testnode.pid contains invalid PID: '$TESTNODE_PID'"
fi
echo "Monitoring testnode PID: $TESTNODE_PID"

cleanup() {
  dump_logs
  if kill -0 "$TESTNODE_PID" 2>/dev/null; then
    echo "Cleaning up testnode process $TESTNODE_PID"
    kill "$TESTNODE_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

attempt=0
while (( $(date +%s) <= timeout_time )); do
  if ! kill -0 "$TESTNODE_PID" 2>/dev/null; then
    die "testnode process died unexpectedly"
  fi
  http_code=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":45678,"method":"eth_chainId","params":[]}' \
    'http://localhost:8547' 2>/dev/null) || http_code="000"
  if [ "$http_code" = "200" ]; then
    trap - EXIT
    exit 0
  fi
  attempt=$((attempt + 1))
  if (( attempt % 5 == 0 )); then
    echo "Attempt $attempt: HTTP $http_code, retrying..."
  fi
  sleep 20
done

echo "=== Final curl attempt (verbose) ==="
curl -v -X POST -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":45678,"method":"eth_chainId","params":[]}' \
  'http://localhost:8547' 2>&1 || true
die "timed out waiting for nitro RPC"
