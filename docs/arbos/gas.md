# Gas and Fees
There are two parties a user pays when submitting a tx:

- the poster, if reimbursable, for L1 resources such as the L1 calldata needed to post the tx
- the network fee account for L2 resources, which include the computation, storage, and other burdens L2 nodes must bare to service the tx

The L1 component is the product of the tx's estimated contribution to its batch's size — computed using Brotli on the tx by itself — and the L2's view of the L1 data price, a value which dynamically adjusts over time to ensure the batch-poster is ultimately fairly compensated. For details, see [L1 Pricing](arbos.md#l1pricingstate).

[The L2 component](arbos.md#l2pricingstate) consists of the traditional fees geth would pay to stakers in a vanilla L1 chain, such as the computation and storage charges applying the state transition function entails. ArbOS charges additional fees for executing its L2-specific [precompiles](precompiles.md), whose fees are dynamically priced according to the specific resources used while executing the call.

## Gas Price Floor
The L2 gas price on a given Arbitrum chain has a set floor, which can be queried via [`ArbGasInfo.getMinimumGasPrice`](precompiles.md) (currently 0.1 gwei on both Arbitrum One and Nova). 

## Estimating Gas
Calling an Arbitrum Node's `eth_estimateGas` RPC gives a value sufficient to cover the full transaction fee at the given L2 gas price; i.e., the value returned from `eth_estimateGas` multiplied by the L2 gas price tells you how much total Ether is required for the transaction to succeed. Note that this means that for a given operation, the value returned by `eth_estimateGas` will change over time (as the L1 calldata price fluctuates.) (See [2-D fees](https://medium.com/offchainlabs/understanding-arbitrum-2-dimensional-fees-fd1d582596c9) for more.)


[drop_l1_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/l1pricing/l1pricing.go#L232

## Tips in L2
The sequencer prioritizes transactions on a first-come first-served basis. Because tips do not make sense in this model, they are ignored. Arbitrum users always just pay the basefee regardless of the tip they choose.

## Gas Estimating Retryables
When a transaction schedules another, the subsequent tx's execution [will be included][estimation_inclusion_link] when estimating gas via the node's RPC. A tx's gas estimate, then, can only be found if all the txes succeed at a given gas limit. This is especially important when working with retryables and scheduling redeem attempts.

Because a call to [`redeem`](precompiles.md#ArbRetryableTx) donates all of the call's gas, doing multiple requires limiting the amount of gas provided to each subcall. Otherwise the first will take all of the gas and force the second to necessarily fail irrespective of the estimation's gas limit.

Gas estimation for Retryable submissions is possible via [`NodeInterface.sol`][node_interface_link] and similarly requires the auto-redeem attempt succeed.

[estimation_inclusion_link]: https://github.com/OffchainLabs/go-ethereum/blob/edf6a19157606070b6a6660c8decc513e2408cb7/internal/ethapi/api.go#L955
[node_interface_link]: https://github.com/OffchainLabs/nitro/blob/master/solgen/src/node-interface/NodeInterface.sol

## NodeInterface.sol
To avoid creating new RPC methods for client-side tooling, nitro geth's [`InterceptRPCMessage`][InterceptRPCMessage_link] hook provides an opportunity to swap out the message its handling before deriving a transaction from it. The node [uses this hook][use_hook_link] to detect messages sent to the address `0xc8`, the location of the fictional `NodeInterface` contract specified in [`NodeInterface.sol`][node_interface_link].

`NodeInterface` isn't deployed on L2 and only exists in the RPC, but it contains methods callable via `0xc8`. Doing so requires setting the `To` field to `0xc8` and supplying calldata for the method. Below is the list of methods.

| Method                                                           | Info                                                |
|:-----------------------------------------------------------------|:----------------------------------------------------|
| [`estimateRetryableTicket`][estimateRetryableTicket_link] &nbsp; | Estimates the gas needed for a retryable submission |

[InterceptRPCMessage_link]: https://github.com/OffchainLabs/go-ethereum/blob/f31341b3dfa987719b012bc976a6f4fe3b8a1221/internal/ethapi/api.go#L929
[use_hook_link]: https://github.com/OffchainLabs/nitro/blob/57e03322926f796f75a21f8735cc64ea0a2d11c3/arbstate/node-interface.go#L17
[estimateRetryableTicket_link]: https://github.com/OffchainLabs/nitro/blob/8ab1d6730164e18d0ca1bd5635ca12aadf36a640/solgen/src/node_interface/NodeInterface.sol#L21
