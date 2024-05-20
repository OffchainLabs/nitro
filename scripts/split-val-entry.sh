#!/bin/bash

xxd -l 32 -ps -c 40 /dev/urandom > /tmp/nitro-val.jwt

echo launching validation servers
# To add validation server:
# > launch them here with a different port and --validation.wasm.root-path
# add their port to wait loop
# edit validation-server-configs-list to include the other nodes
/usr/local/bin/nitro-val --file-logging.enable=false --auth.addr 127.0.0.10 --auth.origins 127.0.0.1 --auth.jwtsecret /tmp/nitro-val.jwt --auth.port 52000 &
/home/user/nitro-legacy/bin/nitro-val --file-logging.enable=false --auth.addr 127.0.0.10 --auth.origins 127.0.0.1 --auth.jwtsecret /tmp/nitro-val.jwt --auth.port 52001 --validation.wasm.root-path /home/user/nitro-legacy/machines &
for port in 52000 52001; do
    while ! nc -w1 -z 127.0.0.10 $port; do
        echo waiting for validation port $port
        sleep 1
    done
done
echo launching nitro-node
/usr/local/bin/nitro --validation.wasm.allowed-wasm-module-roots /home/user/nitro-legacy/machines,/workspace/machines --node.block-validator.validation-server-configs-list='[{"jwtsecret":"/tmp/nitro-val.jwt","url":"http://127.0.0.10:52000"}, {"jwtsecret":"/tmp/nitro-val.jwt","url":"http://127.0.0.10:52001"}]' "$@"
