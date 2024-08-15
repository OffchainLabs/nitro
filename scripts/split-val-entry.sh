#!/bin/bash

xxd -l 32 -ps -c 40 /dev/urandom > /tmp/nitro-val.jwt

legacyvalopts=()
latestvalopts=()
while [[ $1 == "--val-options"* ]]; do
    setlegacy=true
    setlatest=true
    if [[ $1 == "--val-options-legacy" ]]; then
        setlatest=false
    fi
    if [[ $1 == "--val-options-latest" ]]; then
        setlegacy=false
    fi
    shift
    while [[ "$1" != "--" ]] && [[ $# -gt 0 ]]; do
        if $setlegacy; then
            legacyvalopts=( "${legacyvalopts[@]}" "$1" )
        fi
        if $setlatest; then
            latestvalopts=( "${latestvalopts[@]}" "$1" )
        fi
        shift
    done
    shift
done
echo launching validation servers
# To add validation server:
# > launch them here with a different port and --validation.wasm.root-path
# add their port to wait loop
# edit validation-server-configs-list to include the other nodes
/usr/local/bin/nitro-val --file-logging.enable=false --auth.addr 127.0.0.10 --auth.origins 127.0.0.1 --auth.jwtsecret /tmp/nitro-val.jwt --auth.port 52000 "${latestvalopts[@]}" &
/home/user/nitro-legacy/bin/nitro-val --file-logging.enable=false --auth.addr 127.0.0.10 --auth.origins 127.0.0.1 --auth.jwtsecret /tmp/nitro-val.jwt --auth.port 52001 --validation.wasm.root-path /home/user/nitro-legacy/machines "${legacyvalopts[@]}" &
for port in 52000 52001; do
    while ! nc -w1 -z 127.0.0.10 $port; do
        echo waiting for validation port $port
        sleep 1
    done
done
echo launching nitro-node
/usr/local/bin/nitro --validation.wasm.allowed-wasm-module-roots /home/user/nitro-legacy/machines,/home/user/target/machines --node.block-validator.validation-server-configs-list='[{"jwtsecret":"/tmp/nitro-val.jwt","url":"ws://127.0.0.10:52000"}, {"jwtsecret":"/tmp/nitro-val.jwt","url":"ws://127.0.0.10:52001"}]' "$@"
