#!/bin/bash
set -e
cd "$(dirname "$0")"

sudo docker build .. -f Dockerfile -t nitro-reproducible-wasm
container=$(sudo docker create nitro-reproducible-wasm)
sudo rm -rf lib
sudo docker cp "$container:/usr/src/nitro/arbitrator/target/env/lib" lib
sudo chown -R $(whoami) lib
mv lib/*.wasm .
rm -rf lib
sudo docker rm "$container"
cargo build --manifest-path ../arbitrator/Cargo.toml --release
sha256sum *.wasm
printf "WAVM module root: "
../arbitrator/target/release/prover --output-module-root replay.wasm -l wasi_stub.wasm -l host_io.wasm -l soft-float.wasm -l go_stub.wasm
