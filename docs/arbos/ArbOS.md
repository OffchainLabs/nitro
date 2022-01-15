# ArbOS

ArbOS is the Layer 2 EVM hypervisor that facilitates the execution environment of L2 Arbitrum. ArbOS accounts for and manages network resources, produces blocks from incoming messages, and operates its instrumented instance of geth for smart contract execution.

## Precompiles

ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This section documents the infrastructure that makes this possible. For more details on specific calls, please refer to the [methods documentation](Precompiles.md).

A precompile consists of a of solidity interface in [`solgen/src/precompiles/`](https://github.com/OffchainLabs/nitro/tree/new-retryables/solgen/src/precompiles) and a corresponding golang implementation in [`precompiles/`](https://github.com/OffchainLabs/nitro/tree/new-retryables/precompiles). Using geth's abi generator, [`solgen/gen.go`](https://github.com/OffchainLabs/nitro/blob/new-retryables/solgen/gen.go) generates [`solgen/go/precompilesgen/precompilesgen.go`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/solgen/gen.go#L55), which collects the ABI data of the precompiles. The [runtime installer](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L365) uses this generated file to check the validity of each precompile's implementer.

[The installer](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L365) uses runtime reflection to ensure each implementer has all the right methods and signatures. This includes restricting access to stateful objects like the EVM and statedb based on the declared purity. Additionally, the installer verifies and populates event function pointers to provide each precompile the ability to emit logs and know their gas costs. Additional configuration like restricting a precompile's methods to only be callable by chain owners is possible by adding precompile wrappers like [`ownerOnly`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/wrapper.go#L59) and [`debugOnly`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/wrapper.go#L26) to their [installation entry](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L390).

The calling, dispatching, and recording of precompile methods is done via runtime reflection as well. This avoids any human error manually parsing and writing bytes could introduce, and uses geth's stable apis for [packing and unpacking](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/precompile.go#L401) values.

Each time a tx calls a method, a [`call context`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/precompiles/context.go#L21) is created to track and record the gas burnt. For convenience, it also provides access to the public fields of the underlying [`TxProcessor`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L26). Because sub-transactions could revert without updates to this struct, the [`TxProcessor`](https://github.com/OffchainLabs/nitro/blob/ac5994e4ecf8c33a54d41c8a288494fbbdd207eb/arbos/tx_processor.go#L26) only makes public that which is safe, such as the amount of L1 calldata paid by the top level transaction.

## Retryables

TODO: fill in their lifetime, usage, etc

TODO: decide on scheme for txids and detail it here
