# Arbitrum Nitro

This is a fork of https://github.com/OffchainLabs/nitro with modifications made to better support testing and development of [The Graph's](https://thegraph.com/) multi-layer protocol. See https://github.com/graphprotocol/contracts for details on the protocol contracts.


## Quick setup

For a quick setup of your local Nitro environment, run:

```bash
  git clone https://github.com/edgeandnode/nitro
  pushd nitro
  git submodule update --init --recursive
  ./test-node.bash --init --no-blockscout --detach
```

This will start a local Arbitrum testnet with a single sequencer node and all Arbitrum contracts deployed and ready to go. The L1 node will be available at http://localhost:8545 while the L2 sequencer at http://localhost:8547. A prefunded account can be accessed using the following private key: `e887f7d17d07cc7b8004053fb8826f6657084e88904bb61590e498ca04704cf2`

__Note__: if you run the test nodes in "attached mode" (by removing the `--detach` flag) you'll need to manually deploy the Arbitrum contracts by running `docker-compose run network-gen` on a separate terminal.