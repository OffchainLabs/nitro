
# Running a Node

Note: If you’re interested in accessing an Arbitrum chain, but you don’t want to set up your own node, see our [Node Providers](./node-providers.md) to get RPC access to fully-managed nodes hosted by a third party provider

### Required Artifacts

- Latest Docker Image: `offchainlabs/nitro-node:v2.0.4-d6a431c`

- Arbitrum One Nitro Genesis Database Snapshot
  - Use the parameter `--init.url="https://snapshot.arbitrum.io/mainnet/nitro.tar"` on first startup to initialize Nitro database
  - If running more than one node, easiest to manually download image from https://snapshot.arbitrum.io/mainnet/nitro.tar and host it locally for your nodes
  - Or use `--init.url="file:///path/to/snapshot/in/container/nitro.tar"` to use a local snapshot archive
  - sha256 checksum: `a609773c6103435b8a04d32c63f42bb5fa0dc8fc38a2acee4d2ab2d05880205c`
  - size: 33.5573504 GB

- Rinkeby Nitro Genesis Database Snapshot
  - Use the parameter `--init.url="https://snapshot.arbitrum.io/rinkeby/nitro.tar"` on first startup to initialize Nitro database
  - If running more than one node, easiest to manually download image from https://snapshot.arbitrum.io/rinkeby/nitro.tar and host it locally for your nodes
  - Or use `--init.url="file:///path/to/snapshot/in/container/nitro.tar"` to use a local snapshot archive

- Other chains do not have classic blocks, and do not require an initial genesis database

### Required parameter

- `--l1.url=<Layer 1 Ethereum RPC URL>`
  - Must provide standard layer 1 node RPC endpoint that you run yourself or from a node provider
- `--l2.chain-id=<L2 Chain ID>`
  - See [public chains](../public-chains.md) for a list of Arbitrum chains and the respective L2 Chain Ids

### Important ports

- RPC: `8547`
- WebSocket: `8548`
- Sequencer Feed: `9642`

### Putting it all together

- When running docker image, an external volume should be mounted to persist the database across restarts. The mount point inside the docker image should be `/home/user/.arbitrum`.
- Here is an example of how to run nitro-node:

  - Note that is important that `/some/local/dir/arbitrum` already exists, otherwise the directory might be created with `root` as owner, and the docker container won't be able to write to it.

  ```shell
  docker run --rm -it  -v /some/local/dir/arbitrum:/home/user/.arbitrum -p 0.0.0.0:8547:8547 -p 0.0.0.0:8548:8548 offchainlabs/nitro-node:v2.0.3-9779dab --l1.url https://l1-node:8545 --l2.chain-id=<L2ChainId> --http.api=net,web3,eth,debug --http.corsdomain=* --http.addr=0.0.0.0 --http.vhosts=*
  ```

  - Note that if you are running L1 node on localhost, you may need to add `--network host` right after `docker run` to use docker host-based networking

  - When shutting down docker image, it is important to allow for a graceful shutdown so that the current state can be saved to disk.  Here is an example of how to do a graceful shutdown of all docker images currently running
  ```shell
  docker stop --time=300 $(docker ps -aq)
  ```

### Note on permissions

- The Docker image is configured to run as non-root UID 1000. This means if you are running in Linux or OSX and you are getting permission errors when trying to run the docker image, run this command to allow all users to update the persistent folders
  ```shell
  mkdir /data/arbitrum
  chmod -fR 777 /data/arbitrum
  ```

### Optional parameters

- `--init.url="https://snapshot.arbitrum.io/mainnet/nitro.tar"`
  - URL to download genesis database from. Only needed when starting Arbitrum One without database
- `--init.url="https://snapshot.arbitrum.io/rinkeby/nitro.tar"`
  - URL to download genesis database from. Only needed when starting Rinkeby Testnet without database
- `--node.rpc.classic-redirect=<classic node RPC>`
  - If set, will redirect archive requests for pre-nitro blocks to the designated RPC, which should be an Arbitrum Classic node with archive database. Only valid for Arbitrum One or Rinkeby Testnet
- `--http.api`
  - APIs offered over the HTTP-RPC interface (default `net,web3,eth`)
  - Add `debug` to enable tracing
- `--http.corsdomain`
  - Comma separated list of domains from which to accept cross origin requests (browser enforced)
- `--http.vhosts`
  - Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts `*` wildcard (default `localhost`)
- `--http.addr`
  - Address to bind RPC to. May need to be set to `0.0.0.0` for docker networking to work properly
- `--node.archive`
  - Retain past block state
- `--node.feed.input.url=<feed address>`
  - Defaults to `wss://<chainName>.arbitrum.io/feed`. If running more than a couple nodes, you will want to provide one feed relay per datacenter, see further instructions below.
- `--node.forwarding-target=<sequencer RPC>`
  - Defaults to appropriate L2 Sequencer RPC depending on L1 and L2 chain IDs provided.
- `--node.rpc.evm-timeout`
  - Defaults to `5s`, timeout used for `eth_call` (0 == no timeout)
- `--node.rpc.gas-cap`
  - Defaults to `50000000`, cap on computation gas that can be used in `eth_call`/`estimateGas` (0 = no cap)
- `--node.rpc.tx-fee-cap`
  - Defaults to `1`, cap on transaction fee (in ether) that can be sent via the RPC APIs (0 = no cap)

### Arb-Relay

- When running more than one node, you want to run a single arb-relay per datacenter, which will reduce ingress fees and improve stability
- The arb-relay is in the same docker image.
- Here is an example of how to run nitro-relay for Arbitrum One:
  ```shell
  docker run --rm -it  -p 0.0.0.0:9642:9642 --entrypoint relay offchainlabs/nitro-node:v2.0.3-9779dab --node.feed.output.addr=0.0.0.0 --node.feed.input.url=wss://arb1.arbitrum.io/feed
  ```
- Here is an example of how to run nitro-node for Arbitrum One with custom relay:
  ```shell
  docker run --rm -it  -v /some/local/dir/arbitrum:/home/user/.arbitrum -p 0.0.0.0:8547:8547 -p 0.0.0.0:8548:8548 offchainlabs/nitro-node:v2.0.3-9779dab --l1.url=https://l1-mainnet-node:8545 --l2.chain-id=42161 --http.api=net,web3,eth,debug --http.corsdomain=* --http.addr=0.0.0.0 --http.vhosts=* --node.feed.input.url=ws://local-relay-address:9642
  ```
