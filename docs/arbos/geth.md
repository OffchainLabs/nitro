# Geth

Nitro makes minimal modifications to geth in hopes of not violating its assumptions. This document will explore the relationship between geth and ArbOS, which consists of a series of hooks, interface implementations, and strategic re-appropriations of geth's basic types.

We store ArbOS's state at an address inside a geth `statedb`. In doing so, ArbOS inherits the `statedb`'s statefulness and lifetime properties. For example, a transaction's direct state changes to ArbOS are discarded upon a revert.

**0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF**<br/>
The fictional account representing ArbOS

## Hooks

Arbitrum uses various hooks to modify geth's behavior when processing transactions. Each provides an opportunity for ArbOS to update its state and make decisions about the tx during its lifetime. Transactions are applied using geth's [`ApplyTransaction`][ApplyTransaction_link] function.

Below is [`ApplyTransaction`][ApplyTransaction_link]'s callgraph, with additional info on where the various Arbitrum-specific hooks are inserted. Click on any to go to their section. By default, these hooks do nothing so as to leave geth's default behavior unchanged, but for chains configured with [`EnableArbOS`](#EnableArbOS) set to true, [`ReadyEVMForL2`](#ReadyEVMForL2) installs the alternative L2 hooks.

* `core.ApplyTransaction` ⮕ `core.applyTransaction` ⮕ `core.ApplyMessage`
    * `core.NewStateTransition`
        * [`ReadyEVMForL2`](#ReadyEVMForL2)
    * `core.TransitionDb`
        * [`StartTxHook`](#StartTxHook)
        * `core.transitionDbImpl`
            * if `IsArbitrum()` remove tip
            * [`GasChargingHook`](#GasChargingHook)
            * `evm.Call`
                * `core.vm.EVMInterpreter.Run`
                    * [`PushCaller`](#PushCaller)
                    * [`PopCaller`](#PopCaller)
            * `core.StateTransition.refundGas`
                * [`ForceRefundGas`](#ForceRefundGas)
                * [`NonrefundableGas`](#NonrefundableGas)
        * [`EndTxHook`](#EndTxHook)
    * added return parameter: `transactionResult`

What follows is an overview of each hook, in chronological order.

### [`ReadyEVMForL2`][ReadyEVMForL2_link]
A call to [`ReadyEVMForL2`][ReadyEVMForL2_link] installs the other transaction-specific hooks into each geth [`EVM`][EVM_link] right before it performs a state transition. Without this call, the state transition will instead use the default [`DefaultTxProcessor`][DefaultTxProcessor_link] and get exactly the same results as vanilla geth. A [`TxProcessor`][TxProcessor_link] object is what carries these hooks and the associated arbitrum-specific state during the transaction's lifetime.

### [`StartTxHook`][StartTxHook_link]
The [`StartTxHook`][StartTxHook_link] is called by geth before a transaction starts executing. This allows ArbOS to handle two arbitrum-specific transaction types. 

If the transaction is `ArbitrumDepositTx`, ArbOS adds balance to the destination account.  This is safe because the L1 bridge submits such a transaction only after collecting the same amount of funds on L1.

If the transaction is an `ArbitrumSubmitRetryableTx`, ArbOS creates a retryable based on the transaction's fields. If the transaction includes sufficient gas, ArbOS schedules a retry of the new retryable.

The hook returns `true` for both of these transaction types, signifying that the state transition is complete. 

### [`GasChargingHook`][GasChargingHook_link]

This fallible hook ensures the user has enough funds to pay their poster's L1 calldata costs. If not, the tx is reverted and the [`EVM`][EVM_link] does not start. In the common case that the user can pay, the amount paid for calldata is set aside for later reimbursement of the poster. All other fees go to the network account, as they represent the tx's burden on validators and nodes more generally.

If the user attempts to purchase compute gas in excess of ArbOS's per-block gas limit, the difference is [set aside][difference_set_aside_link] and [refunded later][refunded_later_link] via [`ForceRefundGas`](#ForceRefundGas) so that only the gas limit is used. Note that the limit observed may not be the same as that seen [at the start of the block][that_seen_link] if ArbOS's larger gas pool falls below the [`MaxPerBlockGasLimit`][max_perblock_limit_link] while processing the block's previous txes.

[difference_set_aside_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/tx_processor.go#L272
[refunded_later_link]: https://github.com/OffchainLabs/go-ethereum/blob/f31341b3dfa987719b012bc976a6f4fe3b8a1221/core/state_transition.go#L389
[that_seen_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/block_processor.go#L146
[max_perblock_limit_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/l2pricing/pools.go#L100

### [`PushCaller`][PushCaller_link]
These hooks track the callers within the EVM callstack, pushing and popping as calls are made and complete. This provides [`ArbSys`](precompiles.md#ArbSys) with info about the callstack, which it uses to implement the methods [`WasMyCallersAddressAliased`](precompiles.md#ArbSys) and [`MyCallersAddressWithoutAliasing`](precompiles.md#ArbSys).

### [`L1BlockHash`][L1BlockHash_link]
In arbitrum, the BlockHash and Number operations return data that relies on underlying L1 blocks intead of L2 blocks, to accomendate the normal use-case of these opcodes, which often assume ethereum-like time passes between different blocks. The L1BlockHash and L1BlockNumber hooks have the required data for these operations.

### [`ForceRefundGas`][ForceRefundGas_link]

This hook allows ArbOS to add additional refunds to the user's tx. This is currently only used to refund any compute gas purchased in excess of ArbOS's per-block gas limit during the [`GasChargingHook`](#GasChargingHook).

### [`NonrefundableGas`][NonrefundableGas_link]

Because poster costs come at the expense of L1 aggregators and not the network more broadly, the amounts paid for L1 calldata should not be refunded. This hook provides geth access to the equivalent amount of L2 gas the poster's cost equals, ensuring this amount is not reimbursed for network-incentivized behaviors like freeing storage slots.

### [`EndTxHook`][EndTxHook_link]
The [`EndTxHook`][EndTxHook_link] is called after the [`EVM`][EVM_link] has returned a transaction's result, allowing one last opportunity for ArbOS to intervene before the state transition is finalized. Final gas amounts are known at this point, enabling ArbOS to credit the network and poster each's share of the user's gas expenditures as well as adjust the pools. The hook returns from the [`TxProcessor`][TxProcessor_link] a final time, in effect discarding its state as the system moves on to the next transaction where the hook's contents will be set afresh.

[ApplyTransaction_link]: https://github.com/OffchainLabs/go-ethereum/blob/8eac46ef5e0298e6cc171f5a46b5c1fe4923bf48/core/state_processor.go#L144
[EVM_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/core/vm/evm.go#L101
[DefaultTxProcessor_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/core/vm/evm_arbitrum.go#L39
[TxProcessor_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L33
[StartTxHook_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L77
[ReadyEVMForL2_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbstate/geth-hook.go#L38
[GasChargingHook_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L205
[PushCaller_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L60
[PopCaller_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L64
[ForceRefundGas_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/tx_processor.go#L291
[NonrefundableGas_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L248
[EndTxHook_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L255
[L1BlockHash_link]: https://github.com/OffchainLabs/nitro/blob/df5344a48f4a24173b9a3794318079a869aae58b/arbos/tx_processor.go#L407
[L1BlockNumber_link]: https://github.com/OffchainLabs/nitro/blob/df5344a48f4a24173b9a3794318079a869aae58b/arbos/tx_processor.go#L399

## Interfaces and components

### [`APIBackend`][APIBackend_link]
APIBackend implements the [`ethapi.Bakend`][ethapi.Bakend_link] interface, which allows simple integration of the arbitrum chain to existing geth API. Most calls are answered using the Backend member.

[APIBackend_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/apibackend.go#L27
[ethapi.Bakend_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/internal/ethapi/backend.go#L42

### [`Backend`][Backend_link]
This struct was created as an arbitrum equivalent to the [`Ethereum`][Ethereum_link] struct. It is mostly glue logic, including a pointer to the ArbInterface interface.

[Backend_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/backend.go#L15
[Ethereum_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/eth/backend.go#L65

### [`ArbInterface`][ArbInterface_link]
This interface is the main interaction-point between geth-standard APIs and the arbitrum chain. Geth APIs mostly either check status by working on the Blockchain struct retrieved from the [`Blockchain`][Blockchain_link] call, or send transactions to arbitrum using the [`PublishTransactions`][PublishTransactions_link] call.

[ArbInterface_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/arbos_interface.go#L10
[Blockchain_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/arbos_interface.go#L12
[PublishTransactions_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/arbos_interface.go#L11

### [`RecordingKV`][RecordingKV_link]
RecordingKV is a read-only key-value store, which retrieves values from an internal trie database. All values accessed by a RecordingKV are also recorded internally. This is used to record all preimages accessed during block creation, which will be needed to proove execution of this particular block.
A [`RecordingChainContext`][RecordingChainContext_link] should also be used, to record which block headers the block execution reads (another option would be to always assume the last 256 block headers were accessed).
The process is simplified using two functions: [`PrepareRecording`][PrepareRecording_link] creates a stateDB and chaincontext objects, running block creation process using these objects records the required preimages, and [`PreimagesFromRecording`][PreimagesFromRecording_link] function extracts the preimages recorded.

[RecordingKV_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/recordingdb.go#L21
[RecordingChainContext_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/recordingdb.go#L101
[PrepareRecording_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/recordingdb.go#L133
[PreimagesFromRecording_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/arbitrum/recordingdb.go#L148

## Transaction Types

Nitro geth includes a few L2-specific transaction types. Click on any to jump to their section.

| Tx Type                                           | Represents                           | Last Hook Reached &nbsp;   | Source |
|:--------------------------------------------------|:-------------------------------------|:---------------------------|--------|
| [`ArbitrumUnsignedTx`][ArbTxUnsigned]             | An L1 to L2 message                  | [`EndTxHook`][HE]          | Bridge |
| [`ArbitrumContractTx`][ArbTxContract]             | A nonce-less L1 to L2 message &nbsp; | [`EndTxHook`][HE]          | Bridge |
| [`ArbitrumDepositTx`][ArbTxDeposit]               | A user deposit                       | [`StartTxHook`][HS]        | Bridge |
| [`ArbitrumSubmitRetryableTx`][ArbTxSubmit] &nbsp; | Creating a retryable                 | [`StartTxHook`][HS] &nbsp; | Bridge |
| [`ArbitrumRetryTx`][ArbTxRetry]                   | A retryable redeem attempt           | [`EndTxHook`][HE]          | L2     |
| [`ArbitrumInternalTx`][ArbTxInternal]             | ArbOS state update                   | [`StartTxHook`][HS]        | ArbOS  |

[ArbTxUnsigned]: #ArbitrumUnsignedTx
[ArbTxContract]: #ArbitrumContractTx
[ArbTxSubmit]: #ArbitrumSubmitRetryableTx
[ArbTxRetry]: #ArbitrumRetryTx
[ArbTxDeposit]: #ArbitrumDepositTx
[ArbTxInternal]: #ArbitrumInternalTx
[HS]: #StartTxHook
[HE]: #EndTxHook

The following reference documents each type.

### [`ArbitrumUnsignedTx`][ArbitrumUnsignedTx_link]
Provides a mechanism for a user on L1 to message a contract on L2. This uses the bridge for authentication rather than requiring the user's signature. Note, the user's acting address will be remapped on L2 to distinguish them from a normal L2 caller.

### [`ArbitrumContractTx`][ArbitrumContractTx_link]
These are like an [`ArbitrumUnsignedTx`][ArbitrumUnsignedTx_link] but are intended for smart contracts. These use the bridge's unique, sequential nonce rather than requiring the caller specify their own. An L1 contract may still use an [`ArbitrumUnsignedTx`][ArbitrumUnsignedTx_link], but doing so may necessitate tracking the nonce in L1 state.

### [`ArbitrumDepositTx`][ArbitrumDepositTx_link]
Represents a user deposit from L1 to L2. This increases the user's balance by the amount deposited on L1.

### [`ArbitrumSubmitRetryableTx`][ArbitrumSubmitRetryableTx_link]
Represents a retryable submission and may schedule an [`ArbitrumRetryTx`](#ArbitrumRetryTx) if provided enough gas. Please see the [retryables documentation](arbos.md#Retryables) for more info.

### [`ArbitrumRetryTx`][ArbitrumRetryTx_link]
These are scheduled by calls to the [`redeem`](precompiles.md#ArbRetryableTx) precompile method and via retryable auto-redemption. Please see the [retryables documentation](arbos.md#Retryables) for more info.

### [`ArbitrumInternalTx`][ArbitrumInternalTx_link]
Because tracing support requires ArbOS's state-changes happen inside a transaction, ArbOS may create a tx of this type to update its state in-between user-generated transactions. Such a tx has a [`Type`][InternalType_link] field signifying the state it will update, though currently this is just future-proofing as there's only one value it may have. Below are the internal tx types.

#### [`ArbInternalTxUpdateL1BlockNumber`][ArbInternalTxUpdateL1BlockNumber_link]
Updates the L1 block number. This tx [is generated][block_generated_link] whenever a message originates from an L1 block newer than any ArbOS has seen thus far. They are [guaranteed to be the first][block_first_link] in their L2 block.

[ArbitrumUnsignedTx_link]: https://github.com/OffchainLabs/go-ethereum/blob/e7e8104942fd2ba676d4b8616c9fa83d88b61c9c/core/types/arb_types.go#L15
[ArbitrumContractTx_link]: https://github.com/OffchainLabs/go-ethereum/blob/e7e8104942fd2ba676d4b8616c9fa83d88b61c9c/core/types/arb_types.go#L76
[ArbitrumSubmitRetryableTx_link]: https://github.com/OffchainLabs/go-ethereum/blob/e7e8104942fd2ba676d4b8616c9fa83d88b61c9c/core/types/arb_types.go#L194
[ArbitrumRetryTx_link]: https://github.com/OffchainLabs/go-ethereum/blob/e7e8104942fd2ba676d4b8616c9fa83d88b61c9c/core/types/arb_types.go#L133
[ArbitrumDepositTx_link]: https://github.com/OffchainLabs/go-ethereum/blob/e7e8104942fd2ba676d4b8616c9fa83d88b61c9c/core/types/arb_types.go#L265
[ArbitrumInternalTx_link]: https://github.com/OffchainLabs/nitro/blob/master/arbos/internal_tx.go

[InternalType_link]: https://github.com/OffchainLabs/go-ethereum/blob/e7e8104942fd2ba676d4b8616c9fa83d88b61c9c/core/types/arb_types.go#L313
[ArbInternalTxUpdateL1BlockNumber_link]: https://github.com/OffchainLabs/nitro/blob/aa55a504d32f71f4ce3a6552822c0791711f8299/arbos/internal_tx.go#L24
[block_generated_link]: https://github.com/OffchainLabs/nitro/blob/aa55a504d32f71f4ce3a6552822c0791711f8299/arbos/block_processor.go#L150
[block_first_link]: https://github.com/OffchainLabs/nitro/blob/aa55a504d32f71f4ce3a6552822c0791711f8299/arbos/block_processor.go#L154

## Transaction Run Modes and Underlying Transactions
A [geth message][geth_message_link] may be processed for various purposes. For example, a message may be used to estimate the gas of a contract call, whereas another may perform the corresponding state transition. Nitro geth denotes the intent behind a message by means of its [`TxRunMode`][TxRunMode_link], [which it sets][set_run_mode_link] before processing it. ArbOS uses this info to make decisions about the tx the message ultimately constructs.

A message [derived from a transaction][AsMessage_link] will carry that transaction in a field accessible via its [`UnderlyingTransaction`][underlying_link] method. While this is related to the way a given message is used, they are not one-to-one. The table below shows the various run modes and whether each could have an underlying transaction.

| Run Mode                                 | Scope                   | Carries an Underlying Tx?                                                          |
|:-----------------------------------------|:------------------------|:-----------------------------------------------------------------------------------|
| [`MessageCommitMode`][MC0]               | state transition &nbsp; | always                                                                             |
| [`MessageGasEstimationMode`][MC1] &nbsp; | gas estimation          | when created via [`NodeInterface.sol`](gas.md#NodeInterface.sol) or when scheduled |
| [`MessageEthcallMode`][MC2]              | eth_calls               | never                                                                              |

[MC0]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/core/types/transaction.go#L648
[MC1]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/core/types/transaction.go#L649
[MC2]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/core/types/transaction.go#L650

[geth_message_link]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/core/types/transaction.go#L628
[TxRunMode_link]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/core/types/transaction.go#L695
[set_run_mode_link]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/internal/ethapi/api.go#L911
[AsMessage_link]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/core/types/transaction.go#L670
[underlying_link]: https://github.com/OffchainLabs/go-ethereum/blob/1e9c9b86135dafebf7ab84641a5674e4249ee849/core/types/transaction.go#L694

## Arbitrum Chain Parameters
Nitro's geth may be configured with the following [l2-specific chain parameters][chain_params_link]. These allow the rollup creator to customize their rollup at genesis.

### `EnableArbos`
Introduces [ArbOS](arbos.md), converting what would otherwise be a vanilla L1 chain into an L2 Arbitrum rollup.

### `AllowDebugPrecompiles`
Allows access to debug precompiles. Not enabled for Arbitrum One. When false, calls to debug precompiles will always revert.

### `DataAvailabilityCommittee`
Currently does nothing besides indicate that the rollup will access a data availability service for preimage resolution in the future. This is not enabled for Arbitrum One, which is a strict state-function of its L1 inbox messages.

[chain_params_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/params/config_arbitrum.go#L25


## Miscellaneous Geth Changes

### ABI Gas Margin
Vanilla Geth's abi library submits txes with the exact estimate the node returns, employing no padding. This means a tx may revert should another arriving just before even slightly change the tx's codepath. To account for this, we've added a `GasMargin` field to `bind.TransactOpts` that [pads estimates][pad_estimates_link] by the number of basis points set.

### Conservation of L2 ETH
The total amount of L2 ether in the system should not change except in controlled cases, such as when bridging. As a safety precaution, ArbOS checks geth's [balance delta][conservation_link] each time a block is created, [alerting or panicking][alert_link] should conservation be violated. 

### MixDigest and ExtraData
To aid with [outbox proof construction][proof_link], the root hash and leaf count of ArbOS's [send merkle accumulator][merkle_link] are stored in the `MixDigest` and `ExtraData` fields of each L2 block. The yellow paper specifies that the `ExtraData` field may be no larger than 32 bytes, so we use the first 8 bytes of the `MixDigest`, which has no meaning in a system without miners/stakers, to store the send count.

### Retryable Support
Retryables are mostly implemented in [ArbOS](arbos.md#retryables). Some modifications were required in geth to support them.
* Added ScheduledTxes field to ExecutionResult. This lists transactions scheduled during the execution. To enable using this field, we also pass the ExecutionResult to callers of ApplyTransaction.
* Added gasEstimation param to DoCall. When enabled, DoCall will also also executing any retryable activated by the original call. This allows estimating gas to enable retryables.

### Added accessors
Added [`UnderlyingTransaction`][UnderlyingTransaction_link] to Message interface
Added [`GetCurrentTxLogs`](../../go-ethereum/core/state/statedb_arbitrum.go) to StateDB
We created the AdvancedPrecompile interface, which executes and charges gas with the same function call. This is used by Arbitrum's precompiles, and also wraps geth's standard precompiles. For more information on Arbitrum precompiles, see [ArbOS doc](arbos.md#precompiles).

### WASM build support
The WASM arbitrum executable does not support file oprations. We created [`fileutil.go`](../../go-ethereum/core/rawdb/fileutil.go) to wrap fileutil calls, stubbing them out when building WASM. [`fake_leveldb.go`](../../go-ethereum/ethdb/leveldb/fake_leveldb.go) is a similar WASM-mock for leveldb. These are not required for the WASM block-replayer.

### Types
Arbitrum introduces a new [`signer`](../../go-ethereum/core/types/arbitrum_signer.go), and multiple new [`transaction types`](../../go-ethereum/core/types/transaction.go).

### ReorgToOldBlock
Geth natively only allows reorgs to a fork of the currently-known network. In nitro, reorgs can sometimes be detected before computing the forked block. We added the [`ReorgToOldBlock`](../../go-ethereum/core/blockchain_arbitrum.go) function to support reorging to a block that's an ancestor of current head.

### Genesis block creation
Genesis block in nitro is not necessarily block #0. Nitro supports importing blocks that take place before genesis. We split out [`WriteHeadBlock`][WriteHeadBlock_link] from gensis.Commit and use it to commit non-zero genesis blocks.

[pad_estimates_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/accounts/abi/bind/base.go#L352
[conservation_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/core/state/statedb.go#L42
[alert_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L290
[proof_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/system_tests/outbox_test.go#L26
[merkle_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/merkleAccumulator/merkleAccumulator.go#L14
[UnderlyingTransaction_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/core/state_transition.go#L68
[WriteHeadBlock_link]: https://github.com/OffchainLabs/go-ethereum/blob/bf2301d747acb2071fdb64dc82fe7fc122581f0c/core/genesis.go#L332
