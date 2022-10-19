# ArbOS

ArbOS is the Layer 2 EVM hypervisor that facilitates the execution environment of L2 Arbitrum. ArbOS accounts for and manages network resources, produces blocks from incoming messages, and operates its instrumented instance of geth for smart contract execution.

## Precompiles

ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This section documents the infrastructure that makes this possible. For more details on specific calls, please refer to the [methods documentation](./precompiles.md).

A precompile consists of a of solidity interface in [`contracts/src/precompiles/`][nitro_precompiles_dir] and a corresponding golang implementation in [`precompiles/`][precompiles_dir]. Using geth's abi generator, [`solgen/gen.go`][gen_file] generates [`solgen/go/precompilesgen/precompilesgen.go`][precompilesgen_link], which collects the ABI data of the precompiles. The [runtime installer][installer_link] uses this generated file to check the type safety of each precompile's implementer.

[The installer][installer_link] uses runtime reflection to ensure each implementer has all the right methods and signatures. This includes restricting access to stateful objects like the EVM and statedb based on the declared purity. Additionally, the installer verifies and populates event function pointers to provide each precompile the ability to emit logs and know their gas costs. Additional configuration like restricting a precompile's methods to only be callable by chain owners is possible by adding precompile wrappers like [`ownerOnly`][ownerOnly_link] and [`debugOnly`][debugOnly_link] to their [installation entry][installation_link].

The calling, dispatching, and recording of precompile methods is done via runtime reflection as well. This avoids any human error manually parsing and writing bytes could introduce, and uses geth's stable apis for [packing and unpacking][packing_link] values.

Each time a tx calls a method of an L2-specific precompile, a [`call context`][call_context_link] is created to track and record the gas burnt. For convenience, it also provides access to the public fields of the underlying [`TxProcessor`][TxProcessor_link]. Because sub-transactions could revert without updates to this struct, the [`TxProcessor`][TxProcessor_link] only makes public that which is safe, such as the amount of L1 calldata paid by the top level transaction.

[nitro_precompiles_dir]: https://github.com/OffchainLabs/nitro/tree/master/contracts/src/precompiles
[precompiles_dir]: https://github.com/OffchainLabs/nitro/tree/master/precompiles
[installer_link]: https://github.com/OffchainLabs/nitro/blob/bc6b52daf7232af2ca2fec3f54a5b546f1196c45/precompiles/precompile.go#L379
[installation_link]: https://github.com/OffchainLabs/nitro/blob/bc6b52daf7232af2ca2fec3f54a5b546f1196c45/precompiles/precompile.go#L403
[gen_file]: https://github.com/OffchainLabs/nitro/blob/master/solgen/gen.go
[ownerOnly_link]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/wrapper.go#L58
[debugOnly_link]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/wrapper.go#L23
[precompilesgen_link]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/gen.go#L55
[packing_link]: https://github.com/OffchainLabs/nitro/blob/bc6b52daf7232af2ca2fec3f54a5b546f1196c45/precompiles/precompile.go#L438
[call_context_link]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/context.go#L26

## Messages

An [`L1IncomingMessage`][L1IncomingMessage_link] represents an incoming sequencer message. A message includes one or more user transactions depending on load, and is made into a [unique L2 block][ProduceBlockAdvanced_link]. The L2 block may include additional system transactions added in while processing the message's user txes, but ultimately the relationship is still bijective: for every [`L1IncomingMessage`][L1IncomingMessage_link] there is an L2 block with a unique L2 block hash, and for every L2 block after chain initialization there was an [`L1IncomingMessage`][L1IncomingMessage_link] that made it. A sequencer batch may contain more than one [`L1IncomingMessage`][L1IncomingMessage_link].

[L1IncomingMessage_link]: https://github.com/OffchainLabs/nitro/blob/4ac7e9268e9885a025e0060c9ec30f9612f9e651/arbos/incomingmessage.go#L54
[ProduceBlockAdvanced_link]: https://github.com/OffchainLabs/nitro/blob/4ac7e9268e9885a025e0060c9ec30f9612f9e651/arbos/block_processor.go#L118

## Retryables

A Retryable is a special message type for creating atomic L1 to L2 messages; for details, see [L1 To L2 Messaging](./l1-to-l2-messaging.md).

## ArbOS State

ArbOS's state is viewed and modified via [`ArbosState`][ArbosState_link] objects, which provide convenient abstractions for working with the underlying data of its [`backingStorage`][BackingStorage_link]. The backing storage's [keyed subspace strategy][subspace_link] makes possible [`ArbosState`][ArbosState_link]'s convenient getters and setters, minimizing the need to directly work with the specific keys and values of the underlying storage's [`stateDB`][stateDB_link].

Because two [`ArbosState`][ArbosState_link] objects with the same [`backingStorage`][BackingStorage_link] contain and mutate the same underlying state, different [`ArbosState`][ArbosState_link] objects can provide different views of ArbOS's contents. [`Burner`][Burner_link] objects, which track gas usage while working with the [`ArbosState`][ArbosState_link], provide the internal mechanism for doing so. Some are read-only, causing transactions to revert with `vm.ErrWriteProtection` upon a mutating request. Others demand the caller have elevated privileges. While yet others dynamically charge users when doing stateful work. For safety the kind of view is chosen when [`OpenArbosState()`][OpenArbosState_link] creates the object and may never change. 

Much of ArbOS's state exists to facilitate its [precompiles](precompiles.md). The parts that aren't are detailed below.

[ArbosState_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/arbosState/arbosstate.go#L36
[BackingStorage_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/storage/storage.go#L51
[stateDB_link]: https://github.com/OffchainLabs/go-ethereum/blob/0ba62aab54fd7d6f1570a235f4e3a877db9b2bd0/core/state/statedb.go#L66
[subspace_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/storage/storage.go#L21
[OpenArbosState_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/arbosState/arbosstate.go#L57
[Burner_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/burn/burn.go#L11

### [`arbosVersion`][arbosVersion_link], [`upgradeVersion`][upgradeVersion_link] and [`upgradeTimestamp`][upgradeTimestamp_link]

ArbOS upgrades are scheduled to happen [when finalizing the first block][FinalizeBlock_link] after the [`upgradeTimestamp`][upgradeTimestamp_link].

[arbosVersion_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/arbosState/arbosstate.go#L37
[upgradeVersion_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/arbosState/arbosstate.go#L38
[upgradeTimestamp_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/arbosState/arbosstate.go#L39
[FinalizeBlock_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L350

### [`blockhashes`][blockhashes_link]

This component maintains the last 256 L1 block hashes in a circular buffer. This allows the [`TxProcessor`][TxProcessor_link] to implement the `BLOCKHASH` and `NUMBER` opcodes as well as support precompile methods that involve the outbox. To avoid changing ArbOS state outside of a transaction, blocks made from messages with a new L1 block number update this info during an [`InternalTxUpdateL1BlockNumber`][InternalTxUpdateL1BlockNumber_link] [`ArbitrumInternalTx`][ArbitrumInternalTx_link] that is included as the first tx in the block.

[blockhashes_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/blockhash/blockhash.go#L15
[InternalTxUpdateL1BlockNumber_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/internal_tx.go#L24
[ArbitrumInternalTx_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L116
[TxProcessor_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/tx_processor.go#L33

### [`l1PricingState`][l1PricingState_link]

In addition to supporting the [`ArbAggregator precompile`](precompiles.md#ArbAggregator), the L1 pricing state provides tools for determining the L1 component of a transaction's gas costs. This part of the state tracks both the total amount of funds collected from transactions in L1 gas fees, as well as the funds spent by batch posters to post data batches on L1. 

Based on this information, ArbOS maintains an L1 data fee, also tracked as part of this state, which determines how much transactions will be charged for L1 fees. ArbOS dynamically adjusts this value so that fees collected are approximately equal to batch posting costs, over time.



[l1PricingState_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/l1pricing/l1pricing.go#L16

### [`l2PricingState`][l2PricingState_link]

The L2 pricing state tracks L2 resource usage to determine a reasonable L2 gas price. This process considers a variety of factors, including user demand, the state of geth, and the computational speed limit. The primary mechanism for doing so consists of a pair of pools, one larger than the other, that drain as L2-specific resources are consumed and filled as time passes. L1-specific resources like L1 calldata are not tracked by the pools, as they have little bearing on the actual work done by the network actors that the speed limit is meant to keep stable and synced. 

While much of this state is accessible through the [`ArbGasInfo`](precompiles.md#ArbGasInfo) and [`ArbOwner`](precompiles.md#ArbOwner) precompiles, most changes are automatic and happen during [block production][block_production_link] and [the transaction hooks](geth.md#Hooks). Each of an incoming message's txes removes from the pool the L2 component of the gas it uses, and afterward the message's timestamp [informs the pricing mechanism][notify_pricer_link] of the time that's passed as ArbOS [finalizes the block][finalizeblock_link].

ArbOS's larger gas pool [determines][maintain_limit_link] the per-block gas limit, setting a dynamic [upper limit][per_block_limit_link] on the amount of compute gas an L2 block may have. This limit is always enforced, though for the [first transaction][first_transaction_link] it's done in the [GasChargingHook](geth.md#GasChargingHook) to avoid sharp decreases in the L1 gas price from over-inflating the compute component purchased to above the gas limit. This improves UX by allowing the first tx to succeed rather than requiring a resubmission. Because the first tx lowers the amount of space left in the block, subsequent transactions do not employ this strategy and may fail from such compute-component inflation. This is acceptable because such txes are only present in cases where the system is under heavy load and the result is that the user's tx is dropped without charges since the state transition fails early. Those trusting the sequencer can rely on the tx being automatically resubmitted in such a scenario.

The reason we need a per-block gas limit is that Arbitrator WAVM execution is much slower than native tx execution. This means that there can only be so much gas -- which roughly translates to wall-clock time -- in an L2 block. It also provides an opportunity for ArbOS to limit the size of blocks should demand continue to surge even as the price rises.

ArbOS's per-block gas limit is distinct from geth's block limit, which ArbOS [sets sufficiently high][geth_pool_set_link] so as to never run out. This is safe since geth's block limit exists to constrain the amount of work done per block, which ArbOS already does via its own per-block gas limit. Though it'll never run out, a block's txes use the [same geth gas pool][same_geth_pool_link] to maintain the invariant that the pool decreases monotonically after each tx. Block headers [use the geth block limit][use_geth_pool_link] for internal consistency and to ensure gas estimation works. These are both distinct from the [`gasLeft`][per_block_limit_link] variable, which ephemerally exists outside of global state to both keep L2 blocks from exceeding ArbOS's per-block gas limit and to [deduct space][deduct_space_link] in situations where the state transition failed or [used negligible amounts][negligible_amounts_link] of compute gas. ArbOS does not need to persist [`gasLeft`][per_block_limit_link] because it is its _pool_ that induces a revert and because txes use the geth block limit during EVM execution.

[l2PricingState_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/l2pricing/l2pricing.go#L14
[block_production_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L77
[notify_pricer_link]: https://github.com/OffchainLabs/nitro/blob/fa36a0f138b8a7e684194f9840315d80c390f324/arbos/block_processor.go#L336

[maintain_limit_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/l2pricing/pools.go#L98
[per_block_limit_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/block_processor.go#L146
[first_transaction_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/block_processor.go#L237
[geth_pool_set_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/block_processor.go#L166
[same_geth_pool_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/block_processor.go#L199
[use_geth_pool_link]: https://github.com/OffchainLabs/nitro/blob/2ba6d1aa45abcc46c28f3d4f560691ce5a396af8/arbos/block_processor.go#L67
[deduct_space_link]: https://github.com/OffchainLabs/nitro/blob/faf55a1da8afcabb1f3c406b291e721bfde71a05/arbos/block_processor.go#L272
[negligible_amounts_link]: https://github.com/OffchainLabs/nitro/blob/faf55a1da8afcabb1f3c406b291e721bfde71a05/arbos/block_processor.go#L328
