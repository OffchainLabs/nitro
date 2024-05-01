#!/bin/bash

xxd -l 32 -ps -c 40 /dev/urandom > /tmp/nitro-val.jwt

echo launching validation servers
# To add validation server:
# > launch them here with a different port and --validation.wasm.root-path
# add their port to wait loop
# edit validation-server-configs-list to include the other nodes
/usr/local/bin/nitro-val --file-logging.enable=false --auth.addr 127.0.0.10 --auth.origins 127.0.0.1 --auth.jwtsecret /tmp/nitro-val.jwt --auth.port 52000 &
for port in 52000; do
    while ! nc -w1 -z 127.0.0.10 $port; do
        echo waiting for validation port $port
        sleep 1
    done
done
echo launching nitro-node
/usr/local/bin/nitro --node.block-validator.pending-upgrade-module-root="0x8b104a2e80ac6165dc58b9048de12f301d70b02a0ab51396c22b4b4b802a16a4" --node.block-validator.validation-server-configs-list='[{"jwtsecret":"/tmp/nitro-val.jwt","url":"http://127.0.0.10:52000"}]' "$@"
