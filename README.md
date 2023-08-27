<br />
<p align="center">
  <a href="https://arbitrum.io/">
    <img src="https://thereisatoken.com/media/stylus-logo.svg" alt="Logo" width="100%" height="80">
  </a>

  <p align="center">
    <a href="https://developer.arbitrum.io/"><strong>Next Generation Ethereum L2 Technology »</strong></a>
    <br />
  </p>
</p>

## About Arbitrum Stylus

Stylus is a next-gen programming environment for Arbitrum chains. Through the power of WebAssembly smart contracts, users can deploy programs written in their favorite programming languages, including Rust, C, and C++, to run alongside EVM smart contracts on Arbitrum. It's over an order of magnitude faster, slashes fees, and is fully interoperable with the Ethereum Virtual Machine.

This repo is a fork of [Arbitrum Nitro][Nitro] and is designed as an upgrade for all Arbitrum chains. Included is the Stylus VM and working fraud prover. If you are looking to write and deploy Stylus programs, please see the following SDKs.

| Repo                           | Use cases                   | License           |
|:-------------------------------|:----------------------------|:------------------|
| [Rust SDK][Rust]               | Everything!                 | Apache 2.0 or MIT |
| [C/C++ SDK][C]                 | Cryptography and algorithms | Apache 2.0 or MIT |
| [Bf SDK][Bf]                   | Educational                 | Apache 2.0 or MIT |
| [Cargo Stylus CLI Tool][Cargo] | Program deployment          | Apache 2.0 or MIT |

[Nitro]: https://github.com/OffchainLabs/nitro
[Orbit]: https://docs.arbitrum.io/launch-orbit-chain/orbit-gentle-introduction

Stylus is entirely opt-in. Devs familiar with Solidity can continue to enjoy Arbitrum's EVM-equivalent experience without any changes. This is because Stylus is entirely additive &mdash; a model we call EVM+. Stylus introduces a second, fully composible virtual machine for executing WebAssembly that coordinates with the EVM to produce state transitions. And since the Stylus SDK uses solidity ABIs, a contract written in one language can call out to any other.

For example, existing Solidity DEXs can &mdash; without modifications &mdash; list Rust ERC20 tokens, which might call out to C programs to do cryptography. Everything is fully interoperable, so users never have to care about the specific language or implementation details of the contracts they call.

## Roadmap

Stylus is currently testnet-only and not recommended for production use. This will change as we complete an audit and add additional features.

Arbitrum [Orbit L3s][Orbit] may opt into Stylus at any time. Arbitrum One and Arbitrum Nova will upgrade to Stylus should the DAO vote for it.

If you'd like to be a part of this journey, join us in the `#stylus` channel on [Discord][discord]!

## Gas Pricing

Stylus introduces new pricing models for WASM programs. Intended for high-compute applications, Stylus makes the following more affordable:

- Compute, which is generally **10-100x** cheaper depending on the program. This is primarily due to the efficiency of the WASM runtime relative to the EVM, and the quality of the code produced by Rust, C, and C++ compilers. Another factor that matters is the quality of the code itself. For example, highly optimized and audited C libraries that implement a particular cryptographic operation are usually deployable without modification and perform exceptionally well. The fee reduction may be smaller for highly optimized Solidity that makes heavy use of native precompiles vs an unoptimized Stylus equivalent that doesn't do the same.

- Memory, which is **100-500x** cheaper due to Stylus's novel exponential pricing mechanism intended to address Vitalik's concerns with the EVM's per-call, [quadratic memory][quadratic] pricing policy. For the first time ever, high-memory applications are possible on an EVM-equivalent chain.

- Storage, for which the Rust SDK promotes better access patterns and type choices. Note that while the underlying [`SLOAD`][SLOAD] and [`SSTORE`][SSTORE] operations cost as they do in the EVM, the Rust SDK implements an optimal caching policy that minimizes their use. Exact savings depends on the program.

- VM affordances, including common operations like keccak and reentrancy detection. No longer is it expensive to make safety the default.

There are, however, minor overheads to using Stylus that may matter to your application:

- The first time a WASM is deployed, it must be _activated_. This is generally a few million gas, though to avoid testnet DoS, we've set it to a fixed 14 million. Note that you do not have to activate future copies of the same program. For example, the same NFT template can be deployed many times without paying this cost more than once. We will soon make the fees paid depend on the program, so that the gas used is based on the complexity of the WASM instead of this very conservative, worst-case estimate.

- Calling a Stylus program costs 200-2000 gas. We're working with Wasmer to improve setup costs, but there will likely always be some amount of gas one pays to jump into WASM execution. This means that if a contract does next to nothing, it may be cheaper in Solidity. However if a contract starts doing interesting work, the dynamic fees will quickly make up for this fixed-cost overhead.

Though conservative bounds have been chosen for testnet, all of this is subject to change as pricing models mature and further optimizations are made. Since gas numbers will vary across updates, it may make more sense to clock the time it takes to perform an operation rather than going solely by the numbers reported in receipts.

[quadratic]: https://notes.ethereum.org/@vbuterin/proposals_to_adjust_memory_gas_costs
[SLOAD]: https://www.evm.codes/#54
[SSTORE]: https://www.evm.codes/#55

## License

We currently have the Stylus VM and fraud prover (the contents of this repo) [licensed](./LICENSE) under a Business Source License, similar to our friends at Uniswap and Aave, with an "Additional Use Grant" to ensure that everyone can have full comfort using and running nodes on all public Arbitrum chains.

The Stylus SDK, however, is licensed under different terms. Please see each repo below for more information.

| Repo                           | Use cases                   | License           |
|:-------------------------------|:----------------------------|:------------------|
| [Rust SDK][Rust]               | Everything!                 | Apache 2.0 or MIT |
| [C/C++ SDK][C]                 | Cryptography and algorithms | Apache 2.0 or MIT |
| [Bf SDK][Bf]                   | Educational                 | Apache 2.0 or MIT |
| [Cargo Stylus CLI Tool][Cargo] | Program deployment          | Apache 2.0 or MIT |

[Rust]: https://github.com/OffchainLabs/stylus-sdk-rs
[C]: https://github.com/OffchainLabs/stylus-sdk-c
[Bf]: https://github.com/OffchainLabs/stylus-sdk-bf
[Cargo]: https://github.com/OffchainLabs/cargo-stylus

## Contact

Discord - [Arbitrum][discord]

Twitter - [OffchainLabs](https://twitter.com/OffchainLabs)

[discord]: https://discord.com/invite/5KE54JwyTs
