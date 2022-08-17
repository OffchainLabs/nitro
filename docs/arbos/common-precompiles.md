# Overview
ArbOS provides L2-specific precompiles with methods smart contracts can call the same way they can solidity functions. This reference details those we expect users to most frequently use. For an exhaustive reference including those we don't expect most users to ever call, please refer to the [Full Precompiles documentation](precompiles.md).

From the perspective of user applications, precompiles live as contracts at the following addresses. Click on any to jump to their section.

| Precompile                                 | Address &nbsp; | Purpose                             |
| :----------------------------------------- | :------------- | :---------------------------------- |
| [`ArbAggregator`](#ArbAggregator)          | `0x6d`         | Configuring transaction aggregation |
| [`ArbGasInfo`](#ArbGasInfo)                | `0x6c`         | Info about gas pricing              |
| [`ArbRetryableTx`](#ArbRetryableTx) &nbsp; | `0x6e`         | Managing retryables                 |
| [`ArbSys`](#ArbSys)                        | `0x64`         | System-level functionality          |

[ArbAggregator_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbAddressTable.go
[ArbGasInfo_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbGasInfo.go
[ArbRetryableTx_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbRetryableTx.go
[ArbSys_link]: https://github.com/OffchainLabs/nitro/blob/master/precompiles/ArbSys.go

# [ArbAggregator][ArbAggregator_link]
Provides aggregators and their users methods for configuring how they participate in L1 aggregation. Arbitrum One's default aggregator is the Sequencer, which a user will prefer unless `SetPreferredAggregator` is invoked to change it.

| Methods                                                        |                                                         |
| :------------------------------------------------------------- | :------------------------------------------------------ |
| [![](e.png)][As0] [`GetPreferredAggregator`][A0]`(account)`    | Gets an account's preferred aggregator                  |
| [![](e.png)][As1] [`SetPreferredAggregator`][A1]`(aggregator)` | Sets the caller's preferred aggregator to that provided |
| [![](e.png)][As2] [`GetDefaultAggregator`][A2]`()`             | Gets the chain's default aggregator                     |

[A0]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbAggregator.go#L22
[A1]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbAggregator.go#L39
[A2]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbAggregator.go#L48

[As0]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbAggregator.sol#L28
[As1]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbAggregator.sol#L32
[As2]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbAggregator.sol#L35


# [ArbGasInfo][ArbGasInfo_link]
Provides insight into the cost of using the chain. These methods have been adjusted to account for Nitro's heavy use of calldata compression. Of note to end-users, we no longer make a distinction between non-zero and zero-valued calldata bytes.

| Methods                                                |                                                                   |
| :----------------------------------------------------- | :---------------------------------------------------------------- |
| [![](e.png)][GIs1] [`GetPricesInWei`][GI1]`()`         | Get prices in wei when using the caller's preferred aggregator    |
| [![](e.png)][GIs3] [`GetPricesInArbGas`][GI3]`()`      | Get prices in ArbGas when using the caller's preferred aggregator |
| [![](e.png)][GIs4] [`GetGasAccountingParams`][GI4]`()` | Get the chain speed limit, pool size, and tx gas limit            |
| [![](e.png)][GIs11] [`GetL1BaseFeeEstimate`][GI11]`()` | Get ArbOS's estimate of the L1 basefee in wei                     |

[GI1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L63
[GI3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L99
[GI4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L111
[GI11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/precompiles/ArbGasInfo.go#L150

[GIs1]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L58
[GIs3]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L83
[GIs4]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L94
[GIs11]: https://github.com/OffchainLabs/nitro/blob/3f504c57fba8ddf0759b7a55b4108e0bf5a078b3/solgen/src/precompiles/ArbGasInfo.sol#L122

# [ArbRetryableTx][ArbRetryableTx_link]
Provides methods for managing retryables. The model has been adjusted for Nitro, most notably in terms of how retry transactions are scheduled. For more information on retryables, please see [the retryable documentation](arbos.md#Retryables).


| Methods                                                     |                                                                                    | Nitro changes          |
| :---------------------------------------------------------- | :--------------------------------------------------------------------------------- | :--------------------- |
| [![](e.png)][RTs0] [`Cancel`][RT0]`(ticket)`                | Cancel the ticket and refund its callvalue to its beneficiary                      |                        |
| [![](e.png)][RTs1] [`GetBeneficiary`][RT1]`(ticket)` &nbsp; | Gets the beneficiary of the ticket                                                 |                        |
| [![](e.png)][RTs3] [`GetTimeout`][RT3]`(ticket)`            | Gets the timestamp for when ticket will expire                                     |                        |
| [![](e.png)][RTs4] [`Keepalive`][RT4]`(ticket)`             | Adds one lifetime period to the ticket's expiry                                    | Doesn't add callvalue  |
| [![](e.png)][RTs5] [`Redeem`][RT5]`(ticket)`                | Schedule an attempt to redeem the retryable, donating all of the call's gas &nbsp; | Happens in a future tx |

[RT0]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbRetryableTx.go#L184
[RT1]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbRetryableTx.go#L171
[RT3]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbRetryableTx.go#L115
[RT4]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbRetryableTx.go#L132
[RT5]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbRetryableTx.go#L36

[RTs0]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbRetryableTx.sol#L70
[RTs1]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbRetryableTx.sol#L63
[RTs3]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbRetryableTx.sol#L45
[RTs4]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbRetryableTx.sol#L55
[RTs5]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbRetryableTx.sol#L32


# [ArbSys][ArbSys_link]
Provides system-level functionality for interacting with L1 and understanding the call stack.

| Methods                                                            |                                                             |
| :----------------------------------------------------------------- | :---------------------------------------------------------- |
| [![](e.png)][Ss0] [`ArbBlockNumber`][S0]`()`                       | Gets the current L2 block number                            |
| [![](e.png)][Ss1] [`ArbBlockHash`][S1]`()`                         | Gets the L2 block hash, if the block is sufficiently recent |
| [![](e.png)][Ss5] [`IsTopLevelCall`][S5]`()`                       | Checks if the call is top-level                             |
| [![](e.png)][Ss9] [`SendTxToL1`][S9]`(destination, calldataForL1)` | Sends a transaction to L1, adding it to the outbox          |
| [![](e.png)][Ss11] [`WithdrawEth`][S11]`(destination)`             | Send paid eth to the destination on L1                      |

[S0]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbSys.go#L30
[S1]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbSys.go#L35
[S5]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbSys.go#L66
[S9]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbSys.go#L98
[S11]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/precompiles/ArbSys.go#L187

[Ss0]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbSys.sol#L31
[Ss1]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbSys.sol#L37
[Ss5]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbSys.sol#L61
[Ss9]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbSys.sol#L100
[Ss11]: https://github.com/OffchainLabs/nitro/blob/704e82bb38ae3ccd70c35e31934c7b45f6c25561/solgen/src/precompiles/ArbSys.sol#L92
