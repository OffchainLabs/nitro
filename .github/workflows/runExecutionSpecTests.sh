#!/bin/bash

set -euo pipefail

# Build local nitro image
docker build --target nitro-node-dev --tag nitro-local-build .

# Clone nitro-devnode repo
git clone https://github.com/OffchainLabs/nitro-devnode.git
cd nitro-devnode

# Start nitro-devnode in background
TARGET_IMAGE=nitro-local-build ./run-dev-node.sh &
NODE_PID=$!
echo "Devnode started with PID $NODE_PID"
cd ..

# Give the devnode time to initialize if needed
sleep 10

# Run execution spec tests
git clone https://github.com/OffchainLabs/execution-specs.git
cd execution-specs
curl -LsSf https://astral.sh/uv/install.sh | sh
uv python install 3.11
uv python pin 3.11
uv sync --all-extras
uv run execute remote --fork=Osaka --rpc-endpoint=http://127.0.0.1:8547 --rpc-seed-key 0xb6b15c8cb491557369f3c7d2c287b053eb229daa9c22138887752191c9520659 --rpc-chain-id 412346 ./tests/ --verbose

# Shut down the dev node
kill $NODE_PID
wait $NODE_PID 2>/dev/null || true

echo "Execution spec tests completed"
