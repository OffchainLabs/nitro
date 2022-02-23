# Gas
TODO

## Tips in L2
While tips are not advised for those using the sequencer, which prioritizes transactions on a first-come first-served basis, 3rd-party aggregators may choose to order txes based on tips. A user specifies a tip by setting a gas price in excess of the basefee and will [pay that difference][pay_difference_link] on the amount of gas the tx uses.

A poster receives the tip only when the user has set them as their [preferred aggregator](Precompiles.md#ArbAggregator). Otherwise the tip [goes to the network fee collector][goes_to_network_link]. This disincentives unpreferred aggregators from racing to post txes with large tips.

[pay_difference_link]: https://github.com/OffchainLabs/go-ethereum/blob/edf6a19157606070b6a6660c8decc513e2408cb7/core/state_transition.go#L358
[goes_to_network_link]: https://github.com/OffchainLabs/nitro/blob/c93c806a5cfe99f92a534d3c952a83c3c8b3088c/arbos/tx_processor.go#L262

## Gas Estimating Retryables
Gas estimation via the node's RPC will 

## NodeInterface.sol
To avoid creating new RPC methods for client-side tooling, nitro geth's [`InterceptRPCMessage`][InterceptRPCMessage_link] hook provides an opportunity to swap out the message its handling before deriving a transaction from it. ArbOS uses this hook to detect messages sent to the address `0xc8`, the location of a fictional contract ... TODO

[InterceptRPCMessage_link]: todo
