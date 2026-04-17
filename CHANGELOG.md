# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [v3.10.0-rc.7](https://github.com/OffchainLabs/nitro/compare/v3.10.0-rc.6...v3.10.0-rc.7) - 2026-04-10

### Configuration

- Add `--execution.disable-arbowner-ethcall` flag to disable ArbOwner precompile calls outside on-chain execution. [[PR]](https://github.com/OffchainLabs/nitro/pull/4591)
- Add `--stylus-target.native-stack-size` config to set the initial Wasmer coroutine stack size for Stylus execution. [[PR]](https://github.com/OffchainLabs/nitro/pull/4538)

### Added

 - `ValidationInputsAt` debug API now includes an `ExpectedEndState` field in the returned JSON, allowing the arbitrator and JIT provers to verify their computed end state when run from the command line with `--json-inputs`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4563)
- Nitro metrics for MEL and L3 system test. [[PR]](https://github.com/OffchainLabs/nitro/pull/4424)
- Added support for id-set-filter for address filter reporting. [[PR]](https://github.com/OffchainLabs/nitro/pull/4594)
- Linter for checking defer usage inside for loops. [[PR]](https://github.com/OffchainLabs/nitro/pull/4616)

### Changed

 - Update the L2 msgs accumulation from merkle tree to a hash-chain based accumulation and implement extracting of message from the accumulator using preimages. [[PR]](https://github.com/OffchainLabs/nitro/pull/4518)
- Poll parent chain's `eth_config` RPC (EIP-7910) to dynamically fetch blob schedule configuration. [[PR]](https://github.com/OffchainLabs/nitro/pull/4511)
 - Update the delayed msgs accumulation from merkle tree to a hash-chain based accumulation (Inbox-Outbox) and implement recording of their preimages and subsequently reading them using preimages. [[PR]](https://github.com/OffchainLabs/nitro/pull/4520)
- Change hashing algorithm for address filtering feature to match the provider specs. [[PR]](https://github.com/OffchainLabs/nitro/pull/4586)
- Log submitted express lane transactions like eth_sendRawTransaction. [[PR]](https://github.com/OffchainLabs/nitro/pull/4588)
- Preallocate slice capacity across codebase to reduce memory allocations. [[PR]](https://github.com/OffchainLabs/nitro/pull/4605)
- Update Stylus SDK to v0.10.3. [[PR]](https://github.com/OffchainLabs/nitro/pull/4621)
 - Replace InboxReader and InboxTracker implementation with Message extractor code. [[PR]](https://github.com/OffchainLabs/nitro/pull/4593)
- Force net.Dialer to use "tcp4" instead of falling back to "tcp6". [[PR]](https://github.com/OffchainLabs/nitro/pull/4611)
- Update Go to 1.25.9 in Dockerfile. [[PR]](https://github.com/OffchainLabs/nitro/pull/4625)

### Fixed

- Execution RPC client correctly handles ResultNotFound error. [[PR]](https://github.com/OffchainLabs/nitro/pull/4580)
- Fixed flakiness in `TestRetryableFilteringStylusSandwichRollback`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4599)
- Add some micro-optimization to hashing of address filter implementation. [[PR]](https://github.com/OffchainLabs/nitro/pull/4600)
- Fix deadlock in `StopWaiterSafe.stopAndWaitImpl` by releasing `RLock` before blocking on `waitChan`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4604)
- Re-enable erc20 test. [[PR]](https://github.com/OffchainLabs/nitro/pull/4621)
- Fix Wasmer stack pool reusing stale smaller stacks after a stack size change. [[PR]](https://github.com/OffchainLabs/nitro/pull/4538)
- Automatically detect native stack overflow during Stylus execution and recover. [[PR]](https://github.com/OffchainLabs/nitro/pull/4538)

### Internal

- Introduce rustfmt.toml. [[PR]](https://github.com/OffchainLabs/nitro/pull/4579)
 - Extract C-FFI related code from prover crate to prover-ffi. [[PR]](https://github.com/OffchainLabs/nitro/pull/4570)
- Added additional tests for stylus contracts redeems. [[PR]](https://github.com/OffchainLabs/nitro/pull/4564)
- Tx pre-checker uses gas estimation dry-run to detect filtered addresses before forwarding. [[PR]](https://github.com/OffchainLabs/nitro/pull/4583)
- Replace TransactionFiltererAPI mutex with channel-based sequential processing and simplify Filter to not return a transaction hash. [[PR]](https://github.com/OffchainLabs/nitro/pull/4428)

## [v3.10.0-rc.6](https://github.com/OffchainLabs/nitro/compare/v3.10.0-rc.5...v3.10.0-rc.6) - 2026-03-27

### Configuration

- Add `--execution.stylus-target.allow-fallback` flag: if true, fall back to an alternative compiler when compilation of a Stylus program fails (default: true). [[PR]](https://github.com/OffchainLabs/nitro/pull/4534)

### Added

- Add `/liveness` and `/readiness` HTTP health check endpoints to the transaction-filterer service. Readiness reports 503 until the sequencer client is connected. [[PR]](https://github.com/OffchainLabs/nitro/pull/4495)
- Prevent MEL node startup if have non-MEL entries in consensus database. [[PR]](https://github.com/OffchainLabs/nitro/pull/4449)
- Add tip collection ArbOS state field and precompile to allow the chain owner to enable or disable collecting transaction tips. [[PR]](https://github.com/OffchainLabs/nitro/pull/4515)
- ArbOS 60: `ArbOwner.setWasmActivationGas` / `ArbWasm.activationGas` — chain owners can set a constant gas charge burned before each Stylus contract activation (default 0). [[PR]](https://github.com/OffchainLabs/nitro/pull/4556)

### Changed

- Disable cranelift fallback for non-onchain execution modes. [[PR]](https://github.com/OffchainLabs/nitro/pull/4541)
- Add recovery to all stopWaiter threads. [[PR]](https://github.com/OffchainLabs/nitro/pull/4547)
- Stylus: reject activation of wasm programs using the multi-value extension (functions with multiple return values, or block/loop/if with parameters) starting from ArbOS version 60. [[PR]](https://github.com/OffchainLabs/nitro/pull/4554)
- Upgrade to wasmer v7.1.0. [[PR]](https://github.com/OffchainLabs/nitro/pull/4569)
- Added consensus v60-rc.1 to Dockerfile. [[PR]](https://github.com/OffchainLabs/nitro/pull/4574)

### Fixed

- Fix nil-dereference and log format in `cmd/nitro/nitro.go` when machine locator creation fails; return early instead of falling through to dereference nil locator. [[PR]](https://github.com/OffchainLabs/nitro/pull/4530)
- Part 3 of integrating MEL into master. [[PR]](https://github.com/OffchainLabs/nitro/pull/4421)
- Re-enable download of previous consensus machine versions (v50, v51, v51.1, v60-alpha.1) in Docker build. [[PR]](https://github.com/OffchainLabs/nitro/pull/4549)
- Harden blocks reexecutor with panic recovery for concurrent trie access races. [[PR]](https://github.com/OffchainLabs/nitro/pull/4531)
- Do not access state in CollectTips for ArbOS < 60. [[PR]](https://github.com/OffchainLabs/nitro/pull/4558)

### Internal

- Introduce `ValidationInput` intermediate data structure with optional rkyv serialization in the validation crate. [[PR]](https://github.com/OffchainLabs/nitro/pull/4521)
- Minor refactor in JIT, prover and validator crates. [[PR]](https://github.com/OffchainLabs/nitro/pull/4521)
- Moved float-related utilities from arbmath to a new package floatmath. [[PR]](https://github.com/OffchainLabs/nitro/pull/4561)
- Add TrackChild/StartAndTrackChild to StopWaiter for automatic LIFO child lifecycle management. [[PR]](https://github.com/OffchainLabs/nitro/pull/4536)

## [v3.10.0-rc.5](https://github.com/OffchainLabs/nitro/compare/v3.10.0-rc.4...v3.10.0-rc.5) - 2026-03-27

### Added

- Group rollback for cascading redeem filtering using deferred statedb clone. [[PR]](https://github.com/OffchainLabs/nitro/pull/4436)

### Changed

- Update Go to 1.25.8 in Dockerfile. [[PR]](https://github.com/OffchainLabs/nitro/pull/4496)
- Preflight the worst-case fragment read gas during multi-fragment Stylus activation, then charge the actual EXTCODECOPY-style cost after the fragment code is read. [[PR]](https://github.com/OffchainLabs/nitro/pull/4489)
- Enable http communication between block validator and validation server. [[PR]](https://github.com/OffchainLabs/nitro/pull/4499)
- Split `ResourceKindStorageAccess` into `ResourceKindStorageAccessRead` and `ResourceKindStorageAccessWrite` for finer-grained multi-dimensional gas metering. [[PR]](https://github.com/OffchainLabs/nitro/pull/4504)
- Upgrade to wasmer7. [[PR]](https://github.com/OffchainLabs/nitro/pull/4332)

### Fixed

- Part 2 of integrating MEL into master. [[PR]](https://github.com/OffchainLabs/nitro/pull/4402)
- Once timeout attempts are exhausted, treat it as any other, possibly fatal failure. [[PR]](https://github.com/OffchainLabs/nitro/pull/4478)
- Use proper comparison for attempts. [[PR]](https://github.com/OffchainLabs/nitro/pull/4478)
- Fix bold StopWaiter usage: start child structs on their own StopWaiters instead of the parent's. Fix StopAndWait ordering. [[PR]](https://github.com/OffchainLabs/nitro/pull/4487)
- Fix multi-gas refunds in retryables (ArbOS60). [[PR]](https://github.com/OffchainLabs/nitro/pull/4498)
- Fix StopWaiter lifecycle ordering: stop children before parent in StopAndWait, and pass managed context to children in Start. [[PR]](https://github.com/OffchainLabs/nitro/pull/4510)
- Fix MEL feature flag bugs: nil message dereference in delayed sequencer, incorrect waitingForFinalizedBlock domain, and missing BatchPoster+MEL config validation. [[PR]](https://github.com/OffchainLabs/nitro/pull/4485)
- Fix ValidationSpawnerRetryWrapper lifecycle: reuse one wrapper per module root instead of creating and leaking one per validation. [[PR]](https://github.com/OffchainLabs/nitro/pull/4514)
- Fix BroadcastClients launching coordination goroutine on child Router's StopWaiter instead of its own. [[PR]](https://github.com/OffchainLabs/nitro/pull/4514)
- Fix ValidationServer and ExecutionSpawner missing StopAndWait for their children. [[PR]](https://github.com/OffchainLabs/nitro/pull/4514)
- Handle too-short AnyTrust certificate data as empty batches instead of crashing. [[PR]](https://github.com/OffchainLabs/nitro/pull/4517)

### Internal

- ArbGasInfo.GetMultiGasPricingConstraints now returns resources in deterministic order. [[PR]](https://github.com/OffchainLabs/nitro/pull/4479)
- Move exponent validation in ArbOwner.SetMultiGasPricingConstraints outside the per-constraint loop. [[PR]](https://github.com/OffchainLabs/nitro/pull/4482)
- Add JWT authentication support to the Rust validation server. [[PR]](https://github.com/OffchainLabs/nitro/pull/4500)
- Move express lane service and tracker from `execution/gethexec` to the `timeboost` package. [[PR]](https://github.com/OffchainLabs/nitro/pull/4484)
- Block sequencing until address filter rules are loaded. [[PR]](https://github.com/OffchainLabs/nitro/pull/4488)

## [v3.10.0-rc.4](https://github.com/OffchainLabs/nitro/compare/v3.10.0-rc.3...v3.10.0-rc.4) - 2026-03-27

### Configuration

- Adds --execution.sequencer.transaction-filtering.disable-delayed-sequencing-filter to enable/disable filtering when sequencing delayed messages. [[PR]](https://github.com/OffchainLabs/nitro/pull/4377)
- Added `--node.block-validator.validation-spawning-allowed-timeouts` (default `3`): maximum number of timeout errors allowed per validation before treating it as fatal. Timeout errors have their own counter, separate from `--node.block-validator.validation-spawning-allowed-attempts`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4455)

### Added

- Peform DNS lookups with IPv4 before IPv6. [[PR]](https://github.com/OffchainLabs/nitro/pull/4461)
- messageSequencingMode messageRunModes. To be used when filtering transactions in the geth layer. [[PR]](https://github.com/OffchainLabs/nitro/pull/4377)
- Fix Message Extraction function to handle cases when number of batch posting reports are not equal to the number of batches. [[PR]](https://github.com/OffchainLabs/nitro/pull/4464)

### Changed

- Make `PruneExecutionDB` only depend on `executionDB` by removing `consensusDB` dependency. [[PR]](https://github.com/OffchainLabs/nitro/pull/4217)

### Removed

- Remove aborted snap sync code. [[PR]](https://github.com/OffchainLabs/nitro/pull/4419)

### Fixed

- Part 1 of improving the MEL runner with latest, tested implementation. [[PR]](https://github.com/OffchainLabs/nitro/pull/4398)
- Use defer to release createBlocksMutex in sequencerWrapper to prevent deadlock on panic. [[PR]](https://github.com/OffchainLabs/nitro/pull/4431)
- Fix opening classic-msg database. [[PR]](https://github.com/OffchainLabs/nitro/pull/4065)
- Fix system test triggered panic in `updateFilterMapsHeads`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4399)
- Fix address filter S3 syncer failing to parse hash list JSON when salt or hash values use `0x`/`0X` hex prefix. Go's `encoding/hex.DecodeString` does not handle the prefix, so it is now stripped before decoding. [[PR]](https://github.com/OffchainLabs/nitro/pull/4435)
- Fix `debug_executionWitness` endpoint. [[PR]](https://github.com/OffchainLabs/nitro/pull/4401)
- Improve block validator error message to suggest enabling staker in watchtower mode when wasmModuleRoot is not set from chain. [[PR]](https://github.com/OffchainLabs/nitro/pull/4451)
- If batchFetcher returns error use existing LegacyBatchGasCost value. [[PR]](https://github.com/OffchainLabs/nitro/pull/4448)
- Fix `rlp: expected List` error when fetching transaction receipts for blocks with Arbitrum legacy receipt encoding. [[PR]](https://github.com/OffchainLabs/nitro/pull/4469)
 - Block validator no longer crashes on timeout errors during validation. Timeout errors are retried separately from other validation failures, up to a configurable limit. [[PR]](https://github.com/OffchainLabs/nitro/pull/4455)

### Internal

- Make Validator request match clients request format. [[PR]](https://github.com/OffchainLabs/nitro/pull/4414)
- Cache precompiled wasm modules for repeated JIT validation. [[PR]](https://github.com/OffchainLabs/nitro/pull/4429)
- S3Syncer's context moved out from new to Initialize. [[PR]](https://github.com/OffchainLabs/nitro/pull/4409)
- `Bytes32`'s `Debug` and `Display` is prefixed with 0x. [[PR]](https://github.com/OffchainLabs/nitro/pull/4439)
- Allow validator's JitMachine to find jit path. [[PR]](https://github.com/OffchainLabs/nitro/pull/4415)
- Migrate Rust validation server to JSON RPC to match Go client communication. [[PR]](https://github.com/OffchainLabs/nitro/pull/4450)
- Move wavmio logic from JIT crate to caller-env (to be reused soon by SP1 validator). [[PR]](https://github.com/OffchainLabs/nitro/pull/4472)
- Make `ValidationInput.max_user_wasm_size` field non-mandatory. [[PR]](https://github.com/OffchainLabs/nitro/pull/4476)

## [v3.10.0-rc.3](https://github.com/OffchainLabs/nitro/compare/v3.10.0-rc.2...v3.10.0-rc.3) - 2026-02-23

### Added

- New dangerous parameters `--node.bold.dangerous.assume-valid` and `--node.bold.dangerous.assume-valid-hash` to have validator assume all messages up to given message have already been validated. [[PR]](https://github.com/OffchainLabs/nitro/pull/4369)
- Filtered retryable submission redirect: when an ArbitrumSubmitRetryableTx is in the onchain filter, redirect beneficiary/feeRefundAddr and skip auto-redeem. [[PR]](https://github.com/OffchainLabs/nitro/pull/4404)
- Add consensus v51.1 to dockerfile. [[PR]](https://github.com/OffchainLabs/nitro/pull/4422)

### Fixed

- Fix expose-multigas feature when using a live tracer. [[PR]](https://github.com/OffchainLabs/nitro/pull/4383)
 - Gracefully handle missing sequencerClient in TransactionFilterer. [[PR]](https://github.com/OffchainLabs/nitro/pull/4397)
- Fix filtered `ArbitrumDepositTx` (L1-to-L2 ETH deposits) permanently stalling the delayed sequencer. [[PR]](https://github.com/OffchainLabs/nitro/pull/4367)
- Version-gate FilteredTransactionsState so it is not opened on pre-v60 blocks. [[PR]](https://github.com/OffchainLabs/nitro/pull/4407)

### Internal

- Introduce MachineLocator for Validator. [[PR]](https://github.com/OffchainLabs/nitro/pull/4350)
 - Fix machine locator tests. [[PR]](https://github.com/OffchainLabs/nitro/pull/4391)
- Add support for consensus v60-alpha.1. [[PR]](https://github.com/OffchainLabs/nitro/pull/4410)

## [v3.10.0-rc.2](https://github.com/OffchainLabs/nitro/compare/v3.10.0-rc.1...v3.10.0-rc.2) - 2026-02-23

### Configuration

- The new `--node.batch-poster.compression-levels` flag allows operators to specify different compression strategies based. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - `backlog`: The minimum backlog size (in number of batches) at which this configuration applies. First entry must be zero. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - `level`: The initial compression level applied to messages when they are added to a batch once the backlog reaches or exceeds the configured threshold. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - `recompression-level`: The recompression level to use for already compressed batches when the backlog meets or exceeds the threshold. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - Example configuration:. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
- Validation rules:. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - The `backlog` values must be in strictly ascending order. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - Both level and recompression-level must be weakly descending (non-increasing) across entries. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - recompression-level must be greater than or equal to level within each entry (recompression should be at least as good as initial compression). [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - All levels must be in valid range: 0-11. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
  - Add `--execution.address-filter.enable` flag to enable/disable address filtering. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Add `--execution.address-filter.poll-interval` flag to set the polling interval for the s3 syncer, e.g. 5m. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Add `--execution.address-filter.cache-size` flag to set the LRU cache size for address lookups (default: 10000). [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Add `--execution.address-filter.s3.*` group of flags to configure S3 access:. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.bucket` flag to specify the S3 bucket name for the hashed address list. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.region` flag to specify the AWS region. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.object-key` flag to specify the S3 object key for the hashed address list. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.access-key` flag to specify the AWS access key. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.secret-key` flag to specify the AWS secret key. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.chunk-size-mb` flag to set S3 multipart download part size in MB (default: 32). [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.concurrency` flag to set S3 multipart download concurrency (default: 10). [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
    - Add `--execution.address-filter.s3.max-retries` flag to set maximum retries for S3 part body download (default: 5). [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
 - Init config must not have `empty` set to true when `genesis-json-file` is provided. [[PR]](https://github.com/OffchainLabs/nitro/pull/4296)
- Add `execution.sequencer.event-filter.path` to configure sequencer-side event-based transaction filtering via a JSON rules file. [[PR]](https://github.com/OffchainLabs/nitro/pull/4271)
 - Extend genesis.json with `serializedConfig` and `arbOSInit.initialL1BaseFee` fields. [[PR]](https://github.com/OffchainLabs/nitro/pull/4292)
 - Remove `initial-l1-base-fee` CLI flag from genesis-generator. [[PR]](https://github.com/OffchainLabs/nitro/pull/4292)
- Added `--execution.address-filter.s3.endpoint` for S3-compatible services (MinIO, localstack). [[PR]](https://github.com/OffchainLabs/nitro/pull/4311)
- Add `--node.data-availability.rest-aggregator.connection-wait` how long to wait for initial anytrust DA connection until it errors (re-attempts every 1 second) (to be deprecated, use `da.anytrust*` instead). [[PR]](https://github.com/OffchainLabs/nitro/pull/4297)
- Add `--node.da.anytrust.rest-aggregator.connection-wait` how long to wait for initial anytrust DA connection until it errors (re-attempts every 1 second). [[PR]](https://github.com/OffchainLabs/nitro/pull/4297)
- Add `--anytrust.rest-aggregator.connection-wait` how long to wait for initial anytrust DA connection until it errors (re-attempts every 1 second). [[PR]](https://github.com/OffchainLabs/nitro/pull/4297)
- Add address-filter.address-checker-worker-count to configure the number of address checker workers. [[PR]](https://github.com/OffchainLabs/nitro/pull/4235)
- Add address-filter.address-checker-queue-size to configure the address checker queue capacity. [[PR]](https://github.com/OffchainLabs/nitro/pull/4235)

### Added

- `cc_brotli` optional feature which when enabled compiles `brotli` automatically using Rust build scripts. [[PR]](https://github.com/OffchainLabs/nitro/pull/3473)
- transaction-filterer command, responsible to receive a transaction that should be filtered, and adding that transaction to the ArbFilteredTransactionsManager precompile. [[PR]](https://github.com/OffchainLabs/nitro/pull/4227)
- Increase Stylus smart contract size limit via merge-on-activate. [[PR]](https://github.com/OffchainLabs/nitro/pull/4193)
- New `--node.batch-poster.compression-levels` configuration flag that accepts a JSON array of compression configurations based on batch backlog thresholds. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
- Support for defining compression level, recompression level, and backlog threshold combinations. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
- Validation rules to ensure compression levels don't increase with higher backlog thresholds. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
- Add address filter service for compliance chains (`addressfilter` package). This feature enables sequencers to block transactions involving filtered addresses by polling a hashed address list from S3. Key capabilities include:. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - S3-based hashed list synchronization with ETag change detection for efficient polling. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Lock-free HashStore using atomic pointer swaps for zero-blocking reads during updates. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Configurable LRU cache for high-performance address lookups (default: 10k entries). [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Privacy-preserving design: addresses are never stored or transmitted in plaintext (SHA256 with salt). [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Forward-compatible hash list JSON format with `hashing_scheme` metadata field. [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
  - Configurable S3 download settings (part size, concurrency, retries). [[PR]](https://github.com/OffchainLabs/nitro/pull/4234)
 - Added a new hook to `replay.wasm` to enable an action just before the first IO (wavmio) instruction. It is expected that every `wasm` execution environment will provide a module `hooks` with a method `beforeFirstIO`. JIT and Arbitrator provers have noop implementations. [[PR]](https://github.com/OffchainLabs/nitro/pull/4283)
- Introduce event filter module for filtering transaction logs based on event selectors and topic-encoded addresses. [[PR]](https://github.com/OffchainLabs/nitro/pull/4271)
- Execute onchain-filtered delayed transactions as no-ops: nonce is incremented, all gas is consumed, and a failed receipt is produced. The sender pays for the failed transaction as a penalty. [[PR]](https://github.com/OffchainLabs/nitro/pull/4247)
- Added a test for batch resizing without fallback (TestBatchResizingWithoutFallback_MessageTooLarge) that validates ErrMessageTooLarge triggers batch rebuild while staying on the same DA provider. [[PR]](https://github.com/OffchainLabs/nitro/pull/4183)
- Redis Pub/Sub–based executionSpawner implementation, including GetProof support, Redis-first interface selection, and the ability to run without any RPC dependency. [[PR]](https://github.com/OffchainLabs/nitro/pull/2354)
- Multi-gas constraints to L2-pricing simulator. [[PR]](https://github.com/OffchainLabs/nitro/pull/4330)
 - L2 message accumulation in MEL and added MessageReader struct to extract recorded messages from preimages map. [[PR]](https://github.com/OffchainLabs/nitro/pull/4258)
- Hashed address filter implementation for address filter interfaces with shared LRU caching. [[PR]](https://github.com/OffchainLabs/nitro/pull/4235)
- sequencer metrics considering tx size: `arb/sequencer/block/txsize` and `arb/sequencer/transactions/txsize` histograms. [[PR]](https://github.com/OffchainLabs/nitro/pull/4317)
- sequencer queue metrics: `arb/sequencer/queue/length`, `arb/sequencer/queue/histogram`, `arb/sequencer/waitfortx`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4317)
- sequencer block counter metrics: `arb/sequencer/block/gaslimited`, `arb/sequencer/block/datalimited`, `arb/sequencer/block/txexhausted`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4317)
- Add filteredFundsRecipient ArbOS state field and precompile for use on chains with transaction filternig. [[PR]](https://github.com/OffchainLabs/nitro/pull/4347)
- arb_getL1Confirmations and arb_findBatchContainingBlock RPC APIs in Consensus side. [[PR]](https://github.com/OffchainLabs/nitro/pull/3985)
- Sequencer calls transaction-filterer command if delayed transaction was filtered. [[PR]](https://github.com/OffchainLabs/nitro/pull/4294)
- Config option genesis-json-file-directory which specifies the directory where genesis json files are located. [[PR]](https://github.com/OffchainLabs/nitro/pull/4291)
 - Adding/removing ChainOwner and NativeTokenOwner emits corresponding events. [[PR]](https://github.com/OffchainLabs/nitro/pull/4364)

### Changed

- Replace static batch poster compression configuration with dynamic, backlog-based compression level system. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)
- Generate `forward_stub.wasm` at compile time using a `build.rs` script. Enables using `prover` as a cargo dependency. [[PR]](https://github.com/OffchainLabs/nitro/pull/3447)
- improve forwarding transaction log in case of error. [[PR]](https://github.com/OffchainLabs/nitro/pull/4288)
- Renamed `arbkeccak` wasm module to `arbcrypto`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4290)
- For wasm compilation target, EC recovery is now expected to be provided by the external `arbcrypto` module. For JIT and arbitrator provers we inject Rust-based implementation. [[PR]](https://github.com/OffchainLabs/nitro/pull/4290)
 - genesis-generator will now read chain config and init message data directly from genesis.json. [[PR]](https://github.com/OffchainLabs/nitro/pull/4292)
- update ProgramPrepare to accept wasm and wasm_size. [[PR]](https://github.com/OffchainLabs/nitro/pull/4284)
- remove unecessary statedb, addressForLogging, codePtr, codeSize, time, program, runCtxPtr params from ProgramPrepare. [[PR]](https://github.com/OffchainLabs/nitro/pull/4284)
- Refactor openInitializeChainDb for Execution/Consensus split. [[PR]](https://github.com/OffchainLabs/nitro/pull/4169)
- Create new config and init package to expose and organize init and config nitro functionality. [[PR]](https://github.com/OffchainLabs/nitro/pull/4169)
 - Nitro initialization uses the serialized chain config from genesis (instead of the deprecated `Config` field). [[PR]](https://github.com/OffchainLabs/nitro/pull/4313)

### Deprecated

- Deprecate `--node.batch-poster.compression-level` flag in favor of `--node.batch-poster.compression-levels`. [[PR]](https://github.com/OffchainLabs/nitro/pull/4145)

### Fixed

 - Fixes bold next batch index and message count computation to cap based on relative comparisons. [[PR]](https://github.com/OffchainLabs/nitro/pull/4279)
 - Fix typed nil ExecutionSequencer in CreateConsensusNode causing crash in RPC client mode. [[PR]](https://github.com/OffchainLabs/nitro/pull/4299)
- Fix address filter bypass for aliased addressed, unsigned delayed messages (L1-to-L2). [[PR]](https://github.com/OffchainLabs/nitro/pull/4314)
- Fix calculation in the expected surplus in the sequencer metrics. [[PR]](https://github.com/OffchainLabs/nitro/pull/4038)

### Internal

- Implement capacity endpoint for Rust Validator. [[PR]](https://github.com/OffchainLabs/nitro/pull/4262)
- Changed the max stylus contract fragments from uint16 to uint8 in ArbOwner and ArbOwnerPublic precompiles to not waste storage space. [[PR]](https://github.com/OffchainLabs/nitro/pull/4285)
 - Add continuous mode to JIT validator. [[PR]](https://github.com/OffchainLabs/nitro/pull/4269)
 - Introduce `JitMachine` (equivalent to Go counterpart `JitMachine`). [[PR]](https://github.com/OffchainLabs/nitro/pull/4269)
 - Introduce graceful shutdown through signals. [[PR]](https://github.com/OffchainLabs/nitro/pull/4269)
 - Move the server side of the validation communication protocol from `jit` to `validation` crate. [[PR]](https://github.com/OffchainLabs/nitro/pull/4280)
 - Add client side implementation. Add tests. [[PR]](https://github.com/OffchainLabs/nitro/pull/4280)
- Merge wasm-libraries workspace with the main one. [[PR]](https://github.com/OffchainLabs/nitro/pull/4298)
- Add support for multiple module roots for Validator. [[PR]](https://github.com/OffchainLabs/nitro/pull/4310)
- Fix Validator continuous mode to run jit binary from inside tokio runtime. [[PR]](https://github.com/OffchainLabs/nitro/pull/4310)
- Add BurnMultiGas to Burner interface. [[PR]](https://github.com/OffchainLabs/nitro/pull/4312)
 - Fix checking machine status in the Rust validation. [[PR]](https://github.com/OffchainLabs/nitro/pull/4344)
 - Run the continuous mode unit test in CI. [[PR]](https://github.com/OffchainLabs/nitro/pull/4344)
- Add benchmarks comparing l2-pricing models. [[PR]](https://github.com/OffchainLabs/nitro/pull/4340)

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
- Fixed batch poster on L3s not waiting for transaction receipt before posting next batch, causing duplicate batch attempts and spurious error logs. [[PR]](https://github.com/OffchainLabs/nitro/pull/4273)

### Internal

- [MEL] - Implement delayed message accumulation in native mode: [[PR]](https://github.com/OffchainLabs/nitro/pull/3389)
- [MEL] - Update melextraction package to use logs instead of receipts and implement logs and headers fetcher: [[PR]](https://github.com/OffchainLabs/nitro/pull/4063)
- [MEL] - Implement preimage recorder for `DelayedMessageDatabase` interface: [[PR]](https://github.com/OffchainLabs/nitro/pull/4119)
- [MEL] - Implement recording of preimages related to sequencer batches (DA providers): [[PR]](https://github.com/OffchainLabs/nitro/pull/4133)
- Add new boolean option to `BlocksReExecutor` called `CommitStateToDisk` that will allow `BlocksReExecutor.Blocks` range to not only re-executes blocks but it will also commit their state to triedb on disk. [[PR]](https://github.com/OffchainLabs/nitro/pull/4132)