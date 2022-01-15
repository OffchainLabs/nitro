# Overview
ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This reference exaustively documents the specific calls ArbOS makes available. For more details on the infrustructure that makes this possible, please refer to the [ArbOS documentation](ArbOS.md). For an abbreviated reference on the precompiles we expect users to most often use, please see the [common precompiles documentation](CommonPrecompiles.md)

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
| [`ArbStatistics`](#ArbStatistics)              | `0x6f`         | Info about the pre-nitro state     |
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

| Methods                                      |                                                           |
| :------------------------------------------- | :-------------------------------------------------------- |
| [`AddressExists`][AT0]`(address)`            | Check if an address exists in the table                   |
| [`Compress`][AT1]`(address)`                 | Gets bytes that represent the address                     |
| [`Decompress`][AT2]`(buffer, offset)` &nbsp; | Replaces the compressed bytes at the given offset with those of the corresponding account |
| [`Lookup`][AT3]`(address)`         | Looks up the index of an address in the table                         |
| [`LookupIndex`][AT4]`(index)`      | Looks up an address in the table by index                             |
| [`Register`][AT5]`(address)`       | Adds an account to the table, shrinking its compressed representation |
| [`Size`][AT6]`()`                  | Gets the number of addresses in the table                             |

[AT0]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAddressTable.go#L16
[AT1]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAddressTable.go#L20
[AT2]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAddressTable.go#L24
[AT3]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAddressTable.go#L36
[AT4]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAddressTable.go#L47
[AT5]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAddressTable.go#L61
[AT6]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAddressTable.go#L66

# [ArbAggregator][ArbAggregator_link]<a name=ArbAggregator></a>
Provides aggregators and their users methods for configuring how they participate in L1 aggregation. Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless `SetPreferredAggregator` is invoked to change it.

| Methods                                      |                                        | Caller must be              |
| :------------------------------------------- | :------------------------------------- | :-------------------------- |
| [`GetFeeCollector`][A0]`(aggregator)`        | Gets an aggregator's fee collector     |                             |
| [`SetFeeCollector`][A1]`(aggregator, collector)` &nbsp; | Sets an aggregator's fee collector | the aggregator       |
| [`GetDefaultAggregator`][A2]`()`             | Gets the rollup's default aggregator   |                             |
| [`SetDefaultAggregator`][A3]`(default)`      | Sets the rollup's default aggregator   | chain owner or the current  |
| [`GetPreferredAggregator`][A4]`(account)`    | Gets an account's preferred aggregator |                             |
| [`SetPreferredAggregator`][A5]`(aggregator)` | Sets the caller's preferred aggregator to that provided              |
| [`GetTxBaseFee`][A6]`(aggregator)`           | Gets an aggregator's current fixed cost charge to submit a tx        |
| [`SetTxBaseFee`][A7]`(aggregator, fee)`      | Sets an aggregator's fixed cost      | chain owner or the aggergator |

[A0]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L15
[A1]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L19
[A2]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L32
[A3]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L36
[A4]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L58
[A5]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L62
[A6]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L66
[A7]: https://github.com/OffchainLabs/nitro/blob/0089a13e21ab9cd8b7ba78806a30f626fb2dbc52/precompiles/ArbAggregator.go#L70

# [ArbBLS][ArbBLS_link]<a name=ArbBLS></a>
Provides a registry of BLS public keys for accounts.

| Methods                                    |                                                       |
| :----------------------------------------- | :---------------------------------------------------- |
| [`GetPublicKey`][B0]`(account)`]           | Retrieves the BLS public key for the account provided |
| [`Register`][B1]`(x0, x1, y0, y1)`] &nbsp; | Sets the caller's BLS public key                      |

[B0]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/precompiles/ArbBLS.go#L11
[B1]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/precompiles/ArbBLS.go#L15

# [ArbDebug][ArbDebug_link]<a name=ArbDebug></a>
Provides mechanisms useful for testing. The methods of `ArbDebug` are only available for rollups with the `AllowDebugPrecompiles` chain parameter set. Otherwise, calls to this precompile will revert.

| Methods                              |                                                    |
| :----------------------------------- | :------------------------------------------------- |
| [`BecomeChainOwner`][D0]`()`         | Caller becomes a chain owner                       |
| [`Events`][D1]`(flag, value)` &nbsp; | Emit events with values based on the args provided |

[D0]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/precompiles/ArbDebug.go#L37
[D1]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/precompiles/ArbDebug.go#L19

| Events                |                                            |
| :-------------------- | :----------------------------------------- |
| [`Basic`][De0] &nbsp; | Emitted in `Events` for testing            |
| [`Mixed`][De1]        | Emitted in `Events` for testing            |
| [`Store`][De2]        | Never emitted (used for testing log sizes) |

[De0]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/solgen/src/precompiles/ArbDebug.sol#L8
[De1]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/solgen/src/precompiles/ArbDebug.sol#L9
[De2]: https://github.com/OffchainLabs/nitro/blob/2845cc9a29a9c19e107d56ac2c3980c462e395ed/solgen/src/precompiles/ArbDebug.sol#L10

# [ArbFunctionTable][ArbFunctionTable_link]<a name=ArbFunctionTable></a>
Provided aggregator's the ability to manage function tables. Aggregation works differently in Nitro, so these methods have been stubbed and their effects disabled. They are kept for backwards compatibility.

| Methods                               |                                             |
| :------------------------------------ | :------------------------------------------ |
| [`Get`][FT0]`(address, index)` &nbsp; | Reverts since the table is empty            |
| [`Size`][FT1]`(address)`              | Return's the empty table's size, which is 0 |
| [`Upload`][FT2]`(bytes)`              | Does nothing                                |

[FT0]: todo
[FT1]: todo
[FT2]: todo

# [ArbGasInfo][ArbGasInfo_link]<a name=ArbGasInfo></a>
Provides insight into the cost of using the rollup.

| Methods                                              |                                                       |
| :--------------------------------------------------- | :---------------------------------------------------- |
| [`GetPricesInWeiWithAggregator`][GI0]`(aggregator)`  | Gets prices in wei when using the provided aggregator |
| [`GetPricesInWei`][GI1]`()`                          | Gets prices in wei when using the caller's preferred aggregator |
| [`GetPricesInArbGasWithAggregator`][GI2]`(aggregator)` &nbsp; | Gets prices in ArbGas when using the provided aggregator |
| [`GetPricesInArbGas`][GI3]`()`                       | Gets prices in ArbGas when using the caller's preferred aggregator |
| [`GetGasAccountingParams`][GI4]`()`                  | Gets the rollup's speed limit, pool size, and tx gas limit |
| [`GetL1GasPriceEstimate`][GI5]`()`                   | Gets the current estimate of the L1 gas price              |
| [`GetCurrentTxL1GasFees`][GI6]`()`                   | Gets the fee paid to the aggregator for posting this tx    |

[GI0]: todo
[GI1]: todo
[GI2]: todo
[GI3]: todo
[GI4]: todo
[GI5]: todo
[GI6]: todo

# [ArbInfo][ArbInfo_link]<a name=ArbInfo></a>
Provides the ability to lookup basic info about accounts and contracts.

| Methods                              |                                       |
| :----------------------------------- | :------------------------------------ |
| [`GetBalance`][I0]`(account)` &nbsp; | Retrieves an account's balance        |
| [`GetCode`][I1]`(account)`           | Retrieves a contract's source program |

[I0]: todo
[I1]: todo

# [ArbosTest][ArbosTest_link]<a name=ArbosTest></a>
Provides a method of burning arbitrary amounts of gas, which exists for historical reasons. In Classic, `ArbosTest` had additional methods only the zero address could call. These have been removed since users don't use them and calls to missing methods revert.

| Methods                     |                                                     | Nitro changes |
| :-------------------------- | :-------------------------------------------------- |               |
| [`BurnArbGas`][T0]`(amount)` &nbsp; | unproductively burns the amount of L2 ArbGas &nbsp; | Now pure      |

[T0]: todo

# [ArbOwner][ArbOwner_link]<a name=ArbOwner></a>
Provides owners with tools for managing the rollup. Calls by non-owners will always revert.

| Methods                               |                                                                     |
| :------------------------------------ | :------------------------------------------------------------------ |
| [`AddChainOwner`][O0]`(account)`              | Promotes the user to chain owner                            |
| [`GetAllChainOwners`][O1]`()`                 | Retrieves the list of chain owners                          |
| [`IsChainOwner`][O2]`(account)`               | See if the user is a chain owner                            |
| [`RemoveChainOwner`][O3]`(account)`           | Demotes the user from chain owner                           |
| [`SetL1GasPriceEstimate`][O4]`(price)` &nbsp; | Sets the L1 gas price estimate directly, bypassing the autoregression |
| [`SetL2GasPrice`][O5]`(price)`                | Sets the L2 gas price directly, bypassing the pool calculus |

[O0]: todo
[O1]: todo
[O2]: todo
[O3]: todo
[O4]: todo
[O5]: todo

# [ArbOwnerPublic][ArbOwnerPublic_link]<a name=ArbOwnerPublic></a>
Provides non-owners with info about the current chain owners.

| Methods                        |                                    |
| :----------------------------- | :--------------------------------- |
| [`GetAllChainOwners`][OP0]`()`          | Retrieves the list of chain owners |
| [`IsChainOwner`][OP1]`(account)` &nbsp; | See if the user is a chain owner   |

[OP0]: todo
[OP1]: todo

# [ArbRetryableTx][ArbRetryableTx_link]<a name=ArbRetryableTx></a>
| Methods                         |                                    |
| :------------------------------ | :--------------------------------- |
| [`Cancel`][RT0]`(ticket)`                |  |
| [`GetBeneficiary`][RT1]`(ticket)` &nbsp; |  |
| [`GetLifetime`][RT2]`()`                 |  |
| [`GetTimeout`][RT3]`(ticket)`            |  |
| [`Keepalive`][RT4]`(ticket)`             |  |
| [`Redeem`][RT5]`(ticket)`                |  |

[RT0]: todo
[RT1]: todo
[RT2]: todo
[RT3]: todo
[RT4]: todo
[RT5]: todo

| Events                           |                                                                  |
| :------------------------------- | :--------------------------------------------------------------- |
| [`TicketCreated`][RTe0]           |     |
| [`LifetimeExtended`][RTe1] &nbsp; |     |
| [`RedeemScheduled`][RTe2]         |     |
| [`Redeemed`][RTe3]                |     |
| [`Canceled`][RTe4]                |     |

[RTe0]: todo
[RTe1]: todo
[RTe2]: todo
[RTe3]: todo
[RTe4]: todo

# [ArbStatistics][ArbStatistics_link]<a name=ArbStatistics></a>
Provides statistics about the rollup right before the Nitro upgrade. In Classic, this was how a user would get info such as the total number of accounts, but there's now better ways to do that with geth.

| Methods             |                                                                            |
| :------------------ | :------------------------------------------------------------------------- |
| [`GetStats`][ST0]`()` &nbsp; | Returns the current block number and some statistics about the rollup's pre-Nitro state |

[ST0]: todo

# [ArbSys][ArbSys_link]<a name=ArbSys></a>
Provides info about 

| Methods                            |                                                     | Nitro changes  |
| :----------------------------------| :-------------------------------------------------- | :------------- |
| [`ArbBlockNumber`][S0]`()`         | Gets the current L2 block number                    |                |
| [`ArbChainID`][S1]`()`             | Gets the rollup's unique chain identifier           |                |
| [`ArbOSVersion`][S2]`()`           | Gets the current ArbOS version                      | Now view       |
| [`GetStorageGasAvailable`][S3]`()` | Returns 0 since Nitro has no concept of storage gas | Now always 0   |
| [`IsTopLevelCall`][S4]`()`         | Checks if the call is top-level                                      |
| [`MapL1SenderContractAddressToL2Alias`][S5]`(contract, unused)` &nbsp; | Gets the contract's L2 alias | 2nd arg is unused |
| [`WasMyCallersAddressAliased`][S6]`()`           | Checks if the caller's caller was aliased               |
| [`MyCallersAddressWithoutAliasing`][S7]`()`      | Gets the caller's caller without any potential aliasing |
| [`SendTxToL1`][S8]`(destination, calldataForL1)` | Sends a transaction to L1, adding it to the outbox      |
| [`SendMerkleTreeState`][S9]`()`                  | Gets the root, size, and partials of the outbox Merkle tree state  |
| [`WithdrawEth`][S10]`(destination)`              | Send paid eth to the destination on L1                  |

[S0]: todo
[S1]: todo
[S2]: todo
[S3]: todo
[S4]: todo
[S5]: todo
[S6]: todo
[S7]: todo
[S8]: todo
[S9]: todo
[S10]: todo

| Events                            |                                                                  |
| :-------------------------------- | :--------------------------------------------------------------- |
| [`L2ToL1Transaction`][Se0] &nbsp; | Logs a send tx from L2 to L1, including data for outbox prooving |
| [`SendMerkleUpdate`][Se1]         | Logs a new merkle branch needed for constructing outbox proofs   |

[Se0]: todo
[Se1]: todo

| Removed                               |                                                                             |
| :------------------------------------ | :-------------------------------------------------------------------------- |
| [`GetStorageAt`][Sr0]`(account, index)` &nbsp; | Nitro doesn't need Classic's `eth_getStorageAt`, and users couldn't call it |
| [`GetTransactionCount`][Sr1]`(account)`        | Nitro doesn't need Classic's `eth_getStorageAt`, and users couldn't call it |

[Sr0]: todo
[Sr1]: todo
