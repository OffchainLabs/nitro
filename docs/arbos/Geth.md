# Geth

Nitro makes minimal modifications to geth in hopes of not violating its assumptions. This document will explore the relationship between geth and ArbOS, which consists of a series of hooks, interface implementations, and strategic re-appropriations of geth's basic types.

We store ArbOS's state at an address inside a geth `statedb`. In doing so, ArbOS inherits the `statedb`'s statefulness and lifetime properties. For example, a transaction's direct state changes to ArbOS are discarded upon a revert.

<p align=center>0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF<br>
<span style="font-size:smaller;">The fictional account representing ArbOS</span></p>

## Hooks<a name=Hooks></a>

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
                * [`NonrefundableGas`](#NonrefundableGas)
        * [`EndTxHook`](#EndTxHook)
    * added return parameter: `transactionResult`

What follows is an overview of each hook, in chronological order.

### [`ReadyEVMForL2`][ReadyEVMForL2_link]<a name=ReadyEVMForL2></a>
A call to [`ReadyEVMForL2`][ReadyEVMForL2_link] installs the other transaction-specific hooks into each geth [`EVM`][EVM_link] right before it performs a state transition. Without this call, the state transition will instead use the default [`DefaultTxProcessor`][DefaultTxProcessor_link] and get exactly the same results as vanilla geth. A [`TxProcessor`][TxProcessor_link] object is what carries these hooks and the associated arbitrum-specific state during the transaction's lifetime.

### [`StartTxHook`][StartTxHook_link]<a name=StartTxHook></a>
The [`StartTxHook`][StartTxHook_link] is called by geth before a transaction starts executing. This allows ArbOS to handle two arbitrum-specific transaction types. 

If the transaction is `ArbitrumDepositTx`, ArbOS adds balance to the destination account.  This is safe because the L1 bridge submits such a transaction only after collecting the same amount of funds on L1.

If the transaction is an `ArbitrumSubmitRetryableTx`, ArbOS creates a retryable based on the transaction's fields. If the transaction includes sufficient gas, ArbOS schedules a retry of the new retryable.

The hook returns `true` for both of these transaction types, signifying that the state transition is complete. 

### [`GasChargingHook`][GasChargingHook_link]<a name=GasChargingHook></a>

This fallible hook ensures the user has enough funds to pay their poster's L1 calldata costs. If not, the tx is reverted and the [`EVM`][EVM_link] does not start. In the common case that the user can pay, the amount paid for calldata is set aside for later reimbursement of the poster. All other fees go to the network account, as they represent the tx's burden on validators and nodes more generally.

### [`PushCaller`][PushCaller_link]<a name=PushCaller></a> and [`PopCaller`][PopCaller_link]<a name=PopCaller></a>
These hooks track the callers within the EVM callstack, pushing and popping as calls are made and complete. This provides [`ArbSys`](Precompiles.md#ArbSys) with info about the callstack, which it uses to implement the methods [`WasMyCallersAddressAliased`](Precompiles.md#ArbSys) and [`MyCallersAddressWithoutAliasing`](Precompiles.md#ArbSys).

### [`L1BlockHash`][L1BlockHash_link]<a name=L1BlockHash></a> and [`L1BlockNumber`][L1BlockNumber_link]<a name=L1BlockNumber></a>
In arbitrum, the BlockHash and Number operations return data that relies on underlying L1 blocks intead of L2 blocks, to accomendate the normal use-case of these opcodes, which often assume ethereum-like time passes between different blocks. The L1BlockHash and L1BlockNumber hooks have the required data for these operations.

### [`NonrefundableGas`][NonrefundableGas_link]<a name=NonrefundableGas></a>

Because poster costs come at the expense of L1 aggregators and not the network more broadly, the amounts paid for L1 calldata should not be refunded. This hook provides geth access to the equivalent amount of L2 gas the poster's cost equals, ensuring this amount is not reimbursed for network-incentivized behaviors like freeing storage slots.

### [`EndTxHook`][EndTxHook_link]<a name=EndTxHook></a>
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


## Arbitrum Chain Parameters
Nitro's geth may be configured with the following [l2-specific chain parameters][chain_params_link]. These allow the rollup creator to customize their rollup at genesis.

### `EnableArbos`<a name=EnableArbOS></a>
Introduces [ArbOS](#ArbOS.md), converting what would otherwise be a vanilla L1 chain into an L2 Arbitrum rollup.

### `AllowDebugPrecompiles`<a name=AllowDebugPrecompiles></a>
Allows access to debug precompiles. Not enabled for Arbitrum One. When false, calls to debug precompiles will always revert.

### `DataAvailabilityCommittee`<a name=DataAvailabilityCommittee></a>
Currently does nothing besides indicate that the rollup will access a data availability service for preimage resolution in the future. This is not enabled for Arbitrum One, which is a strict state-function of its L1 inbox messages.

[chain_params_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/params/config_arbitrum.go#L25


## Miscellaneous Geth Changes

### ABI Gas Margin
Vanilla Geth's abi library submits txes with the exact estimate the node returns, employing no padding. This means a tx may revert should another arriving just before even slightly change the tx's codepath. To account for this, we've added a `GasMargin` field to `bind.TransactOpts` that [pads estimates][pad_estimates_link] by the number of basis points set.

### Conservation of L2 ETH
The total amount of L2 ether in the system should not change except in controlled cases, such as when bridging. As a safety precaution, ArbOS checks geth's [balance delta][conservation_link] each time a block is created, [alerting or panicking][alert_link] should conservation be violated. 

### MixDigest and ExtraData
To aid with [outbox proof construction][proof_link], the root hash and leaf count of ArbOS's [send merkle accumulator][merkle_link] are stored in the `MixDigest` and `ExtraData` fields of each L2 block. The yellow paper specifies that the `ExtraData` field may be no larger than 32 bytes, so we use the first 8 bytes of the `MixDigest`, which has no meaning in a system without miners, to store the send count.

## Retryable Support
Retryables are mostly implemented in [ArbOS](ArbOS.md#retryables). Some modifications were required in geth to support them.
* Added ScheduledTxes field to ExecutionResult. This lists transactions scheduled during the execution. To enable using this field, we also pass the ExecutionResult to callers of ApplyTransaction.
* Added gasEstimation param to DoCall. When enabled, DoCall will also also executing any retriable activated by the original call. This allows estimating gas to enable retriables.

## Added accessors
Added ['UnderlyingTransaction'][UnderlyingTransaction_link] to Message interface
Added ['GetCurrentTxLogs'](../../go-ethereum/core/state/statedb_arbitrum.go) to StateDB
We created the AdvancedPrecompile interface, which executes and charges gas with the same function call. This is used by Arbitrum's precompiles, and also wraps geth's standard precompiles. For more information on Arbitrum precompiles, see [ArbOS doc](ArbOS.md#precompiles).

### WASM build support
The WASM arbitrum executable does not support file oprations. We created [fileutil.go](../../go-ethereum/core/rawdb/fileutil.go) to wrap fileutil calls, stubbing them out when building WASM. ['fake_leveldb.go'](../../go-ethereum/ethdb/leveldb/fake_leveldb.go) is a similar WASM-mock for leveldb. These are not required for the WASM block-replayer.

## Types
Arbitrum introduces a new ['signer'](../../go-ethereum/core/types/arbitrum_signer.go), and multiple new [`transaction types`](../../go-ethereum/core/types/transaction.go).

## ReorgToOldBlock
Geth natively only allows reorgs to a fork of the currently-known network. In nitro, reorgs can sometimes be detected before computing the forked block. We added the ['ReorgToOldBlock'](../../go-ethereum/core/blockchain_arbitrum.go) function to support reorging to a block that's an ancestor of current head.

## Genesis block creation
Genesis block in nitro is not necessarily block #0. Nitro supports importing blocks that take place before genesis. We split out ['WriteHeadBlock'][WriteHeadBlock_link] from gensis.Commit and use it to commit non-zero genesis blocks.

[pad_estimates_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/accounts/abi/bind/base.go#L352
[conservation_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/core/state/statedb.go#L42
[alert_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L290
[proof_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/system_tests/outbox_test.go#L26
[merkle_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/merkleAccumulator/merkleAccumulator.go#L14
[UnderlyingTransaction_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/core/state_transition.go#L68
[WriteHeadBlock_link]: https://github.com/OffchainLabs/go-ethereum/blob/bf2301d747acb2071fdb64dc82fe7fc122581f0c/core/genesis.go#L332
