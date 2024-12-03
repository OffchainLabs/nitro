#!/usr/bin/env bash

echo "==== Pull eigenda-proxy container ===="
docker pull  ghcr.io/layr-labs/eigenda-proxy:v1.6.0

echo "==== Starting eigenda-proxy container ===="

# proxy has a bug currently which forces the use of the service manager address 
# & eth rpc despite cert verification being disabled.

docker run -d --name eigenda-proxy-nitro-test-instance \
  -p 4242:6666 \
  -e EIGENDA_PROXY_ADDR=0.0.0.0 \
  -e EIGENDA_PROXY_PORT=6666 \
  -e EIGENDA_PROXY_MEMSTORE_ENABLED=true \
  -e EIGENDA_PROXY_MEMSTORE_EXPIRATION=1m \
  -e EIGENDA_PROXY_EIGENDA_ETH_RPC=http://localhost:6969 \
  -e EIGENDA_PROXY_EIGENDA_SERVICE_MANAGER_ADDR="0x0000000000000000000000000000000000000000" \
  -e EIGENDA_PROXY_EIGENDA_CERT_VERIFICATION_DISABLED=true \
  ghcr.io/layr-labs/eigenda-proxy:v1.6.0

# shellcheck disable=SC2181
if [ $? -ne 0 ]; then
  echo "==== Failed to start eigenda-proxy container ===="
  exit 1
fi

echo "==== eigenda-proxy container started ===="

## TODO - support teardown or embed a docker client wrapper that spins up and tears down resource 
# within system tests. Since this is only used by one system test, it's not a large priority atm.