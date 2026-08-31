#!/bin/bash

set -euo pipefail

if [ -n "${PRIVATE_REPO_TOKEN:-}" ]; then
  GH_TOKEN="$PRIVATE_REPO_TOKEN" DOCKER_BUILDKIT=1 docker build --secret id=gh_token,env=GH_TOKEN --target nitro-node-dev --tag nitro-local-build .
else
  # No private token (e.g. public nitro): the consensus machine is fetched from public releases.
  DOCKER_BUILDKIT=1 docker build --target nitro-node-dev --tag nitro-local-build .
fi

# Clone nitro-devnode repo
git clone https://github.com/OffchainLabs/nitro-devnode.git
cd nitro-devnode

# Start nitro-devnode in background
TARGET_IMAGE=nitro-local-build ./run-dev-node.sh &
NODE_PID=$!
echo "Devnode started with PID $NODE_PID"
cd ..

# Wait until the devnode RPC accepts requests; a fixed sleep races with node
# startup and the tests then fail with connection resets.
node_ready() {
  curl -s -o /dev/null -X POST -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
    http://127.0.0.1:8547
}
deadline=$(($(date +%s) + 600))
until node_ready; do
  if ! kill -0 "$NODE_PID" 2>/dev/null; then
    echo "ERROR: devnode process died before the RPC came up" >&2
    exit 1
  fi
  if (( $(date +%s) > deadline )); then
    echo "ERROR: timed out waiting for the devnode RPC" >&2
    exit 1
  fi
  sleep 5
done

# Run execution spec tests
git clone https://github.com/OffchainLabs/execution-specs.git
cd execution-specs
curl -LsSf --retry 3 https://astral.sh/uv/install.sh | sh
# The installer puts uv in ~/.local/bin, which is not on PATH on the CI runners.
export PATH="$HOME/.local/bin:$PATH"
uv python install 3.11
uv python pin 3.11
uv sync --all-extras
uv run execute remote --fork=Osaka --rpc-endpoint=http://127.0.0.1:8547 --rpc-seed-key 0xb6b15c8cb491557369f3c7d2c287b053eb229daa9c22138887752191c9520659 --rpc-chain-id 412346 ./tests/ --verbose

# Shut down the dev node
kill $NODE_PID
wait $NODE_PID 2>/dev/null || true

echo "Execution spec tests completed"
