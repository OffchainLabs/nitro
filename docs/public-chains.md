# Public Chains

The following is a comprehensive list of all of the currently live Arbitrum chains:

| Name                         | RPC Url(s)                                                                                                                         | ID     | Native Currency | Explorer(s)                                                          | Underlying L1 | Current Tech Stack  | Sequencer Feed                         | Nitro Seed Database URLs                 |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ------ | --------------- | -------------------------------------------------------------------- | ------------- | ------------------- | -------------------------------------- | ---------------------------------------- |
| Arbitrum One                 | `https://arb1.arbitrum.io/rpc` `https://arbitrum-mainnet.infura.io/v3/YOUR-PROJECT-ID` `https://arb-mainnet.g.alchemy.com/v2/-KEY` | 42161  | ETH             | `https://arbiscan.io/` `https://explorer.arbitrum.io/`               | Ethereum      | Nitro Rollup (8/31) | `wss://arb1.arbitrum.io/feed`          |  `snapshot.arbitrum.io/mainnet/nitro.tar`                        |
| Arbitrum Nova                | `https://nova.arbitrum.io/rpc`                                                                                                     | 42170  | ETH             |  `https://nova.arbiscan.io/` `https://nova-explorer.arbitrum.io/`                                 | Ethereum      | Nitro AnyTrust      | `wss://nova.arbitrum.io/feed`          | N/A                                      |
| RinkArby^                    | `https://rinkeby.arbitrum.io/rpc`                                                                                                  | 421611 | RinkebyETH      | `https://testnet.arbiscan.io` `https://rinkeby-explorer.arbitrum.io` | Rinkeby       | Nitro Rollup        | `wss://rinkeby.arbitrum.io/feed`       | `snapshot.arbitrum.io/rinkeby/nitro.tar` |
| Nitro Goerli Rollup Testnet^ | `https://goerli-rollup.arbitrum.io/rpc`                                                                                            | 421613 | GoerliETH       | `https://goerli.arbiscan.io` `https://goerli-rollup-explorer.arbitrum.io`                         | Goerli        | Nitro Rollup        | `wss://goerli-rollup.arbitrum.io/feed` | N/A                                      |

^ Testnet

All chains use [bridge.arbitrum.io/](https://bridge.arbitrum.io/) for bridging assets and [retryable-dashboard.arbitrum.io](https://retryable-dashboard.arbitrum.io/) for executing [retryable tickets](l1-to-l2-messagaing) if needed.

For a list of useful contract addresses, see [here](useful-addresses).

### Arbitrum Chains Summary

**Arbitrum One**: Arbitrum One is the flagship Arbitrum mainnet chain; it is an Optimistic Rollup chain running on top of Ethereum Mainnet, and is open to all users. In an upgrade on 8/31, the Arbitrum One chain is/was upgraded to use the [Nitro](https://medium.com/offchainlabs/its-nitro-time-86944693bf29) tech stack, maintaining the same state.
Users can now use [Alchemy](https://alchemy.com/?a=arbitrum-docs), [Infura](https://infura.io/), [QuickNode](https://www.quicknode.com), [Moralis](https://moralis.io/), [Ankr](https://www.ankr.com/), [BlockVision](https://blockvision.org/), and [GetBlock](https://getblock.io/) to interact with the Arbitrum One. See [node providers](node-providers) for more.

**Arbitrum Nova**: Arbitrum Nova is the first mainnet [AnyTrust](inside-anytrust) chain. The following are the members of the initial data availability committee (DAC):
- Consensys
- Google Cloud
- Offchain Labs
- P2P
- Quicknode
- Reddit

Users can now use [QuickNode](https://www.quicknode.com) to interact with the Arbitrum Nova chain. For a full guide of how to set up an Arbitrum node on QuickNode, see the QuickNode's Arbitrum RPC documentation.

**RinkArby**: RinkArby is the longest running Arbitrum testnet. It previously ran on the classic stack, but at block 7/28/2022 it was migrated use the Nitro stack! Rinkarby will be deprecated [when Rinkeby itself gets deprecated](https://blog.ethereum.org/2022/06/21/testnet-deprecation/); plan accordingly!
Users can now use [Alchemy](https://alchemy.com/?a=arbitrum-docs), [Infura](https://infura.io/), [QuickNode](https://www.quicknode.com), [Moralis](https://moralis.io/), [Ankr](https://www.ankr.com/), [BlockVision](https://blockvision.org/), and [GetBlock](https://getblock.io/) to interact with the Arbitrum One. See [node providers](node-providers) for the full guide.

**Nitro Goerli Rollup Testnet**: This testnet (421613) uses the Nitro rollup tech stack; it is expected to be the primary, stable Arbitrum testnet moving forward.
Users can now use [Alchemy](https://alchemy.com/?a=arbitrum-docs), [Infura](https://infura.io/), and [QuickNode](https://www.quicknode.com) to interact with the Arbitrum One. See [node providers](./node-running/node-providers.md) for more.
