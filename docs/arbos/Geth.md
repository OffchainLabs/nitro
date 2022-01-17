# Geth

Nitro makes minimal modifications to geth in hopes of not violating its assumptions. This document will explore the relationship between geth and ArbOS, which consists of a series of hooks, interface implementations, and strategic re-appropriations of geth's basic types.

We store ArbOS's state at an address inside a geth `statedb`. In doing so, ArbOS inherits the `statedb`'s statefulness and lifetime properties. For example, a transaction's direct state changes to ArbOS are discarded upon a revert.

<p align=center>0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF<br>
<span style="font-size:smaller;">The fictional account representing ArbOS</span></p>

## Hooks

A call to [`ReadyEVMForL2`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbstate/geth-hook.go#L40) installs the following transaction-specific hooks into each geth [`EVM`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/core/vm/evm.go#L101) right before it performs a state transition. Each provides an opportunity for ArbOS to update its state and make decisions about the tx during its lifetime. Without this call, the state transition will instead use the default [`DefaultTxProcessor`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/core/vm/arbitrum_evm.go#L26) and get exactly the same results as vanilla geth. A [`TxProcessor`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L26) object is what carries these hooks and the associated arbitrum-specific state during the transaction's lifetime. What follows is an overview of each hook, in chronological order.

### [`StartTxHook`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L63)
The [`StartTxHook`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L63) prepares ArbOS for arbitrum-specific transaction types: retryables are accounted for, and deposits are made.

Because a revert will discard changes made here, modifications to ArbOS's state are done as if the tx succeeded:

* `ArbitrumRetryTx` immediately deletes its underlying retryable and adds its gas back to the pools
* `ArbitrumSubmitRetryableTx` creates its retryable
* `ArbitrumDepositTx` adds balance to a user's account (the bridge submits these only after collecting funds on L1)

The hook returns `true` in the case of an `ArbitrumDepositTx`, signifying that the state transition is complete. This is the simplest kind of arbitrum transaction and requires no additional work after the balance update.

TODO: fix the above once we've settled on how retryables work

### [`GasChargingHook`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L100)

This fallible hook ensures the user has enough funds to pay their poster's L1 calldata costs. If not, the tx is reverted and the [`EVM`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/core/vm/evm.go#L101) does not start. In the common case that the user can pay, the amount paid for calldata is set aside for later reimbursement of the poster. All other fees go to the network account, as they represent the tx's burden on validators and nodes more generally.

### [`EndTxHook`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L145)
The [`EndTxHook`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L145) is called after the [`EVM`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/core/vm/evm.go#L101) has returned a transaction's result, allowing one last opportunity for ArbOS to intervene before the state transition is finalized. Final gas amounts are known at this point, enabling ArbOS to credit the network and poster each's share of the user's gas expenditures as well as adjust the pools. The hook returns from the [`TxProcessor`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L26) a final time, in effect discarding its state as the system moves on to the next transaction where the hook's contents will be set afresh.

### [`NonrefundableGas`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L138)

Because poster costs come at the expense of L1 aggregators and not the network more broadly, the amounts paid for L1 calldata should not be refunded. This hook provides geth access to the equivalent amount of L2 gas the poster's cost equals, ensuring this amount is not reimbursed for network-incentivized behaviors like freeing storage slots.

## Interfaces and components

### [`APIBackend`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/apibackend.go#L27)
APIBackend implements the [`ethapi.Bakend`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/internal/ethapi/backend.go#L42) interface, which allows simple integration of the arbitrum chain to existing geth API. Most calls are answered using the Backend member.

### [`Backend`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/backend.go#L14)
This struct was created as an arbitrum equivalent to the [`Ethereum`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/eth/backend.go#L65) struct. It is mostly glue logic, including a pointer to the ArbInterface interface.

### [`ArbInterface`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/arbos_interface.go#L10)
This interface is the main interaction-point between geth-standard APIs and the arbitrum chain. Geth APIs mostly either check status by working on the Blockchain struct retrieved from the [`Blockchain`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/arbos_interface.go#L12) call, or send transactions to arbitrum using the [`PublishTransactions`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/arbos_interface.go#L11) call.

### [`RecordingKV`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L21)
RecordingKV is a read-only key-value store, which retrieves values from an internal trie database. All values accessed by a RecordingKV are also recorded internally. This is used to record all preimages accessed during block creation, which will be needed to proove execution of this particular block.
A [`RecordingChainContext`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L101) should also be used, to record which block headers the block execution reads (another option would be to always assume the last 256 block headers were accessed).
The process is simplified using two functions: [`PrepareRecording`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L133) creates a stateDB and chaincontext objects, running block creation process using these objects records the required preimages, and [`PreimagesFromRecording`](https://github.com/OffchainLabs/go-ethereum/blob/f796d1a6abc99ff0d4ff668e1213a7dfe2d27a0d/arbitrum/recordingdb.go#L148) function extracts the preimages recorded.

## Arbitrum Chain Parameters
Arbitrum One is not 
A Nitro rollup can be configured at genesis with 

### `EnableArbos` 
Introduces ArbOS, creating an

### `AllowDebugPrecompiles` 
Allows access to debug precompiles. Not enabled for Arbitrum One. When false, calls to debug precompiles will always revert.

### `DataAvailabilityCommittee`
Currently does nothing besides indicate that the rollup will access a data availability service for preimage resolution. This is not enabled for Arbitrum One, which is a strict state-function of its L1 inbox messages.
