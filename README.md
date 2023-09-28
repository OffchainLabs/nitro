<br />
<p align="center">
  <a href="https://arbitrum.io/">
    <img src="https://arbitrum.io/assets/arbitrum/logo_color.png" alt="Logo" width="80" height="80">
  </a>

  <h3 align="center">Arbitrum Nitro</h3>

  <p align="center">
    <a href="https://developer.arbitrum.io/"><strong>Next Generation Ethereum L2 Technology »</strong></a>
    <br />
  </p>
</p>

## About Arbitrum Nitro

<img src="https://arbitrum.io/assets/arbitrum/logo_color.png" alt="Logo" width="80" height="80">

Nitro is the latest iteration of the Arbitrum technology. It is a fully integrated, complete
layer 2 optimistic rollup system, including fraud proofs, the sequencer, the token bridges, 
advanced calldata compression, and more.

See the live docs-site [here](https://developer.arbitrum.io/) (or [here](https://github.com/OffchainLabs/arbitrum-docs) for markdown docs source.)

See [here](./audits) for security audit reports.

The Nitro stack is built on several innovations. At its core is a new prover, which can do Arbitrum’s classic 
interactive fraud proofs over WASM code. That means the L2 Arbitrum engine can be written and compiled using 
standard languages and tools, replacing the custom-designed language and compiler used in previous Arbitrum
versions. In normal execution, 
validators and nodes run the Nitro engine compiled to native code, switching to WASM if a fraud proof is needed. 
We compile the core of Geth, the EVM engine that practically defines the Ethereum standard, right into Arbitrum. 
So the previous custom-built EVM emulator is replaced by Geth, the most popular and well-supported Ethereum client.

The last piece of the stack is a slimmed-down version of our ArbOS component, rewritten in Go, which provides the 
rest of what’s needed to run an L2 chain: things like cross-chain communication, and a new and improved batching 
and compression system to minimize L1 costs.

Essentially, Nitro runs Geth at layer 2 on top of Ethereum, and can prove fraud over the core engine of Geth 
compiled to WASM.

Arbitrum One successfully migrated from the Classic Arbitrum stack onto Nitro on 8/31/22. (See [state migration](https://developer.arbitrum.io/migration/state-migration) and [dapp migration](https://developer.arbitrum.io/migration/dapp_migration) for more info).

## License

We currently have Nitro [licensed](./LICENSE) under a Business Source License, similar to our friends at Uniswap and Aave, with an "Additional Use Grant" to ensure that everyone can have full comfort using and running nodes on all public Arbitrum chains.

## Contact

Discord - [Arbitrum](https://discord.com/invite/5KE54JwyTs)

Twitter: [Arbitrum](https://twitter.com/arbitrum)


