# Overview
ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This reference details those we expect users to most frequently use. For an exhaustive reference including those we don't expect most users to ever call, please refer to the [Full Precompiles documentation](Precompiles.md).

From the perspective of user applications, precompiles live as contracts at the following addresses. Click on any to jump to their section.

| Precompile                                     | Address &nbsp; | Purpose                            |
| :--------------------------------------------- | :------------- | :--------------------------------- |
| [`ArbAggregator`](#ArbAggregator)              | `0x6d`         | Configuring aggregation            |
| [`ArbGasInfo`](#ArbGasInfo)                    | `0x6c`         | Info about gas pricing             |
| [`ArbRetryableTx`](#ArbRetryableTx)            | `0x6e`         | Managing retryables                |
| [`ArbSys`](#ArbSys)                            | `0x64`         | System-level functionality         |

[ArbAggregator_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbAddressTable.go
[ArbGasInfo_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbGasInfo.go
[ArbRetryableTx_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbRetryableTx.go
[ArbSys_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbSys.go

# [ArbAggregator][ArbAggregator_link]<a name=ArbAggregator></a>
Provides aggregator's and their users methods for configuring how they participate in L1 aggregation. Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless `SetPreferredAggregator` is invoked to change it.

| Methods                                      |                                                                      |
| :------------------------------------------- | :------------------------------------------------------------------- |
| [`GetDefaultAggregator`][A2]`()`             | Gets the rollup's default aggregator                                 |
| [`GetPreferredAggregator`][A4]`(account)`    | Gets an account's preferred aggregator                               |
| [`SetPreferredAggregator`][A5]`(aggregator)` | Sets the caller's preferred aggregator to that provided              |

[A0]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAggregator.go#L15
[A1]: https://github.com/OffchainLabs/nitro/blob/0cc34f548483d59f445d3744c9d912b58a87e563/precompiles/ArbAggregator.go#L19
[A2]: todo
[A3]: todo
[A4]: todo
[A5]: todo
[A6]: todo
[A7]: todo

# [ArbGasInfo][ArbGasInfo_link]<a name=ArbGasInfo></a>
Provides insight into the cost of using the rollup.

| Methods                             |                                                                    |
| :-----------------------------------| :----------------------------------------------------------------- |
| [`GetPricesInWei`][GI1]`()`         | Gets prices in wei when using the caller's preferred aggregator    |
| [`GetPricesInArbGas`][GI3]`()`      | Gets prices in ArbGas when using the caller's preferred aggregator |
| [`GetGasAccountingParams`][GI4]`()` | Gets the rollup's speed limit, pool size, and tx gas limit         |
| [`GetL1GasPriceEstimate`][GI5]`()`  | Gets the current estimate of the L1 gas price                      |

[GI0]: todo
[GI1]: todo
[GI2]: todo
[GI3]: todo
[GI4]: todo
[GI5]: todo
[GI6]: todo

# [ArbSys][ArbSys_link]<a name=ArbSys></a>
Provides info about 

| Methods                                          |                                                    |
| :----------------------------------------------- | :------------------------------------------------- |
| [`ArbBlockNumber`][S0]`()`                       | Gets the current L2 block number                   |
| [`IsTopLevelCall`][S4]`()`                       | Checks if the call is top-level                    |
| [`SendTxToL1`][S8]`(destination, calldataForL1)` | Sends a transaction to L1, adding it to the outbox |
| [`WithdrawEth`][S10]`(destination)`              | Send paid eth to the destination on L1             |

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
