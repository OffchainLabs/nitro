# ArbOS

ArbOS is the Layer 2 EVM hypervisor that facilitates the execution environment of L2 Arbitrum. ArbOS accounts for and manages network resources, produces blocks from incoming messages, and operates its instrumented instance of geth for smart contract execution.

## Precompiles

ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This section documents the infrastructure that makes this possible. For more details on specific calls, please refer to the [methods documentation](Precompiles.md).

A precompile consists of a of solidity interface in [`solgen/src/precompiles/`](https://github.com/OffchainLabs/nitro/tree/new-retryables/solgen/src/precompiles) and a corresponding golang implementation in [`precompiles/`](https://github.com/OffchainLabs/nitro/tree/new-retryables/precompiles). Using geth's abi generator, [`solgen/gen.go`](https://github.com/OffchainLabs/nitro/blob/new-retryables/solgen/gen.go) generates [`solgen/go/precompilesgen/precompilesgen.go`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/solgen/gen.go#L55), which collects the ABI data of the precompiles. The [runtime installer](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L365) uses this generated file to check the type safety of each precompile's implementer.

[The installer](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L365) uses runtime reflection to ensure each implementer has all the right methods and signatures. This includes restricting access to stateful objects like the EVM and statedb based on the declared purity. Additionally, the installer verifies and populates event function pointers to provide each precompile the ability to emit logs and know their gas costs. Additional configuration like restricting a precompile's methods to only be callable by chain owners is possible by adding precompile wrappers like [`ownerOnly`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/wrapper.go#L59) and [`debugOnly`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/wrapper.go#L26) to their [installation entry](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L390).

The calling, dispatching, and recording of precompile methods is done via runtime reflection as well. This avoids any human error manually parsing and writing bytes could introduce, and uses geth's stable apis for [packing and unpacking](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L401) values.

Each time a tx calls a method of an L2-specific precompile, a [`call context`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/context.go#L21) is created to track and record the gas burnt. For convenience, it also provides access to the public fields of the underlying [`TxProcessor`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L26). Because sub-transactions could revert without updates to this struct, the [`TxProcessor`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L26) only makes public that which is safe, such as the amount of L1 calldata paid by the top level transaction.

## Retryables

A Retryable is a transaction whose *submission* is separate from its *execution*.   A retryable can be submitted for a fixed cost (dependent only on its calldata size) paid at L1.  If the L1 transition to request submission succeeds (i.e. does not revert) then the submission of the Retryable to the L2 state is guaranteed to succeed.

After a Retryable is submitted, anyone can try to *redeem* it, by calling the `redeem` method of the `ArbRetryableTx` precompile.  The party requesting the redeem provides the gas that will be used to execute the Retryable.  If execution of the Retryable succeeds, the Retryable is deleted.  If execution fails, the Retryable continues to exist and further attempts can be made to redeem it.  If a fixed period (currently one week) elapses without a successful redeem, the Retryable expires and will automatically be discarded, unless some party has paid a fee to *renew* the Retryable for another full period.  A Retryable can live indefinitely as long as it is renewed each time before it expires.

### Submitting a Retryable

A transaction to submit a Retryable does the following:

* create a new Retryable with the caller, destination, callvalue, and calldata of the submit transaction
* deduct funds to cover the callvalue from the caller (as usual) and place them into escrow for later use in redeeming the Retryable
* assign a unique TicketID to the Retryable
* cause the ArbRetryableTx precompiled contract to emit a TicketCreated event containing the TicketID
* if the submit transaction contains gas, schedule a redeem of the new Retryable, using the supplied gas, as if the `redeem` method of the `ArbRetryableTx` precompile had been called.

In many use cases, the submitter will provide gas and will intend for the immediate redeem to succeed, with later retries available only as a backup mechanism should the immediate redeem fail. (It might fail, for example, because the L2 gas price has increased unexpectedly.) In this way, an L1 contract can submit a transaction to L2 in such a way that the transaction will normally run immediately at L2 but allowing any party to retry the transaction should it fail.

When a Retryable is redeemed, it will execute with the sender, destination, callvalue, and calldata of the original submission. The callvalue will have been escrowed during the initial submission of the Retryable, for this purpose.  If a Retryable with callvalue is eventually discarded, having never successfully run, the escrowed callvalue will be paid out to a "beneficiary" account that is specified in the initial submission.

### Redeeming a Retryable

## ArbOS State

ArbOS's state is viewed and modified via [`ArbosState`][ArbosState_link] objects, which provide convenient abstractions for working with the underlying data of its [`backingStorage`][BackingStorage_link]. The backing storage's [keyed subspace strategy][subspace_link] makes possible [`ArbosState`][ArbosState_link]'s convenient getters and setters, minimizing the need to directly work with the specific keys and values of the underlying storage's [`stateDB`][stateDB_link].

Because two [`ArbosState`][ArbosState_link] objects with the same [`backingStorage`][BackingStorage_link] contain and mutate the same underlying state, different [`ArbosState`][ArbosState_link] objects can provide different views of ArbOS's contents. [`Burner`][Burner_link] objects, which track gas usage while working with the [`ArbosState`][ArbosState_link], provide the internal mechanism for doing so. Some are read-only, causing transactions to revert with `vm.ErrWriteProtection` upon a mutating request. Others demand the caller have elevated privileges. While yet others dynamically charge users when doing stateful work. For safety the kind of view is chosen when [`OpenArbosState()`][OpenArbosState_link] creates the object and may never change. 

Much of ArbOS's state exists to facilitate its [precompiles](Precompiles.md). The parts that aren't are detailed below.

### `l1PricingState`

### `l2PricingState`

### `retryableState`

[BackingStorage_link]: todo
[ArbosState_link]: todo
[stateDB_link]: todo
[subspace_link]: todo
[OpenArbosState_link]: todo
[Burner_link]: todo


