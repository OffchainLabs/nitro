# Q: How do gas fees work on Arbitrum?

Fees on Arbitrum chains are collected on L2 in the chains' native currency (ETH on Arbitrum One and Nova).

A transaction fee is comprised of both an L1 and an L2 component:

The L1 component is meant to compensate the Sequencer for the cost of posting transactions on L1 (but no more). (See [L1 Pricing](arbos/L1_pricing.md).)

The L2 component covers the cost of operating the L2 chain; it uses Geth for gas calculation and thus behaves nearly identically to L1 Ethereum (See [Gas](arbos/Gas)).

L2 Gas price adjusts responsively to chain congestion, ala EIP 1559.

Calling an Arbitrum node's `eth_estimateGas` returns a value sufficient to cover both the L1 and L2 components of the fee for the current gas price (see [here](https://medium.com/offchainlabs/understanding-arbitrum-2-dimensional-fees-fd1d582596c9) for more.)
