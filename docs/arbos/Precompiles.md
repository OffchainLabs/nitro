# Overview
ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This reference exhaustively documents the specific calls ArbOS makes available. For more details on the infrastructure that makes this possible, please refer to the [ArbOS documentation](ArbOS.md). For an abbreviated reference on the precompiles we expect users to most often use, please see the [common precompiles documentation](CommonPrecompiles.md).

From the perspective of user applications, precompiles live as contracts at the following addresses. Click on any to jump to their section.

| Precompile                                     | Address &nbsp; | Purpose                            |
| :--------------------------------------------- | :------------- | :--------------------------------- |
| [`ArbAddressTable`](#ArbAddressTable)          | `0x66`         | Saving calldata costs for accounts |
| [`ArbAggregator`](#ArbAggregator)              | `0x6d`         | Configuring aggregation            |
| [`ArbBLS`](#ArbBLS)                            | `0x67`         | Managing BLS keys                  |
| [`ArbDebug`](#ArbDebug)                        | `0xff`         | Testing tools                      |
| [`ArbFunctionTable`](#ArbFunctionTable) &nbsp; | `0x68`         | No longer used                     |
| [`ArbGasInfo`](#ArbGasInfo)                    | `0x6c`         | Info about gas pricing             |
| [`ArbInfo`](#ArbInfo)                          | `0x65`         | Info about accounts                |
| [`ArbOwner`](#ArbOwner)                        | `0x70`         | Owner operations                   |
| [`ArbOwnerPublic`](#ArbOwnerPublic)            | `0x6b`         | Info about owners                  |
| [`ArbosTest`](#ArbosTest)                      | `0x69`         | No longer used                     |
| [`ArbRetryableTx`](#ArbRetryableTx)            | `0x6e`         | Managing retryables                |
| [`ArbStatistics`](#ArbStatistics)              | `0x6f`         | Info about the pre-Nitro state     |
| [`ArbSys`](#ArbSys)                            | `0x64`         | System-level functionality         |

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

| Methods                                                                    |                                                                                           |
|:---------------------------------------------------------------------------|:------------------------------------------------------------------------------------------|
| [<img src=e height=16>][ATs0] [`AddressExists`][AT0]`(address)`            | Check if an address exists in the table                                                   |
| [<img src=e height=16>][ATs1] [`Compress`][AT1]`(address)`                 | Gets bytes that represent the address                                                     |
| [<img src=e height=16>][ATs2] [`Decompress`][AT2]`(buffer, offset)` &nbsp; | Replaces the compressed bytes at the given offset with those of the corresponding account |
| [<img src=e height=16>][ATs3] [`Lookup`][AT3]`(address)`                   | Looks up the index of an address in the table                                             |
| [<img src=e height=16>][ATs4] [`LookupIndex`][AT4]`(index)`                | Looks up an address in the table by index                                                 |
| [<img src=e height=16>][ATs5] [`Register`][AT5]`(address)`                 | Adds an account to the table, shrinking its compressed representation                     |
| [<img src=e height=16>][ATs6] [`Size`][AT6]`()`                            | Gets the number of addresses in the table                                                 |

[AT0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAddressTable.go#L18
[AT1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAddressTable.go#L23
[AT2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAddressTable.go#L28
[AT3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAddressTable.go#L41
[AT4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAddressTable.go#L53
[AT5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAddressTable.go#L68
[AT6]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAddressTable.go#L74

[ATs0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAddressTable.sol#L12
[ATs1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAddressTable.sol#L19
[ATs2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAddressTable.sol#L27
[ATs3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAddressTable.sol#L33
[ATs4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAddressTable.sol#L39
[ATs5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAddressTable.sol#L46
[ATs6]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAddressTable.sol#L51


# [ArbAggregator][ArbAggregator_link]<a name=ArbAggregator></a>
Provides aggregators and their users methods for configuring how they participate in L1 aggregation. Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless `SetPreferredAggregator` is invoked to change it.

| Methods                                                                              |                                                               | Caller must be                |
|:-------------------------------------------------------------------------------------|:--------------------------------------------------------------|:------------------------------|
| [<img src=e height=16>][As0] [`GetPreferredAggregator`][A0]`(account)`               | Gets an account's preferred aggregator                        |                               |
| [<img src=e height=16>][As1] [`SetPreferredAggregator`][A1]`(aggregator)`            | Sets the caller's preferred aggregator to that provided       |                               |
| [<img src=e height=16>][As2] [`GetDefaultAggregator`][A2]`()`                        | Gets the rollup's default aggregator                          |                               |
| [<img src=e height=16>][As3] [`SetDefaultAggregator`][A3]`(default)`                 | Sets the rollup's default aggregator                          | chain owner or the current    |
| [<img src=e height=16>][As4] [`GetFeeCollector`][A4]`(aggregator)`                   | Gets an aggregator's fee collector                            |                               |
| [<img src=e height=16>][As5] [`SetFeeCollector`][A5]`(aggregator, collector)` &nbsp; | Sets an aggregator's fee collector                            | the aggregator                |
| [<img src=e height=16>][As6] [`GetTxBaseFee`][A6]`(aggregator)`                      | Gets an aggregator's current fixed cost charge to submit a tx |                               |
| [<img src=e height=16>][As7] [`SetTxBaseFee`][A7]`(aggregator, fee)`                 | Sets an aggregator's fixed cost                               | chain owner or the aggergator |

[A0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L19
[A1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L24
[A2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L29
[A3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L34
[A4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L57
[A5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L62
[A6]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L76
[A7]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L81

[As0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L7
[As1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L11
[As2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L14
[As3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L18
[As4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L22
[As5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L27
[As6]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L30
[As7]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L35


# [ArbBLS][ArbBLS_link]<a name=ArbBLS></a>
Provides a registry of BLS public keys for accounts.

| Methods                                                                |                                                       |
|:-----------------------------------------------------------------------|:------------------------------------------------------|
| [<img src=e height=16>][Bs0] [`GetPublicKey`][B0]`(account)`           | Retrieves the BLS public key for the account provided |
| [<img src=e height=16>][Bs1] [`Register`][B1]`(x0, x1, y0, y1)` &nbsp; | Sets the caller's BLS public key                      |

[B0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbBLS.go#L13
[B1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbBLS.go#L18

[Bs0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbBLS.sol#L5
[Bs1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbBLS.sol#L8


# [ArbDebug][ArbDebug_link]<a name=ArbDebug></a>
Provides mechanisms useful for testing. The methods of `ArbDebug` are only available for rollups with the `AllowDebugPrecompiles` chain parameter set. Otherwise, calls to this precompile will revert.

| Methods                                                           |                                                    |
|:------------------------------------------------------------------|:---------------------------------------------------|
| [<img src=e height=16>][Ds0] [`BecomeChainOwner`][D0]`()`         | Caller becomes a chain owner                       |
| [<img src=e height=16>][Ds1] [`Events`][D1]`(flag, value)` &nbsp; | Emit events with values based on the args provided |

[D0]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/precompiles/ArbDebug.go#L37
[D1]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/precompiles/ArbDebug.go#L19

[Ds0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbDebug.go#L38
[Ds1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbDebug.go#L19


| Events                                              |                                            |
|:----------------------------------------------------|:-------------------------------------------|
| [<img src=e height=16>][Des0] [`Basic`][De0] &nbsp; | Emitted in `Events` for testing            |
| [<img src=e height=16>][Des1] [`Mixed`][De1]        | Emitted in `Events` for testing            |
| [<img src=e height=16>][Des2] [`Store`][De2]        | Never emitted (used for testing log sizes) |

[De0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbDebug.go#L24
[De1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbDebug.go#L29
[De2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbDebug.go#L13

[Des0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbDebug.sol#L8
[Des1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbDebug.sol#L9
[Des2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbDebug.sol#L10


# [ArbFunctionTable][ArbFunctionTable_link]<a name=ArbFunctionTable></a>
Provided aggregator's the ability to manage function tables. Aggregation works differently in Nitro, so these methods have been stubbed and their effects disabled. They are kept for backwards compatibility.

| Methods                                                             |                                            |
|:--------------------------------------------------------------------|:-------------------------------------------|
| [<img src=e height=16>][FTs0] [`Get`][FT0]`(address, index)` &nbsp; | Reverts since the table is empty           |
| [<img src=e height=16>][FTs1] [`Size`][FT1]`(address)`              | Returns the empty table's size, which is 0 |
| [<img src=e height=16>][FTs2] [`Upload`][FT2]`(bytes)`              | Does nothing                               |

[FT0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbFunctionTable.go#L30
[FT1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbFunctionTable.go#L25
[FT2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbFunctionTable.go#L20

[FTs0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbFunctionTable.sol#L15
[FTs1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbFunctionTable.sol#L12
[FTs2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbFunctionTable.sol#L9


# [ArbGasInfo][ArbGasInfo_link]<a name=ArbGasInfo></a>
Provides insight into the cost of using the rollup. These methods have been adjusted to account for Nitro's heavy use of calldata compression. Of note to end-users, we no longer make a distinction between non-zero and zero-valued calldata bytes.

| Methods                                                                                     |                                                                    |
|:--------------------------------------------------------------------------------------------|:-------------------------------------------------------------------|
| [<img src=e height=16>][GIs0] [`GetPricesInWeiWithAggregator`][GI0]`(aggregator)`           | Gets prices in wei when using the provided aggregator              |
| [<img src=e height=16>][GIs1] [`GetPricesInWei`][GI1]`()`                                   | Gets prices in wei when using the caller's preferred aggregator    |
| [<img src=e height=16>][GIs2] [`GetPricesInArbGasWithAggregator`][GI2]`(aggregator)` &nbsp; | Gets prices in ArbGas when using the provided aggregator           |
| [<img src=e height=16>][GIs3] [`GetPricesInArbGas`][GI3]`()`                                | Gets prices in ArbGas when using the caller's preferred aggregator |
| [<img src=e height=16>][GIs4] [`GetGasAccountingParams`][GI4]`()`                           | Gets the rollup's speed limit, pool size, and tx gas limit         |
| [<img src=e height=16>][GIs5] [`GetL1GasPriceEstimate`][GI5]`()`                            | Gets the current estimate of the L1 gas price                      |
| [<img src=e height=16>][GIs6] [`GetCurrentTxL1GasFees`][GI6]`()`                            | Gets the fee paid to the aggregator for posting this tx            |

[GI0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L26
[GI1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L62
[GI2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L71
[GI3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L95
[GI4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L104
[GI5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L112
[GI6]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L117

[GIs0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L13
[GIs1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L17
[GIs2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L21
[GIs3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L25
[GIs4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L28
[GIs5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L31
[GIs6]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L34


# [ArbInfo][ArbInfo_link]<a name=ArbInfo></a>
Provides the ability to lookup basic info about accounts and contracts.

| Methods                                                           |                                       |
|:------------------------------------------------------------------|:--------------------------------------|
| [<img src=e height=16>][Is0] [`GetBalance`][I0]`(account)` &nbsp; | Retrieves an account's balance        |
| [<img src=e height=16>][Is1] [`GetCode`][I1]`(account)`           | Retrieves a contract's source program |

[I0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbInfo.go#L18
[I1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbInfo.go#L26

[Is0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbInfo.sol#L5
[Is1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbInfo.sol#L8


# [ArbosTest][ArbosTest_link]<a name=ArbosTest></a>
Provides a method of burning arbitrary amounts of gas, which exists for historical reasons. In Classic, `ArbosTest` had additional methods only the zero address could call. These have been removed since users don't use them and calls to missing methods revert.

| Methods                                                          |                                                     | Nitro changes |
|:-----------------------------------------------------------------|:----------------------------------------------------|---------------|
| [<img src=e height=16>][Ts0] [`BurnArbGas`][T0]`(amount)` &nbsp; | unproductively burns the amount of L2 ArbGas &nbsp; | Now pure      |

[T0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbosTest.go#L17

[Ts0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbosTest.sol#L4


# [ArbOwner][ArbOwner_link]<a name=ArbOwner></a>
Provides owners with tools for managing the rollup. Calls by non-owners will always revert.

Most of Classic's owner methods have been removed since they no longer make sense in Nitro:

- What were once chain parameters are now parts of ArbOS's state, and those that remain are set at genesis. 
- ArbOS upgrades happen with the rest of the system rather than being independent
- No one asked to be exept from address aliasing so address remapping is unconditional

| Methods                                                                    |                                                                       |
|:---------------------------------------------------------------------------|:----------------------------------------------------------------------|
| [<img src=e height=16>][Os0] [`AddChainOwner`][O0]`(account)`              | Promotes the user to chain owner                                      |
| [<img src=e height=16>][Os1] [`RemoveChainOwner`][O1]`(account)`           | Demotes the user from chain owner                                     |
| [<img src=e height=16>][Os2] [`IsChainOwner`][O2]`(account)`               | See if the user is a chain owner                                      |
| [<img src=e height=16>][Os3] [`GetAllChainOwners`][O3]`()`                 | Retrieves the list of chain owners                                    |
| [<img src=e height=16>][Os4] [`SetL1GasPriceEstimate`][O4]`(price)` &nbsp; | Sets the L1 gas price estimate directly, bypassing the autoregression |
| [<img src=e height=16>][Os5] [`SetL2GasPrice`][O5]`(price)`                | Sets the L2 gas price directly, bypassing the pool calculus           |

[O0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwner.go#L22
[O1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwner.go#L27
[O2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwner.go#L36
[O3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwner.go#L41
[O4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwner.go#L46
[O5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwner.go#L51

[Os0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwner.sol#L8
[Os1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwner.sol#L11
[Os2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwner.sol#L14
[Os3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwner.sol#L17
[Os4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwner.sol#L20
[Os5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwner.sol#L23


# [ArbOwnerPublic][ArbOwnerPublic_link]<a name=ArbOwnerPublic></a>
Provides non-owners with info about the current chain owners.

| Methods                                                               |                                    |
|:----------------------------------------------------------------------|:-----------------------------------|
| [<img src=e height=16>][OPs0] [`GetAllChainOwners`][OP0]`()`          | Retrieves the list of chain owners |
| [<img src=e height=16>][OPs1] [`IsChainOwner`][OP1]`(account)` &nbsp; | See if the user is a chain owner   |

[OP0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwnerPublic.go#L19
[OP1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbOwnerPublic.go#L24

[OPs0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwnerPublic.sol#L10
[OPs1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbOwnerPublic.sol#L7


# [ArbRetryableTx][ArbRetryableTx_link]<a name=ArbRetryableTx></a>
| Methods                                                                |                                |
|:-----------------------------------------------------------------------|:-------------------------------|
| [<img src=e height=16>][RTs0] [`Cancel`][RT0]`(ticket)`                | TODO: document when stabilized |
| [<img src=e height=16>][RTs1] [`GetBeneficiary`][RT1]`(ticket)` &nbsp; | TODO: document when stabilized |
| [<img src=e height=16>][RTs2] [`GetLifetime`][RT2]`()`                 | TODO: document when stabilized |
| [<img src=e height=16>][RTs3] [`GetTimeout`][RT3]`(ticket)`            | TODO: document when stabilized |
| [<img src=e height=16>][RTs4] [`Keepalive`][RT4]`(ticket)`             | TODO: document when stabilized |
| [<img src=e height=16>][RTs5] [`Redeem`][RT5]`(ticket)`                | TODO: document when stabilized |

[RT0]: todo
[RT1]: todo
[RT2]: todo
[RT3]: todo
[RT4]: todo
[RT5]: todo

[RTs0]: todo
[RTs1]: todo
[RTs2]: todo
[RTs3]: todo
[RTs4]: todo
[RTs5]: todo


| Events                                                           |
|:-----------------------------------------------------------------|
| [<img src=e height=16>][RTes0] [`TicketCreated`][RTe0]           |
| [<img src=e height=16>][RTes1] [`LifetimeExtended`][RTe1] &nbsp; |
| [<img src=e height=16>][RTes2] [`RedeemScheduled`][RTe2]         |
| [<img src=e height=16>][RTes3] [`Redeemed`][RTe3]                |
| [<img src=e height=16>][RTes4] [`Canceled`][RTe4]                |

[RTe0]: todo
[RTe1]: todo
[RTe2]: todo
[RTe3]: todo
[RTe4]: todo

[RTes0]: todo
[RTes1]: todo
[RTes2]: todo
[RTes3]: todo
[RTes4]: todo


# [ArbStatistics][ArbStatistics_link]<a name=ArbStatistics></a>
Provides statistics about the rollup right before the Nitro upgrade. In Classic, this was how a user would get info such as the total number of accounts, but there's now better ways to do that with geth.

| Methods                                                    |                                                                                         |
|:-----------------------------------------------------------|:----------------------------------------------------------------------------------------|
| [<img src=e height=16>][STs0] [`GetStats`][ST0]`()` &nbsp; | Returns the current block number and some statistics about the rollup's pre-Nitro state |

[ST0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbStatistics.go#L19

[STs0]: https://github.com/OffchainLabs/nitro/blob/b010c466db6c772b6e4b8f4b79854297177fe457/solgen/src/precompiles/ArbStatistics.sol#L11


# [ArbSys][ArbSys_link]<a name=ArbSys></a>
Provides system-level functionality for interacting with L1 and understanding the call stack.

| Methods                                                                                             |                                                                   | Nitro changes     |
|:----------------------------------------------------------------------------------------------------|:------------------------------------------------------------------|:------------------|
| [<img src=e height=16>][Ss0] [`ArbBlockNumber`][S0]`()`                                             | Gets the current L2 block number                                  |                   |
| [<img src=e height=16>][Ss1] [`ArbBlockHash`][S1]`()`                                               | Gets the L2 block hash, if the block is sufficiently recent       |                   |
| [<img src=e height=16>][Ss2] [`ArbChainID`][S2]`()`                                                 | Gets the rollup's unique chain identifier                         |                   |
| [<img src=e height=16>][Ss3] [`ArbOSVersion`][S3]`()`                                               | Gets the current ArbOS version                                    | Now view          |
| [<img src=e height=16>][Ss4] [`GetStorageGasAvailable`][S4]`()`                                     | Returns 0 since Nitro has no concept of storage gas               | Now always 0      |
| [<img src=e height=16>][Ss5] [`IsTopLevelCall`][S5]`()`                                             | Checks if the call is top-level                                   |                   |
| [<img src=e height=16>][Ss6] [`MapL1SenderContractAddressToL2Alias`][S6]`(contract, unused)` &nbsp; | Gets the contract's L2 alias                                      | 2nd arg is unused |
| [<img src=e height=16>][Ss7] [`WasMyCallersAddressAliased`][S7]`()`                                 | Checks if the caller's caller was aliased                         |                   |
| [<img src=e height=16>][Ss8] [`MyCallersAddressWithoutAliasing`][S8]`()`                            | Gets the caller's caller without any potential aliasing           | New outbox scheme |
| [<img src=e height=16>][Ss9] [`SendTxToL1`][S9]`(destination, calldataForL1)`                       | Sends a transaction to L1, adding it to the outbox                | New outbox scheme |
| [<img src=e height=16>][Ss10] [`SendMerkleTreeState`][S10]`()`                                      | Gets the root, size, and partials of the outbox Merkle tree state | New outbox scheme |
| [<img src=e height=16>][Ss11] [`WithdrawEth`][S11]`(destination)`                                   | Send paid eth to the destination on L1                            |                   |

[S0]: https://github.com/OffchainLabs/nitro/blob/d27b2e270fe0a608ee1b4e2f272b895229a57e0e/precompiles/ArbSys.go#L29
[S1]: https://github.com/OffchainLabs/nitro/blob/d27b2e270fe0a608ee1b4e2f272b895229a57e0e/precompiles/ArbSys.go#L34
[S2]: https://github.com/OffchainLabs/nitro/blob/d27b2e270fe0a608ee1b4e2f272b895229a57e0e/precompiles/ArbSys.go#L49
[S3]: https://github.com/OffchainLabs/nitro/blob/d27b2e270fe0a608ee1b4e2f272b895229a57e0e/precompiles/ArbSys.go#L54
[S4]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L43
[S5]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L48
[S6]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L53
[S7]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L58
[S8]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L64
[S9]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L80
[S10]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L137
[S11]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L153

[Ss0]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L27
[Ss1]: https://github.com/OffchainLabs/nitro/blob/d27b2e270fe0a608ee1b4e2f272b895229a57e0e/solgen/src/precompiles/ArbSys.sol#L33
[Ss2]: https://github.com/OffchainLabs/nitro/blob/d27b2e270fe0a608ee1b4e2f272b895229a57e0e/solgen/src/precompiles/ArbSys.sol#L39
[Ss3]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L39
[Ss4]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L45
[Ss5]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L51
[Ss6]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L59
[Ss7]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L65
[Ss8]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L71
[Ss9]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L87
[Ss10]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L95
[Ss11]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L79


| Events                                                          |                                                                 |
|:----------------------------------------------------------------|:----------------------------------------------------------------|
| [<img src=e height=16>][Ses0] [`L2ToL1Transaction`][Se0] &nbsp; | Logs a send tx from L2 to L1, including data for outbox proving |
| [<img src=e height=16>][Ses1] [`SendMerkleUpdate`][Se1]         | Logs a new merkle branch needed for constructing outbox proofs  |

[Se0]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L118
[Se1]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L98

[Ses0]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L101
[Ses1]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L120

| Removed                                                                      |                                                                             |
|:-----------------------------------------------------------------------------|:----------------------------------------------------------------------------|
| [<img src=e height=16>][Srs0] [`GetStorageAt`][Sr0]`(account, index)` &nbsp; | Nitro doesn't need Classic's `eth_getStorageAt`, and users couldn't call it |
| [<img src=e height=16>][Srs1] [`GetTransactionCount`][Sr1]`(account)`        | Nitro doesn't need Classic's `eth_getStorageAt`, and users couldn't call it |

[Sr0]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/arb_os/arbsys.mini#L335
[Sr1]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/arb_os/arbsys.mini#L315

[Srs0]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/contracts/arbos/builtin/ArbSys.sol#L51
[Srs1]: https://github.com/OffchainLabs/arb-os/blob/89e36db597c4857a4dac3efd7cc01b13c7845cc0/contracts/arbos/builtin/ArbSys.sol#L42
