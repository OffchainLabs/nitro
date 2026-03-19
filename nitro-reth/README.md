# ArbReth

[![license](https://img.shields.io/badge/license-BUSL--1.1-blue.svg)](LICENSE.md)
[![lint](https://github.com/0xBloctopus/arbreth/actions/workflows/lint.yml/badge.svg)](https://github.com/0xBloctopus/arbreth/actions/workflows/lint.yml)
[![test](https://github.com/0xBloctopus/arbreth/actions/workflows/test.yml/badge.svg)](https://github.com/0xBloctopus/arbreth/actions/workflows/test.yml)
[![build](https://github.com/0xBloctopus/arbreth/actions/workflows/build.yml/badge.svg)](https://github.com/0xBloctopus/arbreth/actions/workflows/build.yml)

A modular, Rust-native execution client for Arbitrum, built on [reth](https://github.com/paradigmxyz/reth).

## What is ArbReth?

ArbReth is a ground-up Rust implementation of [Arbitrum Nitro](https://github.com/OffchainLabs/nitro)'s execution layer. It replaces Nitro's embedded Geth fork with [reth](https://github.com/paradigmxyz/reth), delivering the same state-transition logic through a modular crate architecture designed for extensibility.

Each component (ArbOS state management, L1/L2 pricing, precompiles, Stylus WASM execution) lives in its own crate and builds on reth's trait system (`BlockExecutor`, `StateProvider`, `EvmConfig`). This makes ArbReth usable both as a full node and as an SDK for building Arbitrum-compatible tooling and infrastructure.

**Supported networks:** Arbitrum Sepolia (421614)

## Quick Start

### Docker (recommended)

```bash
cp .env.example .env
# Set PARENT_CHAIN_RPC_URL and PARENT_CHAIN_BEACON_URL
docker compose up -d
```

### Build from Source

**Requirements:** Rust 1.93+, clang, cmake

```bash
cargo build --release -p arb-reth
```

Run the node:

```bash
./target/release/arb-reth node \
  --chain=genesis/arbitrum-sepolia.json \
  --datadir=/path/to/data \
  --http \
  --http.addr=0.0.0.0 \
  --http.api=eth,web3,net,debug \
  --authrpc.addr=0.0.0.0 \
  --authrpc.jwtsecret=/path/to/jwt.hex
```

| Port | Service |
|------|---------|
| 8545 | JSON-RPC (HTTP) |
| 8551 | Engine API (JWT auth) |

See [`.env.example`](.env.example) for all configuration options.

## Architecture

ArbReth is organized as a Cargo workspace of focused, independently consumable crates:

```
crates/
├── arbos/             Core ArbOS state machine, pricing models, retryables
├── arb-evm/           Block executor, custom opcodes, EVM integration
├── arb-precompiles/   Arbitrum precompile contracts (0x64+)
├── arb-stylus/        Stylus WASM runtime and host functions
├── arb-primitives/    Transaction types, receipts, gas types
├── arb-chainspec/     Chain spec and ArbOS version constants
├── arb-storage/       Storage-backed types over reth's StateProvider
├── arb-node/          Node builder plugin for reth
├── arb-rpc/           Custom JSON-RPC methods
├── arb-payload/       Payload building primitives
└── arb-txpool/        Transaction pool validation

bin/
├── arb-reth/          Node binary
└── gen-genesis/       Genesis state generator
```

All crates integrate with the reth ecosystem through its standard traits and can be consumed individually as libraries for custom tooling, indexers, or alternative node configurations.

## Contributing

```bash
git clone https://github.com/0xBloctopus/arbreth.git
cd arbreth
cargo check
cargo test
```

Please open an issue before starting work on larger changes.

## License

Licensed under the [Business Source License 1.1](LICENSE.md), consistent with the [Arbitrum Nitro license](https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md). See [LICENSE.md](LICENSE.md) and [NOTICE](NOTICE) for full terms and third-party attributions.

## Acknowledgements

ArbReth builds on the work of several projects:

- [**reth**](https://github.com/paradigmxyz/reth) by Paradigm, the modular Ethereum execution client that provides the node framework, trait system, and database infrastructure that ArbReth extends.
- [**Arbitrum Nitro**](https://github.com/OffchainLabs/nitro) by Offchain Labs, the Arbitrum node implementation that ArbReth is derived from.
- [**revm**](https://github.com/bluealloy/revm), the Rust EVM that powers transaction execution.
- [**alloy**](https://github.com/alloy-rs/alloy), Rust types and primitives for the Ethereum ecosystem.
