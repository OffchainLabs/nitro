# Overview
ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This reference details those we expect users to most frequently use. For an exhaustive reference including those we don't expect most users to ever call, please refer to the [Full Precompiles documentation](Precompiles.md).

From the perspective of user applications, precompiles live as contracts at the following addresses. Click on any to jump to their section.

| Precompile                                 | Address &nbsp; | Purpose                             |
|:-------------------------------------------|:---------------|:------------------------------------|
| [`ArbAggregator`](#ArbAggregator)          | `0x6d`         | Configuring transaction aggregation |
| [`ArbGasInfo`](#ArbGasInfo)                | `0x6c`         | Info about gas pricing              |
| [`ArbRetryableTx`](#ArbRetryableTx) &nbsp; | `0x6e`         | Managing retryables                 |
| [`ArbSys`](#ArbSys)                        | `0x64`         | System-level functionality          |

[ArbAggregator_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbAddressTable.go
[ArbGasInfo_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbGasInfo.go
[ArbRetryableTx_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbRetryableTx.go
[ArbSys_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbSys.go

# [ArbAggregator][ArbAggregator_link]<a name=ArbAggregator></a>
Provides aggregators and their users methods for configuring how they participate in L1 aggregation. Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless `SetPreferredAggregator` is invoked to change it.

| Methods                                                                              |                                                         |
|:-------------------------------------------------------------------------------------|:--------------------------------------------------------|
| [<img src=e.png height=16>][As0] [`GetPreferredAggregator`][A0]`(account)`           | Gets an account's preferred aggregator                  |
| [<img src=e.png height=16>][As1] [`SetPreferredAggregator`][A1]`(aggregator)` &nbsp; | Sets the caller's preferred aggregator to that provided |
| [<img src=e.png height=16>][As2] [`GetDefaultAggregator`][A2]`()`                    | Gets the chain's default aggregator                     |

[A0]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbAggregator.go#L21
[A1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbAggregator.go#L26
[A2]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbAggregator.go#L31

[As0]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbAggregator.sol#L7
[As1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbAggregator.sol#L11
[As2]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbAggregator.sol#L14


# [ArbGasInfo][ArbGasInfo_link]<a name=ArbGasInfo></a>
Provides insight into the cost of using the chain. These methods have been adjusted to account for Nitro's heavy use of calldata compression. Of note to end-users, we no longer make a distinction between non-zero and zero-valued calldata bytes.

| Methods                                                               |                                                                    |
|:----------------------------------------------------------------------|:-------------------------------------------------------------------|
| [<img src=e.png height=16>][GIs1] [`GetPricesInWei`][GI1]`()`         | Gets prices in wei when using the caller's preferred aggregator    |
| [<img src=e.png height=16>][GIs3] [`GetPricesInArbGas`][GI3]`()`      | Gets prices in ArbGas when using the caller's preferred aggregator |
| [<img src=e.png height=16>][GIs4] [`GetGasAccountingParams`][GI4]`()` | Gets the rollup's speed limit, pool size, and tx gas limit         |
| [<img src=e.png height=16>][GIs5] [`GetL1GasPriceEstimate`][GI5]`()`  | Gets the current estimate of the L1 gas price                      |
[GI1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbGasInfo.go#L61
[GI3]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbGasInfo.go#L94
[GI4]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbGasInfo.go#L103
[GI5]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbGasInfo.go#L112

[GIs1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbGasInfo.sol#L17
[GIs3]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbGasInfo.sol#L25
[GIs4]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbGasInfo.sol#L28
[GIs5]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbGasInfo.sol#L31

# [ArbRetryableTx][ArbRetryableTx_link]<a name=ArbRetryableTx></a>
Provides methods for managing retryables. The model has been adjusted for Nitro, most notably in terms of how retry transactions are scheduled. For more information on retryables, please see [the retryable documentation](ArbOS.md#Retryables).


| Methods                                                                    |                                                                                    | Nitro changes         |
|:---------------------------------------------------------------------------|:-----------------------------------------------------------------------------------|:----------------------|
| [<img src=e.png height=16>][RTs0] [`Cancel`][RT0]`(ticket)`                | Cancel the ticket and refund its callvalue to its beneficiary                      |                       |
| [<img src=e.png height=16>][RTs1] [`GetBeneficiary`][RT1]`(ticket)` &nbsp; | Gets the beneficiary of the ticket                                                 |                       |
| [<img src=e.png height=16>][RTs3] [`GetTimeout`][RT3]`(ticket)`            | Gets the timestamp for when ticket will expire                                     |                       |
| [<img src=e.png height=16>][RTs4] [`Keepalive`][RT4]`(ticket)`             | Adds one lifetime period to the ticket's expiry                                    | Doesn't add callvalue |
| [<img src=e.png height=16>][RTs5] [`Redeem`][RT5]`(ticket)`                | Schedule an attempt to redeem the retryable, donating all of the call's gas &nbsp; | Happens next tx       |

[RT0]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbRetryableTx.go#L178
[RT1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbRetryableTx.go#L165
[RT3]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbRetryableTx.go#L109
[RT4]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbRetryableTx.go#L126
[RT5]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbRetryableTx.go#L33

[RTs0]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbRetryableTx.sol#L53
[RTs1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbRetryableTx.sol#L46
[RTs3]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbRetryableTx.sol#L28
[RTs4]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbRetryableTx.sol#L38
[RTs5]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbRetryableTx.sol#L15

# [ArbSys][ArbSys_link]<a name=ArbSys></a>
Provides system-level functionality for interacting with L1 and understanding the call stack.

| Methods                                                                           |                                                             |
|:----------------------------------------------------------------------------------|:------------------------------------------------------------|
| [<img src=e.png height=16>][Ss0] [`ArbBlockNumber`][S0]`()`                       | Gets the current L2 block number                            |
| [<img src=e.png height=16>][Ss1] [`ArbBlockHash`][S1]`()`                         | Gets the L2 block hash, if the block is sufficiently recent |
| [<img src=e.png height=16>][Ss5] [`IsTopLevelCall`][S5]`()`                       | Checks if the call is top-level                             |
| [<img src=e.png height=16>][Ss9] [`SendTxToL1`][S9]`(destination, calldataForL1)` | Sends a transaction to L1, adding it to the outbox          |
| [<img src=e.png height=16>][Ss11] [`WithdrawEth`][S11]`(destination)`             | Send paid eth to the destination on L1                      |

[S0]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbSys.go#L30
[S1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbSys.go#L35
[S5]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbSys.go#L66
[S9]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbSys.go#L98
[S11]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/precompiles/ArbSys.go#L172

[Ss0]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbSys.sol#L27
[Ss1]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbSys.sol#L33
[Ss5]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbSys.sol#L57
[Ss9]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbSys.sol#L93
[Ss11]: https://github.com/OffchainLabs/nitro/blob/f11ba39cf91ee1fe1b5f6b67e8386e5efd147667/solgen/src/precompiles/ArbSys.sol#L85
