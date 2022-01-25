# Geth

Nitro makes minimal modifications to geth in hopes of not violating its assumptions. This document will explore the relationship between geth and ArbOS, which consists of a series of hooks, interface implementations, and strategic re-appropriations of geth's basic types.

We store ArbOS's state at an address inside a geth `statedb`. In doing so, ArbOS inherits the `statedb`'s statefulness and lifetime properties. For example, a transaction's direct state changes to ArbOS are discarded upon a revert.

<p align=center>0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF<br>
<span style="font-size:smaller;">The fictional account representing ArbOS</span></p>


## Hooks<a name=Hooks></a>

A call to [`ReadyEVMForL2`][ReadyEVMForL2_link] installs the following transaction-specific hooks into each geth [`EVM`][EVM_link] right before it performs a state transition. Each provides an opportunity for ArbOS to update its state and make decisions about the tx during its lifetime. Without this call, the state transition will instead use the default [`DefaultTxProcessor`][DefaultTxProcessor_link] and get exactly the same results as vanilla geth. A [`TxProcessor`][TxProcessor_link] object is what carries these hooks and the associated arbitrum-specific state during the transaction's lifetime. What follows is an overview of each hook, in chronological order.

### [`StartTxHook`][StartTxHook_link]
The [`StartTxHook`][StartTxHook_link] is called by geth before a transaction starts executing. This allows ArbOS to handle two arbitrum-specific transaction types. 

If the transaction is `ArbitrumDepositTx`, ArbOS adds balance to the destination account.  This is safe because the L1 bridge submits such a transaction only after collecting the same amount of funds on L1.

If the transaction is an `ArbitrumSubmitRetryableTx`, ArbOS creates a retryable based on the transaction's fields. If the transaction includes sufficient gas, ArbOS schedules a retry of the new retryable.

The hook returns `true` for both of these transaction types, signifying that the state transition is complete. 

### [`GasChargingHook`][GasChargingHook_link]

This fallible hook ensures the user has enough funds to pay their poster's L1 calldata costs. If not, the tx is reverted and the [`EVM`][EVM_link] does not start. In the common case that the user can pay, the amount paid for calldata is set aside for later reimbursement of the poster. All other fees go to the network account, as they represent the tx's burden on validators and nodes more generally.

### [`PushCaller`][PushCaller_link] and [`PopCaller`][PopCaller_link]
These hooks track the callers within the EVM callstack, pushing and popping as calls are made and complete. This provides [`ArbSys`](Precompiles.md#ArbSys) with info about the callstack, which it uses to implement the methods [`WasMyCallersAddressAliased`](Precompiles.md#ArbSys) and [`MyCallersAddressWithoutAliasing`](Precompiles.md#ArbSys).

### [`EndTxHook`][EndTxHook_link]
The [`EndTxHook`][EndTxHook_link] is called after the [`EVM`][EVM_link] has returned a transaction's result, allowing one last opportunity for ArbOS to intervene before the state transition is finalized. Final gas amounts are known at this point, enabling ArbOS to credit the network and poster each's share of the user's gas expenditures as well as adjust the pools. The hook returns from the [`TxProcessor`][TxProcessor_link] a final time, in effect discarding its state as the system moves on to the next transaction where the hook's contents will be set afresh.

### [`NonrefundableGas`][NonrefundableGas_link]

Because poster costs come at the expense of L1 aggregators and not the network more broadly, the amounts paid for L1 calldata should not be refunded. This hook provides geth access to the equivalent amount of L2 gas the poster's cost equals, ensuring this amount is not reimbursed for network-incentivized behaviors like freeing storage slots.

[EVM_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/core/vm/evm.go#L101
[DefaultTxProcessor_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/core/vm/arbitrum_evm.go#L26
[TxProcessor_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L26
[StartTxHook_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L63
[ReadyEVMForL2_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbstate/geth-hook.go#L40
[GasChargingHook_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L100
[PushCaller_link]: https://github.com/OffchainLabs/go-ethereum/blob/3fcac93ff22f3761be687f066369ea96bed469e3/core/vm/interpreter.go#L122
[PopCaller_link]: https://github.com/OffchainLabs/go-ethereum/blob/3fcac93ff22f3761be687f066369ea96bed469e3/core/vm/interpreter.go#L124
[NonrefundableGas_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L138
[EndTxHook_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L145


## Interfaces and components

### [`APIBackend`][APIBackend_link]
APIBackend implements the [`ethapi.Bakend`][ethapi.Bakend_link] interface, which allows simple integration of the arbitrum chain to existing geth API. Most calls are answered using the Backend member.

[APIBackend_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/apibackend.go#L27
[ethapi.Bakend_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/internal/ethapi/backend.go#L42

### [`Backend`][Backend_link]
This struct was created as an arbitrum equivalent to the [`Ethereum`][Ethereum_link] struct. It is mostly glue logic, including a pointer to the ArbInterface interface.

[Backend_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/backend.go#L14
[Ethereum_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/eth/backend.go#L65

### [`ArbInterface`][ArbInterface_link]
This interface is the main interaction-point between geth-standard APIs and the arbitrum chain. Geth APIs mostly either check status by working on the Blockchain struct retrieved from the [`Blockchain`][Blockchain_link] call, or send transactions to arbitrum using the [`PublishTransactions`][PublishTransactions_link] call.

[ArbInterface_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/arbos_interface.go#L10
[Blockchain_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/arbos_interface.go#L12
[PublishTransactions_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/arbos_interface.go#L11

### [`RecordingKV`][RecordingKV_link]
RecordingKV is a read-only key-value store, which retrieves values from an internal trie database. All values accessed by a RecordingKV are also recorded internally. This is used to record all preimages accessed during block creation, which will be needed to proove execution of this particular block.
A [`RecordingChainContext`][RecordingChainContext_link] should also be used, to record which block headers the block execution reads (another option would be to always assume the last 256 block headers were accessed).
The process is simplified using two functions: [`PrepareRecording`][PrepareRecording_link] creates a stateDB and chaincontext objects, running block creation process using these objects records the required preimages, and [`PreimagesFromRecording`][PreimagesFromRecording_link] function extracts the preimages recorded.

[RecordingKV_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L21
[RecordingChainContext_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L101
[PrepareRecording_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L133
[PreimagesFromRecording_link]: https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L148


## Arbitrum Chain Parameters
Nitro's geth may be configured with the following [l2-specific chain parameters][chain_params_link]. These allow the rollup creator to customize their rollup at genesis.

### `EnableArbos` 
Introduces [ArbOS](#ArbOS.md), converting what would otherwise be a vanilla L1 chain into an L2 Arbitrum rollup.

### `AllowDebugPrecompiles` 
Allows access to debug precompiles. Not enabled for Arbitrum One. When false, calls to debug precompiles will always revert.

### `DataAvailabilityCommittee`
Currently does nothing besides indicate that the rollup will access a data availability service for preimage resolution in the future. This is not enabled for Arbitrum One, which is a strict state-function of its L1 inbox messages.

[chain_params_link]: todo


## Miscellaneous Geth Changes

### ABI Gas Margin
Vanilla Geth's abi library submits txes with the exact estimate the node returns, employing no padding. This means a tx may revert should another arriving just before even slightly change the tx's codepath. To account for this, we've added a `GasMargin` field to `bind.TransactOpts` that [pads estimates][pad_estimates_link] by the number of basis points set.

### Conservation of L2 ETH
The total amount of L2 ether in the system should not change except in controlled cases, such as when bridging. As a safety precaution, ArbOS checks geth's [balance delta][conservation_link] each time a block is created, [alerting or panicking][alert_link] should conservation be violated. 

### MixDigest and ExtraData
To aid with [outbox proof construction][proof_link], the root hash and leaf count of ArbOS's [send merkle accumulator][merkle_link] are stored in the `MixDigest` and `ExtraData` fields of each L2 block. The yellow paper specifies that the `ExtraData` field may be no larger than 32 bytes, so we use the first 8 bytes of the `MixDigest`, which has no meaning in a system without miners, to store the send count.

[pad_estimates_link]: todo
[conservation_link]: todo
[alert_link]: todo
[proof_link]: todo
[merkle_link]: todo
