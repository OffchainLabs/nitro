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

| Methods                                                              |                                                         |
|:---------------------------------------------------------------------|:--------------------------------------------------------|
| [![](e | height=12)][As0] [`GetPreferredAggregator`][A0]`(account)`    | Gets an account's preferred aggregator                  |
| [![](e | height=12)][As1] [`SetPreferredAggregator`][A1]`(aggregator)` | Sets the caller's preferred aggregator to that provided |
| [![](e | height=12)][As2] [`GetDefaultAggregator`][A2]`()`             | Gets the rollup's default aggregator                    |

[A0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L19
[A1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L24
[A2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbAggregator.go#L29

[As0]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L7
[As1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L11
[As2]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbAggregator.sol#L14


# [ArbGasInfo][ArbGasInfo_link]<a name=ArbGasInfo></a>
Provides insight into the cost of using the rollup.

| Methods                                                      |                                                                    |
|:-------------------------------------------------------------|:-------------------------------------------------------------------|
| [![](e | height=12)][GIs1] [`GetPricesInWei`][GI1]`()`         | Gets prices in wei when using the caller's preferred aggregator    |
| [![](e | height=12)][GIs3] [`GetPricesInArbGas`][GI3]`()`      | Gets prices in ArbGas when using the caller's preferred aggregator |
| [![](e | height=12)][GIs4] [`GetGasAccountingParams`][GI4]`()` | Gets the rollup's speed limit, pool size, and tx gas limit         |
| [![](e | height=12)][GIs5] [`GetL1GasPriceEstimate`][GI5]`()`  | Gets the current estimate of the L1 gas price                      |

[GI1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L62
[GI3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L95
[GI4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L104
[GI5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/precompiles/ArbGasInfo.go#L112

[GIs1]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L17
[GIs3]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L25
[GIs4]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L28
[GIs5]: https://github.com/OffchainLabs/nitro/blob/7e4c1a5119d83e144f5398597d046074c1741717/solgen/src/precompiles/ArbGasInfo.sol#L31


# [ArbSys][ArbSys_link]<a name=ArbSys></a>
Provides info about 

| Methods                                                                  |                                                    |
|:-------------------------------------------------------------------------|:---------------------------------------------------|
| [![](e =7x12)][Ss0] [`ArbBlockNumber`][S0]`()`                       | Gets the current L2 block number                   |
| [![](e =x12)][Ss4] [`IsTopLevelCall`][S4]`()`                       | Checks if the call is top-level                    |
| [![](e =7x12)[Ss8]] [`SendTxToL1`][S8]`(destination, calldataForL1)` | Sends a transaction to L1, adding it to the outbox |
| ![](e =x12)[Ss10] [`WithdrawEth`][S10]`(destination)`             | Send paid eth to the destination on L1             |

[S0]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L27
[S4]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L48
[S8]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L80
[S10]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/precompiles/ArbSys.go#L153

[Ss0]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L27
[Ss4]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L51
[Ss8]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L87
[Ss10]: https://github.com/OffchainLabs/nitro/blob/a9f2030de70460f65377174895836d3e4e33519e/solgen/src/precompiles/ArbSys.sol#L79
