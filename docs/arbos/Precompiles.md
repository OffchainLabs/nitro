# Overview
ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This reference exhaustively documents the specific calls ArbOS makes available. For more details on the infrastructure that makes this possible, please refer to the [ArbOS documentation](ArbOS.md). For an abbreviated reference on the precompiles we expect users to most often use, please see the [common precompiles documentation](CommonPrecompiles.md).

From the perspective of user applications, precompiles live as contracts at the following addresses. Click on any to jump to their section.

| Precompile                                     | Address &nbsp; | Purpose                                            |
| :--------------------------------------------- | :------------- | :------------------------------------------------- |
| [`ArbAddressTable`](#ArbAddressTable)          | `0x66`         | Supporting compression of addresses                |
| [`ArbAggregator`](#ArbAggregator)              | `0x6d`         | Configuring transaction aggregation                |
| [`ArbBLS`](#ArbBLS)                            | `0x67`         | Managing BLS keys                                  |
| [`ArbDebug`](#ArbDebug)                        | `0xff`         | Testing tools                                      |
| [`ArbFunctionTable`](#ArbFunctionTable) &nbsp; | `0x68`         | No longer used                                     |
| [`ArbGasInfo`](#ArbGasInfo)                    | `0x6c`         | Info about gas pricing                             |
| [`ArbInfo`](#ArbInfo)                          | `0x65`         | Info about accounts                                |
| [`ArbOwner`](#ArbOwner)                        | `0x70`         | Chain administration, callable only by chain owner |
| [`ArbOwnerPublic`](#ArbOwnerPublic)            | `0x6b`         | Info about chain owners                            |
| [`ArbosTest`](#ArbosTest)                      | `0x69`         | No longer used                                     |
| [`ArbRetryableTx`](#ArbRetryableTx)            | `0x6e`         | Managing retryables                                |
| [`ArbStatistics`](#ArbStatistics)              | `0x6f`         | Info about the pre-Nitro state                     |
| [`ArbSys`](#ArbSys)                            | `0x64`         | System-level functionality                         |

[ArbAddressTable_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbAddressTable.go
[ArbAggregator_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbAddressTable.go
[ArbBLS_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbBLS.go
[ArbDebug_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbDebug.go
[ArbFunctionTable_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbFunctionTable.go
[ArbInfo_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbInfo.go
[ArbGasInfo_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbGasInfo.go
[ArbosTest_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbosTest.go
[ArbOwner_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbOwner.go
[ArbOwnerPublic_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbOwnerPublic.go
[ArbRetryableTx_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbRetryableTx.go
[ArbStatistics_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbStatistics.go
[ArbSys_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbSys.go

# [ArbAddressTable][ArbAddressTable_link]<a name=ArbAddressTable></a>
Provides the ability to create short-hands for commonly used accounts.

| Methods                                                                        |                                                                                           |
|:-------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------|
| [<img src=e.png height=16>][ATs0] [`AddressExists`][AT0]`(address)`            | Checks if an address exists in the table                                                  |
| [<img src=e.png height=16>][ATs1] [`Compress`][AT1]`(address)`                 | Gets bytes that represent the address                                                     |
| [<img src=e.png height=16>][ATs2] [`Decompress`][AT2]`(buffer, offset)` &nbsp; | Replaces the compressed bytes at the given offset with those of the corresponding account |
| [<img src=e.png height=16>][ATs3] [`Lookup`][AT3]`(address)`                   | Gets the index of an address in the table                                                 |
| [<img src=e.png height=16>][ATs4] [`LookupIndex`][AT4]`(index)`                | Gets the address at an index in the table                                                 |
| [<img src=e.png height=16>][ATs5] [`Register`][AT5]`(address)`                 | Adds an address to the table, shrinking its compressed representation                     |
| [<img src=e.png height=16>][ATs6] [`Size`][AT6]`()`                            | Gets the number of addresses in the table                                                 |

[AT0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAddressTable.go#L18
[AT1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAddressTable.go#L23
[AT2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAddressTable.go#L28
[AT3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAddressTable.go#L41
[AT4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAddressTable.go#L53
[AT5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAddressTable.go#L68
[AT6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAddressTable.go#L74

[ATs0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAddressTable.sol#L31
[ATs1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAddressTable.sol#L38
[ATs2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAddressTable.sol#L46
[ATs3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAddressTable.sol#L55
[ATs4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAddressTable.sol#L61
[ATs5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAddressTable.sol#L68
[ATs6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAddressTable.sol#L73


# [ArbAggregator][ArbAggregator_link]<a name=ArbAggregator></a>
Provides aggregators and their users methods for configuring how they participate in L1 aggregation. Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless `SetPreferredAggregator` is invoked to change it.

Compression ratios are measured in basis points. Methods that are checkmarked are access-controlled and will revert if not called by the aggregator, its fee collector, or a chain owner.

| Methods                                                                                  |                                                         |  |
|:-----------------------------------------------------------------------------------------|:--------------------------------------------------------|:------------------|
| [<img src=e.png height=16>][As0] [`GetPreferredAggregator`][A0]`(account)`               | Gets an account's preferred aggregator                  |                   |
| [<img src=e.png height=16>][As1] [`SetPreferredAggregator`][A1]`(aggregator)`            | Sets the caller's preferred aggregator to that provided |                   |
| [<img src=e.png height=16>][As2] [`GetDefaultAggregator`][A2]`()`                        | Gets the chain's default aggregator                     |                   |
| [<img src=e.png height=16>][As3] [`SetDefaultAggregator`][A3]`(default)`                 | Sets the chain's default aggregator                     | ✔️                 |
| [<img src=e.png height=16>][As4] [`GetCompressionRatio`][A4]`(aggregator)`               | Gets the aggregator's compression ratio                 |                   |
| [<img src=e.png height=16>][As5] [`SetCompressionRatio`][A5]`(aggregator, ratio)`        | Set the aggregator's compression ratio                  | ✔️                 |
| [<img src=e.png height=16>][As6] [`GetFeeCollector`][A6]`(aggregator)`                   | Gets an aggregator's fee collector                      |                   |
| [<img src=e.png height=16>][As7] [`SetFeeCollector`][A7]`(aggregator, collector)` &nbsp; | Sets an aggregator's fee collector                      | ✔️                 |
| [<img src=e.png height=16>][As8] [`GetTxBaseFee`][A8]`(aggregator)`                      | Gets an aggregator's current fixed fee to submit a tx   |                   |
| [<img src=e.png height=16>][As9] [`SetTxBaseFee`][A9]`(aggregator, fee)`                 | Sets an aggregator's fixed fee                          | ✔️                 |

[A0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L22
[A1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L39
[A2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L48
[A3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L53
[A4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L70
[A5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L75
[A6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L87
[A7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L92
[A8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L104
[A9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbAggregator.go#L109

[As0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L28
[As1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L32
[As2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L35
[As3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L40
[As4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L45
[As5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L51
[As6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L56
[As7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L62
[As8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L66
[As9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbAggregator.sol#L73


# [ArbBLS][ArbBLS_link]<a name=ArbBLS></a>
Provides a registry of BLS public keys for accounts.

| Methods                                                                            |                                                             |
|:-----------------------------------------------------------------------------------|:------------------------------------------------------------|
| [<img src=e.png height=16>][Bs0] [`RegisterAltBN128`][B0]`(x0, x1, y0, y1)` &nbsp; | Associate an AltBN128 public key with the caller's address  |
| [<img src=e.png height=16>][Bs1] [`GetAltBN128`][B1]`(account)`                    | Gets the AltBN128 public key associated with an address     |
| [<img src=e.png height=16>][Bs2] [`RegisterBLS12381`][B2]`(key)`                   | Associate a BLS 12-381 public key with the caller's address |
| [<img src=e.png height=16>][Bs3] [`GetBLS12381`][B3]`(account)`                    | Gets the BLS 12-381 public key associated with an address   |

[B0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbBLS.go#L27
[B1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbBLS.go#L32
[B2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbBLS.go#L37
[B3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbBLS.go#L46

[Bs0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbBLS.sol#L44
[Bs1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbBLS.sol#L52
[Bs2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbBLS.sol#L63
[Bs3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbBLS.sol#L66

| Deprecated Methods                                                           |                                |
|:-----------------------------------------------------------------------------|:-------------------------------|
| [<img src=e.png height=16>][Bds0] [`Register`][Bd0]`(x0, x1, y0, y1)` &nbsp; | equivalent to registerAltBN128 |
| [<img src=e.png height=16>][Bds1] [`GetPublicKey`][Bd1]`(account)`           | equivalent to getAltBN128      |

[Bd0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbBLS.go#L17
[Bd1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbBLS.go#L22

[Bds0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbBLS.sol#L25
[Bds1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbBLS.sol#L33


# [ArbDebug][ArbDebug_link]<a name=ArbDebug></a>
Provides mechanisms useful for testing. The methods of `ArbDebug` are only available for chains with the `AllowDebugPrecompiles` chain parameter set. Otherwise, calls to this precompile will revert.

| Methods                                                               |                                                    |
|:----------------------------------------------------------------------|:---------------------------------------------------|
| [<img src=e.png height=16>][Ds0] [`BecomeChainOwner`][D0]`()`         | Caller becomes a chain owner                       |
| [<img src=e.png height=16>][Ds1] [`Events`][D1]`(flag, value)` &nbsp; | Emit events with values based on the args provided |

[D0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbDebug.go#L38
[D1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbDebug.go#L19

[Ds0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbDebug.sol#L27
[Ds1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbDebug.sol#L30


| Events                                                  |                                            |
|:--------------------------------------------------------|:-------------------------------------------|
| [<img src=e.png height=16>][Des0] [`Basic`][De0] &nbsp; | Emitted in `Events` for testing            |
| [<img src=e.png height=16>][Des1] [`Mixed`][De1]        | Emitted in `Events` for testing            |
| [<img src=e.png height=16>][Des2] [`Store`][De2]        | Never emitted (used for testing log sizes) |

[De0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbDebug.go#L24
[De1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbDebug.go#L29
[De2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbDebug.go#L13

[Des0]:https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbDebug.sol#L33
[Des1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbDebug.sol#L34
[Des2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbDebug.sol#L41


# [ArbFunctionTable][ArbFunctionTable_link]<a name=ArbFunctionTable></a>
Provided aggregator's the ability to manage function tables, to enable one form of transaction compression. The Nitro aggregator implementation does not use these, so these methods have been stubbed and their effects disabled. They are kept for backwards compatibility.

| Methods                                                                 |                                            |
|:------------------------------------------------------------------------|:-------------------------------------------|
| [<img src=e.png height=16>][FTs0] [`Get`][FT0]`(address, index)` &nbsp; | Reverts since the table is empty           |
| [<img src=e.png height=16>][FTs1] [`Size`][FT1]`(address)`              | Returns the empty table's size, which is 0 |
| [<img src=e.png height=16>][FTs2] [`Upload`][FT2]`(bytes)`              | Does nothing                               |

[FT0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbFunctionTable.go#L30
[FT1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbFunctionTable.go#L25
[FT2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbFunctionTable.go#L20

[FTs0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbFunctionTable.sol#L35
[FTs1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbFunctionTable.sol#L32
[FTs2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbFunctionTable.sol#L29


# [ArbGasInfo][ArbGasInfo_link]<a name=ArbGasInfo></a>
Provides insight into the cost of using the chain. These methods have been adjusted to account for Nitro's heavy use of calldata compression. Of note to end-users, we no longer make a distinction between non-zero and zero-valued calldata bytes.

| Methods                                                                                  |                                                                                                  |
|:-----------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------|
| [<img src=e.png height=16>][GIs0] [`GetPricesInWeiWithAggregator`][GI0]`(aggregator)`    | Get prices in wei when using the provided aggregator                                             |
| [<img src=e.png height=16>][GIs1] [`GetPricesInWei`][GI1]`()`                            | Get prices in wei when using the caller's preferred aggregator                                   |
| [<img src=e.png height=16>][GIs2] [`GetPricesInArbGasWithAggregator`][GI2]`(aggregator)` | Get prices in ArbGas when using the provided aggregator                                          |
| [<img src=e.png height=16>][GIs3] [`GetPricesInArbGas`][GI3]`()`                         | Get prices in ArbGas when using the caller's preferred aggregator                                |
| [<img src=e.png height=16>][GIs4] [`GetGasAccountingParams`][GI4]`()`                    | Get the chain speed limit, pool size, and tx gas limit                                           |
| [<img src=e.png height=16>][GIs5] [`GetMinimumGasPrice`][GI5]`()`                        | Get the minimum gas price needed for a transaction to succeed                                    |
| [<img src=e.png height=16>][GIs6] [`GetGasPoolSeconds`][GI6]`()`                         | Get the number of seconds worth of the speed limit the gas pool contains                         |
| [<img src=e.png height=16>][GIs7] [`GetGasPoolTarget`][GI7]`()`                          | Get the target fullness in bips the pricing model will try to keep the pool at                   |
| [<img src=e.png height=16>][GIs8] [`GetGasPoolWeight`][GI8]`()`                          | Get the extent in bips to which the pricing model favors filling the pool over increasing speeds |
| [<img src=e.png height=16>][GIs9] [`GetRateEstimate`][GI9]`()`                           | Get ArbOS's estimate of the amount of gas being burnt per second                                 |
| [<img src=e.png height=16>][GIs10] [`GetRateEstimateInertia`][GI10]`()`                  | Get how slowly ArbOS updates its estimate the amount of gas being burnt per second               |
| [<img src=e.png height=16>][GIs11] [`GetL1BaseFeeEstimate`][GI11]`()`                    | Get ArbOS's estimate of the L1 basefee in wei                                                    |
| [<img src=e.png height=16>][GIs12] [`GetL1BaseFeeEstimateInertia`][GI12]`()`             | Get how slowly ArbOS updates its estimate of the L1 basefee                                      |
| [<img src=e.png height=16>][GIs13] [`GetL1GasPriceEstimate`][GI13]`()`                   | Deprecated -- Same as getL1BaseFeeEstimate()                                                     |
| [<img src=e.png height=16>][GIs14] [`GetCurrentTxL1GasFees`][GI14]`()`                   | Get L1 gas fees paid by the current transaction                                                  |


[GI0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L27
[GI1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L63
[GI2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L75
[GI3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L99
[GI4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L111
[GI5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L120
[GI6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L125
[GI7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L130
[GI8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L135
[GI9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L140
[GI10]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L145
[GI11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L150
[GI12]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L155
[GI13]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L160
[GI14]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L165

[GIs0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L36
[GIs1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L58
[GIs2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L72
[GIs3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L83
[GIs4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L94
[GIs5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L104
[GIs6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L107
[GIs7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L110
[GIs8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L113
[GIs9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L116
[GIs10]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L119
[GIs11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L122
[GIs12]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L125
[GIs13]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L128
[GIs14]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L131


# [ArbInfo][ArbInfo_link]<a name=ArbInfo></a>
Provides the ability to lookup basic info about accounts and contracts.

| Methods                                                               |                                      |
|:----------------------------------------------------------------------|:-------------------------------------|
| [<img src=e.png height=16>][Is0] [`GetBalance`][I0]`(account)` &nbsp; | Retrieves an account's balance       |
| [<img src=e.png height=16>][Is1] [`GetCode`][I1]`(account)`           | Retrieves a contract's deployed code |

[I0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbInfo.go#L18
[I1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbInfo.go#L26

[Is0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbInfo.sol#L25
[Is1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbInfo.sol#L28


# [ArbosTest][ArbosTest_link]<a name=ArbosTest></a>
Provides a method of burning arbitrary amounts of gas, which exists for historical reasons. In Classic, `ArbosTest` had additional methods only the zero address could call. These have been removed since users don't use them and calls to missing methods revert.

| Methods                                                              |                                                     | Nitro changes |
|:---------------------------------------------------------------------|:----------------------------------------------------|---------------|
| [<img src=e.png height=16>][Ts0] [`BurnArbGas`][T0]`(amount)` &nbsp; | unproductively burns the amount of L2 ArbGas &nbsp; | Now pure      |

[T0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbosTest.go#L17

[Ts0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbosTest.sol#L27


# [ArbOwner][ArbOwner_link]<a name=ArbOwner></a>
Provides owners with tools for managing the rollup. Calls by non-owners will always revert.

Most of Arbitrum Classic's owner methods have been removed since they no longer make sense in Nitro:

- What were once chain parameters are now parts of ArbOS's state, and those that remain are set at genesis. 
- ArbOS upgrades happen with the rest of the system rather than being independent
- Exemptions to address aliasing are no longer offered. Exemptions were intended to support backward compatibility for contracts deployed before aliasing was introduced, but no exemptions were ever requested.

| Methods                                                                         |                                                                                                  |
|:--------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------|
| [<img src=e.png height=16>][Os0] [`AddChainOwner`][O0]`(account)`               | Add account as a chain owner                                                                     |
| [<img src=e.png height=16>][Os1] [`RemoveChainOwner`][O1]`(account)`            | Remove account from the list of chain owners                                                     |
| [<img src=e.png height=16>][Os2] [`IsChainOwner`][O2]`(account)`                | See if account is a chain owner                                                                  |
| [<img src=e.png height=16>][Os3] [`GetAllChainOwners`][O3]`()`                  | Retrieves the list of chain owners                                                               |
| [<img src=e.png height=16>][Os4] [`SetL1BaseFeeEstimate`][O4]`(price)`          | Set the L1 basefee estimate directly, bypassing the autoregression                               |
| [<img src=e.png height=16>][Os5] [`SetL1BaseFeeEstimateInertia`][O5]`(inertia)` | Set how slowly ArbOS updates its estimate of the L1 basefee                                      |
| [<img src=e.png height=16>][Os6] [`SetL2GasPrice`][O6]`(price)`                 | Set the L2 gas price directly, bypassing the pool calculus                                       |
| [<img src=e.png height=16>][Os7] [`SetMinimumGasPrice`][O7]`(price)`            | Set the minimum gas price needed for a transaction to succeed                                    |
| [<img src=e.png height=16>][Os8] [`SetSpeedLimit`][O8]`(limit)`                 | Set the computational speed limit for the chain                                                  |
| [<img src=e.png height=16>][Os9] [`SetGasPoolSeconds`][O9]`(seconds)`           | Set the number of seconds worth of the speed limit the gas pool contains                         |
| [<img src=e.png height=16>][Os10] [`SetGasPoolTarget`][O10]`(target)`           | Set the target fullness in bips the pricing model will try to keep the pool at                   |
| [<img src=e.png height=16>][Os11] [`SetGasPoolWeight`][O11]`(weight)`           | Set the extent in bips to which the pricing model favors filling the pool over increasing speeds |
| [<img src=e.png height=16>][Os12] [`SetRateEstimateInertia`][O12]`(inertia)`    | Set how slowly ArbOS updates its estimate the amount of gas being burnt per second               |
| [<img src=e.png height=16>][Os13] [`SetMaxTxGasLimit`][O13]`(limit)`            | Set the maximum size a tx (and block) can be                                                     |
| [<img src=e.png height=16>][Os14] [`GetNetworkFeeAccount`][O14]`()`             | Get the network fee collector                                                                    |
| [<img src=e.png height=16>][Os15] [`SetNetworkFeeAccount`][O15]`(account)`      | Set the network fee collector                                                                    |

[O0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L24
[O1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L29
[O2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L38
[O3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L43
[O4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L48
[O5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L53
[O6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L58
[O7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L63
[O8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L68
[O9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L73
[O10]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L78
[O11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L83
[O12]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L88
[O13]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L93
[O14]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L98
[O15]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwner.go#L103

[Os0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L30
[Os1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L33
[Os2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L36
[Os3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L39
[Os4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L42
[Os5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L45
[Os6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L48
[Os7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L51
[Os8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L54
[Os9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L57
[Os10]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L60
[Os11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L63
[Os12]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L66
[Os13]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L69
[Os14]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L72
[Os15]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L75

| Events                                                      |                                                           |
|:------------------------------------------------------------|:----------------------------------------------------------|
| [<img src=e.png height=16>][Oes0] [`OwnerActs`][Oe0] &nbsp; | Emitted when a successful call is made to this precompile |

[Oe0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/wrapper.go#L105

[Oes0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwner.sol#L78


# [ArbOwnerPublic][ArbOwnerPublic_link]<a name=ArbOwnerPublic></a>
Provides non-owners with info about the current chain owners.

| Methods                                                                   |                                 |
|:--------------------------------------------------------------------------|:--------------------------------|
| [<img src=e.png height=16>][OPs0] [`IsChainOwner`][OP0]`(account)` &nbsp; | See if account is a chain owner |
| [<img src=e.png height=16>][OPs1] [`GetAllChainOwners`][OP1]`()`          | Gets the list of chain owners   |
| [<img src=e.png height=16>][OPs2] [`GetNetworkFeeAccount`][OP2]`()`       | Gets the network fee collector  |

[OP0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwnerPublic.go#L24
[OP1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwnerPublic.go#L19
[OP2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbOwnerPublic.go#L29

[OPs0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwnerPublic.sol#L25
[OPs1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwnerPublic.sol#L28
[OPs2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbOwnerPublic.sol#L31


# [ArbRetryableTx][ArbRetryableTx_link]<a name=ArbRetryableTx></a>
Provides methods for managing retryables. The model has been adjusted for Nitro, most notably in terms of how retry transactions are scheduled. For more information on retryables, please see [the retryable documentation](ArbOS.md#Retryables).


| Methods                                                                    |                                                                                    | Nitro changes          |
|:---------------------------------------------------------------------------|:-----------------------------------------------------------------------------------|:-----------------------|
| [<img src=e.png height=16>][RTs0] [`Cancel`][RT0]`(ticket)`                | Cancel the ticket and refund its callvalue to its beneficiary                      |                        |
| [<img src=e.png height=16>][RTs1] [`GetBeneficiary`][RT1]`(ticket)` &nbsp; | Gets the beneficiary of the ticket                                                 |                        |
| [<img src=e.png height=16>][RTs2] [`GetLifetime`][RT2]`()`                 | Gets the default lifetime period a retryable has at creation                       | Reverts when not found |
| [<img src=e.png height=16>][RTs3] [`GetTimeout`][RT3]`(ticket)`            | Gets the timestamp for when ticket will expire                                     |                        |
| [<img src=e.png height=16>][RTs4] [`Keepalive`][RT4]`(ticket)`             | Adds one lifetime period to the ticket's expiry                                    | Doesn't add callvalue  |
| [<img src=e.png height=16>][RTs5] [`Redeem`][RT5]`(ticket)`                | Schedule an attempt to redeem the retryable, donating all of the call's gas &nbsp; | Happens in a future tx |

[RT0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L184
[RT1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L171
[RT2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L110
[RT3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L115
[RT4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L132
[RT5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L36

[RTs0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L70
[RTs1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L63
[RTs2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L38
[RTs3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L45
[RTs4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L55
[RTs5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L32

| Events                                                               |                                                    | Nitro Changes                       |
|:---------------------------------------------------------------------|:---------------------------------------------------|:------------------------------------|
| [<img src=e.png height=16>][RTes0] [`TicketCreated`][RTe0]           | Emitted when creating a retryable                  |                                     |
| [<img src=e.png height=16>][RTes1] [`LifetimeExtended`][RTe1] &nbsp; | Emitted when extending a retryable's expiry &nbsp; |                                     |
| [<img src=e.png height=16>][RTes2] [`RedeemScheduled`][RTe2]         | Emitted when scheduling a retryable                | Replaces [Redeemed][old_event_link] |
| [<img src=e.png height=16>][RTes3] [`Canceled`][RTe3]                | Emitted when cancelling a retryable                |                                     |

[RTe0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/arbos/tx_processor.go#L143
[RTe1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L163
[RTe2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/arbos/tx_processor.go#L186
[RTe3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbRetryableTx.go#L209

[RTes0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L72
[RTes1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L73
[RTes2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L74
[RTes3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbRetryableTx.sol#L81

[old_event_link]: https://github.com/OffchainLabs/arb-os/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/arb_os/arbretryable.mini#L90

# [ArbStatistics][ArbStatistics_link]<a name=ArbStatistics></a>
Provides statistics about the chain as of just before the Nitro upgrade. In Arbitrum Classic, this was how a user would get info such as the total number of accounts, but there are better ways to get that info in Nitro.

| Methods                                                        |                                                                                         |
|:---------------------------------------------------------------|:----------------------------------------------------------------------------------------|
| [<img src=e.png height=16>][STs0] [`GetStats`][ST0]`()` &nbsp; | Returns the current block number and some statistics about the rollup's pre-Nitro state |

[ST0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbStatistics.go#L19

[STs0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbStatistics.sol#L32


# [ArbSys][ArbSys_link]<a name=ArbSys></a>
Provides system-level functionality for interacting with L1 and understanding the call stack.

| Methods                                                                                                 |                                                                                                              | Nitro changes     |
|:--------------------------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------|:------------------|
| [<img src=e.png height=16>][Ss0] [`ArbBlockNumber`][S0]`()`                                             | Gets the current L2 block number                                                                             |                   |
| [<img src=e.png height=16>][Ss1] [`ArbBlockHash`][S1]`(blocknum)`                                       | Gets the L2 block hash at blocknum, if blocknum is sufficiently recent                                       |                   |
| [<img src=e.png height=16>][Ss2] [`ArbChainID`][S2]`()`                                                 | Gets the chain's ChainID                                                                                     |                   |
| [<img src=e.png height=16>][Ss3] [`ArbOSVersion`][S3]`()`                                               | Gets the current ArbOS version                                                                               | Now view          |
| [<img src=e.png height=16>][Ss4] [`GetStorageGasAvailable`][S4]`()`                                     | Returns 0 since Nitro has no concept of storage gas                                                          | Now always 0      |
| [<img src=e.png height=16>][Ss5] [`IsTopLevelCall`][S5]`()`                                             | Checks if the caller is top-level (i.e. if the caller was called directly by an EOA or an L1 contract)       |                   |
| [<img src=e.png height=16>][Ss6] [`MapL1SenderContractAddressToL2Alias`][S6]`(contract, unused)` &nbsp; | Gets contract's L2 alias                                                                                     | 2nd arg is unused |
| [<img src=e.png height=16>][Ss7] [`WasMyCallersAddressAliased`][S7]`()`                                 | Checks if the caller's caller was aliased                                                                    |                   |
| [<img src=e.png height=16>][Ss8] [`MyCallersAddressWithoutAliasing`][S8]`()`                            | Gets the caller's caller without any potential address aliasing                                              | New outbox scheme |
| [<img src=e.png height=16>][Ss9] [`SendTxToL1`][S9]`(destination, calldataForL1)`                       | Sends a transaction to L1, adding it to the outbox; callvalue is sent to L1 attached to the sent transaction | New outbox scheme |
| [<img src=e.png height=16>][Ss10] [`SendMerkleTreeState`][S10]`()`                                      | Gets the root, size, and partials of the outbox Merkle tree state                                            | New outbox scheme |
| [<img src=e.png height=16>][Ss11] [`WithdrawEth`][S11]`(destination)`                                   | Send callvalue to the destination address on L1                                                              |                   |

[S0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L30
[S1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L35
[S2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L50
[S3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L55
[S4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L61
[S5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L66
[S6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L71
[S7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L76
[S8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L82
[S9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L98
[S10]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L171
[S11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L187

[Ss0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L31
[Ss1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L37
[Ss2]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L43
[Ss3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L49
[Ss4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L55
[Ss5]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L61
[Ss6]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L69
[Ss7]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L78
[Ss8]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L84
[Ss9]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L100
[Ss10]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L111
[Ss11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L92


| Events                                                              |                                                                 |
|:--------------------------------------------------------------------|:----------------------------------------------------------------|
| [<img src=e.png height=16>][Ses0] [`L2ToL1Transaction`][Se0] &nbsp; | Logs a send tx from L2 to L1, including data for outbox proving |
| [<img src=e.png height=16>][Ses1] [`SendMerkleUpdate`][Se1]         | Logs a new merkle branch needed for constructing outbox proofs  |

[Se0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L152
[Se1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbSys.go#L138

[Ses0]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L124
[Ses1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbSys.sol#L143

| Removed Methods                                                                  |                                                                   |
|:---------------------------------------------------------------------------------|:------------------------------------------------------------------|
| [<img src=e.png height=16>][Srs0] [`GetStorageAt`][Sr0]`(account, index)` &nbsp; | Nitro doesn't need this introspection, and users couldn't call it |
| [<img src=e.png height=16>][Srs1] [`GetTransactionCount`][Sr1]`(account)`        | Nitro doesn't need this introspection, and users couldn't call it |

[Sr0]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/arb_os/arbsys.mini#L335
[Sr1]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/arb_os/arbsys.mini#L315

[Srs0]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/contracts/arbos/builtin/ArbSys.sol#L51
[Srs1]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/contracts/arbos/builtin/ArbSys.sol#L42
