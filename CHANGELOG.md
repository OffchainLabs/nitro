# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [v3.10.0-rc.1](https://github.com//nitro/compare/v3.9.5...v3.10.0-rc.1) - 2026-01-22

### Configuration

- `cmd/daserver` -> `cmd/anytrustserver`
- `cmd/datool` -> `cmd/anytrusttool`
- `--node.data-availability.*` -> `--node.da.anytrust.*`
- `--node.batch-poster.das-retention-period` -> `--node.batch-poster.anytrust-retention-period`
- `--node.data-availability.rpc-aggregator.das-rpc-client.*` -> `--node.data-availability.rpc-aggregator.rpc-client.*`
- `--node.batch-poster.max-size` -> `--node.batch-poster.max-calldata-batch-size`
- `--node.da-provider.*` -> `--node.da.external-provider.*`
- `anytrusttool` `--das-retention-period` -> `--anytrust-retention-period`
- `anytrusttool` `--das-rpc-client.*` -> `--rpc-client.*`
- `daprovider` `--anytrust.parent-chain.node-url` -> `--parent-chain-node-url`
- `daprovider` `--anytrust.parent-chain.connection-attempts` -> `--parent-chain-connection-attempts`
- `daprovider` `--anytrust.parent-chain.sequencer-inbox-address` -> `--parent-chain-sequencer-inbox-address`
- `daserver` `--anytrust.parent-chain.node-url` -> `--parent-chain-node-url`
- `daserver` `--anytrust.parent-chain.connection-attempts` -> `--parent-chain-connection-attempts`
- `daserver` `--anytrust.parent-chain.sequencer-inbox-address` -> `--parent-chain-sequencer-inbox-address`

### Added

- Enable Execution and Consensus to connect to the other via json-rpc [[PR]](https://github.com/OffchainLabs/nitro/pull/3617)
- Merge go-ethereum v1.16.7: [[PR]](https://github.com/OffchainLabs/nitro/pull/3965)
- Add log for genesis assertion validation: [[PR]](https://github.com/OffchainLabs/nitro/pull/4042)
- Precompiles for multi dimensional multi constraint pricer: [[PR]](https://github.com/OffchainLabs/nitro/pull/3995)
- Add return error in case of missing code for SetProgramCached: [[PR]](https://github.com/OffchainLabs/nitro/pull/4077)
- Guard zero batch count in inbox search and avoid validator underflow: [[PR]](https://github.com/OffchainLabs/nitro/pull/4028)
- Custom DA Complete Fraud Proof Support: [[PR]](https://github.com/OffchainLabs/nitro/pull/3237)
- Make uncompressed batch size limit configurable: [[PR]](https://github.com/OffchainLabs/nitro/pull/3947)
- Add new option to allow BlocksReExecutor to commit state to disk: [[PR]](https://github.com/OffchainLabs/nitro/pull/4132)
- Implement Execution/Consensus interface over RPC: [[PR]](https://github.com/OffchainLabs/nitro/pull/3617)
- Add comment about blob decoding failure: [[PR]](https://github.com/OffchainLabs/nitro/pull/4182)
- Add metric when validator stops validating because of low memory: [[PR]](https://github.com/OffchainLabs/nitro/pull/4196)
- Add address-based transaction filtering for sequencer. [[PR]](https://github.com//nitro/pull/4157)
- Add support for Geth state size tracking with a flag `--execution.caching.state-size-tracking`. [[PR]](https://github.com//nitro/pull/4210)
- Added a note to the `--node.feed.output.signed` flag that this will use batch poster's wallet for signing. [[PR]](https://github.com//nitro/pull/4211)
- Add GetMultiGasBaseFee precompile to retrieve fees per resource kind. [[PR]](https://github.com//nitro/pull/4188)
- Add `execution.caching.trie-cap-batch-size` option that sets batch size in bytes used in the TrieDB Cap operation (0 = use geth default). [[PR]](https://github.com//nitro/pull/3221)
- Add `execution.caching.trie-commit-batch-size` option that sets batch size in bytes used in the TrieDB Commit operation (0 = use geth default). [[PR]](https://github.com//nitro/pull/3221)
- Add database batch size checks to prevent panic on pebble batch overflow. [[PR]](https://github.com//nitro/pull/3221)
- new wasm import programPrepare. [[PR]](https://github.com//nitro/pull/4013)
- new wasm import programRequiresPrepare. [[PR]](https://github.com//nitro/pull/4013)
- Enable running JIT validation with native input mode. [[PR]](https://github.com//nitro/pull/4228)
- Enabled consensus node to communicate with ExecutionRecorder over RPC. [[PR]](https://github.com//nitro/pull/4233)
- Implement recording of txs for MEL validation. [[PR]](https://github.com//nitro/pull/4198)
- Added a new endpoint to arb namespace called arb_getMinRequiredNitroVersion that returns minimum required version of the nitro node software. [[PR]](https://github.com//nitro/pull/3808)
- Add new precompile ArbFilteredTransactionsManager to manage filtered transactions. [[PR]](https://github.com//nitro/pull/4174)
- Add transaction filterers to ArbOwner to limit access to ArbFilteredTransactionsManager. [[PR]](https://github.com//nitro/pull/4174)
- Limit ArbOwners' ability to create transaction filterers with TransactionFilteringFromTime. [[PR]](https://github.com//nitro/pull/4174)

### Changed

- Arbos storage for multi dimensional constraints: [[PR]](https://github.com/OffchainLabs/nitro/pull/3954)
- Arbitrator workspace enhancements: [[PR]](https://github.com/OffchainLabs/nitro/pull/4010)
- Only sign important fields in feed: [[PR]](https://github.com/OffchainLabs/nitro/pull/3996)
- Post report-only batch after MaxEmptyBatchDelay: [[PR]](https://github.com/OffchainLabs/nitro/pull/3948)
- Remove bold/util/StopWaiter use util/StopWaiter instead: [[PR]](https://github.com/OffchainLabs/nitro/pull/4044)
- Enhance state management in StopWaiterSafe: [[PR]](https://github.com/OffchainLabs/nitro/pull/4039)
- Log critical error when fails to flush batch in setHead: [[PR]](https://github.com/OffchainLabs/nitro/pull/4052)
- Broadcaster refactor: [[PR]](https://github.com/OffchainLabs/nitro/pull/3982)
- Use stateless keccak where possible: [[PR]](https://github.com/OffchainLabs/nitro/pull/4025)
- Optimize ConcatByteSlices to avoid repeated reallocations: [[PR]](https://github.com/OffchainLabs/nitro/pull/4055)
- Centralize validator worker throttling in BlockValidator #NIT-3339: [[PR]](https://github.com/OffchainLabs/nitro/pull/4032)
- Do not require BatchMetadata for reading DelayedInbox: [[PR]](https://github.com/OffchainLabs/nitro/pull/4106)
- redis pubsub: add retries limit and option to disable retries: [[PR]](https://github.com/OffchainLabs/nitro/pull/2803)
- Extract saturating arithmetics: [[PR]](https://github.com/OffchainLabs/nitro/pull/4126)
- Add MaxTxSize check to ValidateExpressLaneTx(): [[PR]](https://github.com/OffchainLabs/nitro/pull/4105)
- Adjust pricing formula with weight normalisation by max weight: [[PR]](https://github.com/OffchainLabs/nitro/pull/4125)
- Make bids receiver buffer size configurable: [[PR]](https://github.com/OffchainLabs/nitro/pull/4117)
- Improve utility for ensuring batch posting and processing: [[PR]](https://github.com/OffchainLabs/nitro/pull/4144)
- Issue refunds based on multi-dimensinal base fee: [[PR]](https://github.com/OffchainLabs/nitro/pull/4082)
- Rename bold packages: [[PR]](https://github.com/OffchainLabs/nitro/pull/4146)
- Add isActiveSequencer to ExpressLaneTracker log messages: [[PR]](https://github.com/OffchainLabs/nitro/pull/4131)
- Remove `ExecutionRecorder.MarkValid` and rely on `ExecutionClient.SetFinalityData`: [[PR]](https://github.com/OffchainLabs/nitro/pull/4154)
- Unify keccaking: [[PR]](https://github.com/OffchainLabs/nitro/pull/4128)
- Rename database variable names: [[PR]](https://github.com/OffchainLabs/nitro/pull/4155)
- [config] Multiple DA provider infrastructure: [[PR]](https://github.com/OffchainLabs/nitro/pull/3949)
- [config] Rename DAS to AnyTrust: [[PR]](https://github.com/OffchainLabs/nitro/pull/4142)
- For wasm compilation target, keccak256 hashing is now expected to be provided by an external module. For JIT and arbitrator provers we inject Rust-based implementation. [[PR]](https://github.com//nitro/pull/4001)
- Renamed execution subpackages and RPC structs, removing redundancy in names and making them more idiomatic. [[PR]](https://github.com//nitro/pull/4206)
- Reorganize and refactor JIT validator CLI configuration. [[PR]](https://github.com//nitro/pull/4203)
- Update `contracts-legacy` submodule pin to `v2-main` branch. [[PR]](https://github.com//nitro/pull/4215)
- Rename database variable names [[PR]](https://github.com/OffchainLabs/nitro/pull/4176)
- Move whole Rust codebase from `arbitrator/` to `crates/` directory. Move workspace files to the root. [[PR]](https://github.com//nitro/pull/4184)
- Align the `execution.ExecutionRecorder` interface API with the other execution interfaces. Makes it more suitable for RPC calling. [[PR]](https://github.com//nitro/pull/4186)
- Fix fifo lock flakey tests and implementation for bold. [[PR]](https://github.com//nitro/pull/4238)
- Merged ethereum/go-ethereum v1.16.8. [[PR]](https://github.com//nitro/pull/4254)
- Treat `finality msg count` as an intermittent issue. Only if it doesn't resolve itself within a short period of time, it will be logged as error. [[PR]](https://github.com//nitro/pull/4256)
- Changed state-history default to zero for path archive. [[PR]](https://github.com//nitro/pull/4197)
- Make the `--cranelift` flag turned on by default for JIT validator. [[PR]](https://github.com//nitro/pull/4228)
- Remove manual gas math from ArbRetryableTx.Redeem by using static L2 pricing backlog update cost. [[PR]](https://github.com//nitro/pull/4101)

### Removed

- remove gas dimension tracers, system tests, and mock contracts. Multigas is now in the stf. [[PR]](https://github.com//nitro/pull/4220)
- Remove bold/util/backend.go: [[PR]](https://github.com/OffchainLabs/nitro/pull/4134)

### Fixed

- Clarify Redis DAS signing key config: [[PR]](https://github.com/OffchainLabs/nitro/pull/3990)
- Enhance data poster internal state handling: [[PR]](https://github.com/OffchainLabs/nitro/pull/3981)
- Align ArbOwner precompile docs: [[PR]](https://github.com/OffchainLabs/nitro/pull/4000)
- Update GenesisBlockNum to first nitro block in chain info: [[PR]](https://github.com/OffchainLabs/nitro/pull/4041)
- Do not set init.empty to true when init.genesis-json-file is set: [[PR]](https://github.com/OffchainLabs/nitro/pull/4051)
- Show correct value in redis database number error: [[PR]](https://github.com/OffchainLabs/nitro/pull/4067)
- Fix timer leak in redislock refresh goroutine: [[PR]](https://github.com/OffchainLabs/nitro/pull/4060)
- Use RWLock in StopWaiter: [[PR]](https://github.com/OffchainLabs/nitro/pull/4033)
- Fix RecentWasms cache bug by using pointers in methods: [[PR]](https://github.com/OffchainLabs/nitro/pull/4035)
- Fix maintenance ticker leak: [[PR]](https://github.com/OffchainLabs/nitro/pull/4102)
- Prevent unintended mutation of latestHeader.Number [[PR]](https://github.com/OffchainLabs/nitro/pull/4116)
- Fix invalid DA cert branch in inbox: [[PR]](https://github.com/OffchainLabs/nitro/pull/4149)
- Solve reexecutor panic if header unavailable: [[PR]](https://github.com/OffchainLabs/nitro/pull/4178)
- Prevent unintended mutation of latestHeader.Number in ParentChainIsUsingEIP7623: [[PR]](https://github.com/OffchainLabs/nitro/pull/4116)
- Disable chunked-store in datool dumpkeyset [[PR]](https://github.com/OffchainLabs/nitro/pull/4176)
- Add sequencer message length check in for the `daprovider.Reader` implementations. [[PR]](https://github.com//nitro/pull/4214)
- Fix nil pointer panic in `auctioneer_submitBid` RPC method when receiving malformed bid data. [[PR]](https://github.com//nitro/pull/4232)
- Fixed ValidateCertificate proof generation to panic on preimageType overflow (> 255) instead of silently using a fallback value, aligning with the Solidity one-step prover which reverts for this case. [[PR]](https://github.com//nitro/pull/4187)
- Update implementation of receipts and txs fetching in mel-replay. [[PR]](https://github.com//nitro/pull/4199)
- Added testing for recording and fetching of logs and txs needed for MEL validation. [[PR]](https://github.com//nitro/pull/4199)

### Internal

- [MEL] - Implement delayed message accumulation in native mode: [[PR]](https://github.com/OffchainLabs/nitro/pull/3389)
- [MEL] - Update melextraction package to use logs instead of receipts and implement logs and headers fetcher: [[PR]](https://github.com/OffchainLabs/nitro/pull/4063)
- [MEL] - Implement preimage recorder for `DelayedMessageDatabase` interface: [[PR]](https://github.com/OffchainLabs/nitro/pull/4119)
- [MEL] - Implement recording of preimages related to sequencer batches (DA providers): [[PR]](https://github.com/OffchainLabs/nitro/pull/4133)
- Add new boolean option to `BlocksReExecutor` called `CommitStateToDisk` that will allow `BlocksReExecutor.Blocks` range to not only re-executes blocks but it will also commit their state to triedb on disk. [[PR]](https://github.com/OffchainLabs/nitro/pull/4132)

