#!/bin/sh

xxd -l 32 -ps -c 40 /dev/urandom > /tmp/nitro-val.jwt
echo launching validation
/usr/local/bin/nitro-val --file-logging.file nitro-val.log --auth.addr 127.0.0.10 --auth.origins 127.0.0.1 --auth.jwtsecret /tmp/nitro-val.jwt --auth.port 2000 &
sleep 2
echo launching nitro-node
/usr/local/bin/nitro --node.block-validator.execution-server.jwtsecret /tmp/nitro-val.jwt --node.block-validator.execution-server.url http://127.0.0.10:2000 "$@"
