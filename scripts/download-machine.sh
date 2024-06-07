#!/usr/bin/env bash
set -e

# Create directory for version
mkdir "$2"
cd "$2"

# Create or update the symlink to the latest version directory
ln -sfn "$(pwd)" ../latest

# Store the module root
echo "$2" > module-root.txt

# Define base URL for downloading files
url_base="https://github.com/OffchainLabs/nitro/releases/download/$1"

# Download machine.wavm.br from the specified version
wget "$url_base/machine.wavm.br"

# Check if replay.wasm exists before attempting to download
status_code="$(curl -LI "$url_base/replay.wasm" -so /dev/null -w '%{http_code}')"
if [ "$status_code" -ne 404 ]; then
    wget "$url_base/replay.wasm"
fi