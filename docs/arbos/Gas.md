# Gas and Fees
Fees exist to 

Gas is a concept that serves two purposes:  and buying network resources.



## Tips in L2
While tips are not advised for those using the sequencer, which prioritizes transactions on a first-come first-served basis, 3rd-party aggregators may choose to order txes based on tips. A user specifies a tip by setting a gas price in excess of the basefee and will [pay that difference][pay_difference_link] on the amount of gas the tx uses.

A poster receives the tip only when the user has set them as their [preferred aggregator](Precompiles.md#ArbAggregator). Otherwise the tip [goes to the network fee collector][goes_to_network_link]. This disincentives unpreferred aggregators from racing to post txes with large tips.

[pay_difference_link]: https://github.com/OffchainLabs/go-ethereum/blob/edf6a19157606070b6a6660c8decc513e2408cb7/core/state_transition.go#L358
[goes_to_network_link]: https://github.com/OffchainLabs/nitro/blob/c93c806a5cfe99f92a534d3c952a83c3c8b3088c/arbos/tx_processor.go#L262

## Geth Gas Pool vs ArbOS's


## Gas Estimating Retryables
When a transaction schedules another, the subsequent tx's execution [will be included][estimation_inclusion_link] when estimating gas via the node's RPC. A tx's gas estimate, then, can only be found if all the txes succeed at a given gas limit. This is especially important when working with retryables and scheduling redeem attempts.

Because a call to [`redeem`](#ArbRetryableTx) donates all of the caller's gas, one must use a subcall to limit the amount sent should multiple calls be made. Otherwise the first will take all of the gas and force the second to necessarily fail irrespective of the estimation's gas limit.

Gas estimation for Retryable submissions is possible via `NodeInterface.sol` and similarly requires the auto-redeem attempt succeed.

## NodeInterface.sol<a name=NodeInterface.sol></a>
To avoid creating new RPC methods for client-side tooling, nitro geth's [`InterceptRPCMessage`][InterceptRPCMessage_link] hook provides an opportunity to swap out the message its handling before deriving a transaction from it. The node [uses this hook][use_hook_link] to detect messages sent to the address `0xc8`, the location of the fictional `NodeInterface` contract specified in [`NodeInterface.sol`][node_interface_link].

`NodeInterface` isn't deployed on L2 and only exists in the RPC, but it contains methods callable via `0xc8`. Doing so requires setting the `To` field to `0xc8` and supplying calldata for the method. Below is the list of methods.

| Method                                                           | Info                                                |
|:-----------------------------------------------------------------|:----------------------------------------------------|
| [`estimateRetryableTicket`][estimateRetryableTicket_link] &nbsp; | Estimates the gas needed for a retryable submission |


[estimation_inclusion_link]: https://github.com/OffchainLabs/go-ethereum/blob/edf6a19157606070b6a6660c8decc513e2408cb7/internal/ethapi/api.go#L955
[use_hook_link]: https://github.com/OffchainLabs/nitro/blob/57e03322926f796f75a21f8735cc64ea0a2d11c3/arbstate/node-interface.go#L17
[node_interface_link]: https://github.com/OffchainLabs/nitro/blob/master/solgen/src/node_interface/NodeInterface.sol
[estimateRetryableTicket_link]: https://github.com/OffchainLabs/nitro/blob/8ab1d6730164e18d0ca1bd5635ca12aadf36a640/solgen/src/node_interface/NodeInterface.sol#L21



[InterceptRPCMessage_link]: todo
