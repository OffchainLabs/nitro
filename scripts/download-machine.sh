#!/usr/bin/env bash
set -e

mkdir "$2"
ln -sf "$2" latest
cd "$2"
echo "$2" > module-root.txt
url_base="https://github.com/OffchainLabs/nitro/releases/download/$1"
wget "$url_base/machine.wavm.br"

status_code="$(curl -LI "$url_base/replay.wasm" -so /dev/null -w '%{http_code}')"
if [ "$status_code" -ne 404 ]; then
    wget "$url_base/replay.wasm"
fi