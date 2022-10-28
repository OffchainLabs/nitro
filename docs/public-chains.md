# Public Chains

The following is a comprehensive list of all of the currently live Arbitrum chains:

| Name                         | RPC Url(s)                                                                                                                         | ID     | Native Currency | Explorer(s)                                                          | Underlying L1 | Current Tech Stack  | Sequencer Feed                         | Nitro Seed Database URLs                 |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ------ | --------------- | -------------------------------------------------------------------- | ------------- | ------------------- | -------------------------------------- | ---------------------------------------- |
| Arbitrum One                 | `(1)https://arb1.arbitrum.io/rpc`  `(2)https://arbitrum-mainnet.infura.io/v3/YOUR-PROJECT-ID`  ` (3)https://arb-mainnet.g.alchemy.com/v2/-KEY` | 42161  | ETH             | `https://arbiscan.io/` `https://explorer.arbitrum.io/`               | Ethereum      | Nitro Rollup (8/31) | `wss://arb1.arbitrum.io/feed`          |  `snapshot.arbitrum.io/mainnet/nitro.tar`                        |
| Arbitrum Nova                | `(1)https://nova.arbitrum.io/rpc`                                                                                                     | 42170  | ETH             |  `https://nova.arbiscan.io/` `https://nova-explorer.arbitrum.io/`                                 | Ethereum      | Nitro AnyTrust      | `wss://nova.arbitrum.io/feed`          | N/A                                      |
| Nitro Goerli Rollup Testnet^ | `(1)https://goerli-rollup.arbitrum.io/rpc`   `(2)https://arb-goerli.g.alchemy.com/v2/-KEY`                                                                                            | 421613 | GoerliETH       | `https://goerli.arbiscan.io` `https://goerli-rollup-explorer.arbitrum.io`                         | Goerli        | Nitro Rollup        | `wss://goerli-rollup.arbitrum.io/feed` | N/A                                      |

^ _Testnet_

All chains use [bridge.arbitrum.io/](https://bridge.arbitrum.io/) for bridging assets and [retryable-dashboard.arbitrum.io](https://retryable-dashboard.arbitrum.io/) for executing [retryable tickets](l1-to-l2-messagaing) if needed. For a list of useful contract addresses, see [here](useful-addresses).

### Arbitrum Chains | Summary

_Tip – users can use [Alchemy](https://alchemy.com/arbitrum/?a=arbitrum-docs) for Arbitrum One mainnet and Arbitrum's Goerli testnet._

- **Arbitrum One**: Arbitrum One is the flagship Arbitrum mainnet chain. It is an Optimistic Rollup chain running on top of Ethereum Mainnet, and is open to all users. In an upgrade on 8/31, the Arbitrum One chain was upgraded to use the [Nitro](https://medium.com/offchainlabs/its-nitro-time-86944693bf29) tech stack, maintaining the same state.

  Users can use [Alchemy](https://alchemy.com/arbitrum/?a=arbitrum-docs), [Ankr](https://www.ankr.com/), [BlockVision](https://blockvision.org/), [GetBlock](https://getblock.io/), [Infura](https://infura.io/), [Moralis](https://moralis.io/), and [QuickNode](https://www.quicknode.com), to interact with the Arbitrum One chain. See the [node providers page](https://developer.arbitrum.io/node-running/node-providers) for more details.

- **Arbitrum Nova**: Arbitrum Nova is the first mainnet [AnyTrust](inside-anytrust) chain. The following are the members of the initial data availability committee (DAC): Consensys, FTX, Google Cloud, Offchain Labs, P2P, Quicknode, and Reddit.

  Users can use [QuickNode](https://www.quicknode.com) to interact with the Arbitrum Nova chain. Check out QuickNode's developer docs on how to set up these nodes.

- **Nitro Goerli Rollup Testnet**: This Goerli testnet (421613) uses the Nitro rollup tech stack, and is the only supported Arbitrum testnet. All other testnets including Rinkeby have been deprecated due to the Ethereum merge in Oct 2022. 

  Beyond offering a RPC endpoint for the Goerli testnet, Alchemy also powers a reliable Goerli testnet available for anyone to receive free Goerli testETH – [goerlifaucet.com](https://goerlifaucet.com). 

