#!/usr/bin/env bash

xxd -l 32 -ps -c 40 /dev/urandom > /tmp/nitro-val.jwt

valopts=()
while [[ $1 == "--val-options"* ]]; do
    shift
    while [[ "$1" != "--" ]] && [[ $# -gt 0 ]]; do
        valopts=( "${valopts[@]}" "$1" )
        shift
    done
    shift
done
echo launching rust validation server
/usr/local/bin/validator --address 0.0.0.0:4141 --jwt-secret /tmp/nitro-val.jwt --root-path /home/user/target/machines "${valopts[@]}" &
echo waiting for rust validation server to start
if ! timeout 30s bash -c 'until curl -s localhost:4141 > /dev/null 2>&1; do sleep 1; done'; then
    echo rust validation server failed to start within timeout
    exit 1
fi
echo launching nitro-node
exec /usr/local/bin/nitro --validation.wasm.allowed-wasm-module-roots /home/user/target/machines --node.block-validator.validation-server-configs-list='[{"jwtsecret":"/tmp/nitro-val.jwt","url":"http://127.0.0.1:4141"}]' "$@"
