# ArbOS

ArbOS is the Layer 2 EVM hypervisor that facilitates the execution environment of L2 Arbitrum. ArbOS accounts for and manages network resources, produces blocks from incoming messages, and operates its instrumented instance of geth for smart contract execution.

## Precompiles

ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This section documents the infrastructure that makes this possible. For more details on specific calls, please refer to the [methods documentation](Precompiles.md).

A precompile consists of a of solidity interface in [`solgen/src/precompiles/`][solgen_precompiles_dir] and a corresponding golang implementation in [`precompiles/`][precompiles_dir]. Using geth's abi generator, [`solgen/gen.go`][gen_file] generates [`solgen/go/precompilesgen/precompilesgen.go`][precompilesgen_link], which collects the ABI data of the precompiles. The [runtime installer][installer_link] uses this generated file to check the type safety of each precompile's implementer.

[The installer][installer_link] uses runtime reflection to ensure each implementer has all the right methods and signatures. This includes restricting access to stateful objects like the EVM and statedb based on the declared purity. Additionally, the installer verifies and populates event function pointers to provide each precompile the ability to emit logs and know their gas costs. Additional configuration like restricting a precompile's methods to only be callable by chain owners is possible by adding precompile wrappers like [`ownerOnly`][ownerOnly_link] and [`debugOnly`][debugOnly_link] to their [installation entry][installation_link].

The calling, dispatching, and recording of precompile methods is done via runtime reflection as well. This avoids any human error manually parsing and writing bytes could introduce, and uses geth's stable apis for [packing and unpacking][packing_link] values.

Each time a tx calls a method of an L2-specific precompile, a [`call context`][call_context_link] is created to track and record the gas burnt. For convenience, it also provides access to the public fields of the underlying [`TxProcessor`][TxProcessor_link]. Because sub-transactions could revert without updates to this struct, the [`TxProcessor`][TxProcessor_link] only makes public that which is safe, such as the amount of L1 calldata paid by the top level transaction.

[solgen_precompiles_dir]: https://github.com/OffchainLabs/nitro/tree/master/solgen/src/precompiles
[precompiles_dir]: https://github.com/OffchainLabs/nitro/tree/master/precompiles
[installer_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L365
[installation_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L390
[gen_file]: https://github.com/OffchainLabs/nitro/blob/master/solgen/gen.go
[ownerOnly_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/wrapper.go#L59
[debugOnly_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/wrapper.go#L26
[precompilesgen_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/solgen/gen.go#L55
[packing_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L401
[call_context_link]: https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/context.go#L21

## Retryables

A Retryable is a transaction whose *submission* is separate from its *execution*.  A retryable can be submitted for a fixed cost (dependent only on its calldata size) paid at L1.  If the L1 transition to request submission succeeds (i.e. does not revert) then the submission of the Retryable to the L2 state is guaranteed to succeed.

After a Retryable is submitted, anyone can try to *redeem* it, by calling the [`redeem`](Precompiles.md#ArbRetryableTx) method of the [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx) precompile.  The party requesting the redeem provides the gas that will be used to execute the Retryable.  If execution of the Retryable succeeds, the Retryable is deleted.  If execution fails, the Retryable continues to exist and further attempts can be made to redeem it.  If a fixed period (currently one week) elapses without a successful redeem, the Retryable expires and will be [automatically discarded][discard_link], unless some party has paid a fee to [*renew*][renew_link] the Retryable for another full period.  A Retryable can live indefinitely as long as it is renewed each time before it expires.

[discard_link]: todo
[renew_link]: todo


### Submitting a Retryable

A transaction to submit a Retryable does the following:

* create a new Retryable with the caller, destination, callvalue, and calldata of the submit transaction
* deduct funds to cover the callvalue from the caller (as usual) and place them into escrow for later use in redeeming the Retryable
* assign a unique TicketID to the Retryable
* cause the ArbRetryableTx precompiled contract to emit a TicketCreated event containing the TicketID
* if the submit transaction contains gas, schedule a redeem of the new Retryable, using the supplied gas, as if the [`redeem`](Precompiles.md#ArbRetryableTx) method of the [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx) precompile had been called.

In many use cases, the submitter will provide gas and will intend for the immediate redeem to succeed, with later retries available only as a backup mechanism should the immediate redeem fail. (It might fail, for example, because the L2 gas price has increased unexpectedly.) In this way, an L1 contract can submit a transaction to L2 in such a way that the transaction will normally run immediately at L2 but allowing any party to retry the transaction should it fail.

When a Retryable is redeemed, it will execute with the sender, destination, callvalue, and calldata of the original submission. The callvalue will have been escrowed during the initial submission of the Retryable, for this purpose.  If a Retryable with callvalue is eventually discarded, having never successfully run, the escrowed callvalue will be paid out to a "beneficiary" account that is specified in the initial submission.

A Retryable's beneficiary has the unique power to [`cancel`](Precompiles.md#ArbRetryableTx) the Retryable. This has the same effect as the Retryable timing out, except when done during a [`redeem`](Precompiles.md#ArbRetryableTx) in which case the escrowed funds  [will have already been moved][moved_link] to the Retryable's `From` address (which Geth then moves to the `To` address or the deployed contract if `To` is not specified). This ensures no additional funds are minted when a retry transaction cancels its own Retryable.

[moved_link]: todo

### Redeeming a Retryable

If a redeem is not done at submission or the submission's initial redeem fails, anyone can attempt to redeem the retryable again by calling [`ArbRetryableTx`](Precompiles.md#ArbRetryableTx)'s [`redeem`](Precompiles.md#ArbRetryableTx) precompile method, which donates the call's gas to the next attempt. ArbOS will [enqueue the redeem][enqueue_link], which is its own special `ArbitrumRetryTx` type, to its list of redeems that ArbOS [guarantees to exhaust][exhaust_link] before moving on to the next non-redeem transaction in the block its forming. In this manner redeems are scheduled to happen as soon as possible, and will always be in the same block since the gas donated came from the pool.

On success, the `To` address keeps the escrowed callvalue, and any unused gas is returned to the pools. On failure, the callvalue is returned to the escrow for the next redeemer. In either case, the network fee was paid during the scheduling tx, so no fees are charged and no refunds are made. 

[enqueue_link]: todo
[exhaust_link]: todo

## ArbOS State

ArbOS's state is viewed and modified via [`ArbosState`][ArbosState_link] objects, which provide convenient abstractions for working with the underlying data of its [`backingStorage`][BackingStorage_link]. The backing storage's [keyed subspace strategy][subspace_link] makes possible [`ArbosState`][ArbosState_link]'s convenient getters and setters, minimizing the need to directly work with the specific keys and values of the underlying storage's [`stateDB`][stateDB_link].

Because two [`ArbosState`][ArbosState_link] objects with the same [`backingStorage`][BackingStorage_link] contain and mutate the same underlying state, different [`ArbosState`][ArbosState_link] objects can provide different views of ArbOS's contents. [`Burner`][Burner_link] objects, which track gas usage while working with the [`ArbosState`][ArbosState_link], provide the internal mechanism for doing so. Some are read-only, causing transactions to revert with `vm.ErrWriteProtection` upon a mutating request. Others demand the caller have elevated privileges. While yet others dynamically charge users when doing stateful work. For safety the kind of view is chosen when [`OpenArbosState()`][OpenArbosState_link] creates the object and may never change. 

Much of ArbOS's state exists to facilitate its [precompiles](Precompiles.md). The parts that aren't are detailed below.

[ArbosState_link]: todo
[BackingStorage_link]: todo
[stateDB_link]: todo
[subspace_link]: todo
[OpenArbosState_link]: todo
[Burner_link]: todo

### [`arbosVersion`][arbosVersion_link], [`upgradeVersion`][upgradeVersion_link] and [`upgradeTimestamp`][upgradeTimestamp_link]

ArbOS upgrades are scheduled to happen [when finalizing the first block][FinalizeBlock_link] after the [`upgradeTimestamp`][upgradeTimestamp_link].

[arbosVersion_link]: todo
[upgradeVersion_link]: todo
[upgradeTimestamp_link]: todo
[FinalizeBlock_link]: todo

### [`blockhashes`][blockhashes_link]

This component maintains the last 256 L1 block hashes in a circular buffer. This allows the [`TxProcessor`][TxProcessor_link] to implement the `BLOCKHASH` and `NUMBER` opcodes as well as support precompile methods that involve the outbox. To avoid changing ArbOS state outside of a transaction, blocks made from messages with a new L1 block number update this info during an [`InternalTxUpdateL1BlockNumber`][InternalTxUpdateL1BlockNumber_link] [`ArbitrumInternalTx`][ArbitrumInternalTx_link] that is included as the first tx in the block.

[blockhashes_link]: todo
[InternalTxUpdateL1BlockNumber_link]: todo
[ArbitrumInternalTx_link]: todo
[TxProcessor_link]: todo

### [`l1PricingState`][l1PricingState_link]

In addition to supporting the [`ArbAggregator precompile`](Precompiles.md#ArbAggregator), the L1 pricing state provides tools for determining the L1 component of a transaction's gas costs. Aggregators, whose compressed batches are the messages ArbOS uses to build L2 blocks, inform ArbOS of their compression ratios so that L2 fees can be fairly allocated between the network fee account and the aggregator posting a given transaction.

Theoretically an aggregator can lie about its compression ratio to slightly inflate the fees their users (and only their users) pay, but a malicious aggregator already has the ability to extract MEV from them so no trust assumptions change. Lying about the ratio being higher than it is is self defeating since it burns money, as is choosing to not compress their users' transactions.

The L1 pricing state also keeps a running estimate of the L1 gas price, which updates as ArbOS processes delayed messages.

[l1PricingState_link]: todo

### [`l2PricingState`][l2PricingState_link]

The L2 pricing state tracks L2 resource usage to determine a reasonable L2 gas price. This process considers a variety of factors, including user demand, the state of geth, and the computational speed limit. The primary mechanism for doing so consists of a pair of pools, one larger than the other, that drain as L2-specific resources are consumed and filled as time passes. L1-specific resources like L1 calldata are not tracked by the pools, as they have little bearing on the actual work done by the network actors that the speed limit is meant to keep stable and synced. 

While much of this state is accessible through the [`ArbGasInfo`](Precompiles.md#ArbGasInfo) and [`ArbOwner`](Precompiles.md#ArbOwner) precompiles, most changes are automatic and happen during [block production][block_production_link] and [the transaction hooks](Geth.md#Hooks). Each of an incoming message's txes removes from the pool the L2 component of the gas it uses, and afterward the message's timestamp [informs the pricing mechanism][notify_pricer_link] of the time that's passed as ArbOS [finalizes the block][finalizeblock_link].

[l2PricingState_link]: todo
[block_production_link]: todo
[notify_pricer_link]: todo
