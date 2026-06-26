### Configuration
- The periodic full re-execution interval used while the sequencer is halted on a
  filtered delayed message moved from
  `--node.delayed-sequencer.filtered-tx-full-retry-interval` to
  `--execution.transaction-filtering.filtered-tx-full-retry-interval` (default 30s,
  now validated to be positive). The dead field on `DelayedSequencerConfig` was
  removed.

### Changed
- Block production is now driven by consensus (`TransactionStreamer`) and gated on
  the execution sequencer's active state (`IsActive()`) instead of the
  coordinator's `ExpectChosenSequencer` check. The authoritative redis lockout
  check still runs downstream in `WriteSequencedMsg`, so correctness is unchanged;
  `IsActive()` additionally accounts for forwarding state and the
  activation-readiness window.
- Delayed messages are now enqueued by consensus (`EnqueueDelayedMessages`, sent in
  batches) and sequenced by execution, replacing the previous one-at-a-time
  `SequenceDelayedMessage` path.
- Regular transactions and delayed messages now alternate sequencing turns so
  neither starves the other; a delayed message halted on a filtered tx is retried
  at most once per `MaxBlockSpeed` instead of busy-looping.

### Removed
- `ExpectChosenSequencer` and the `ConsensusSequencer` interface, plus the unused
  `IsTxHashInOnchainFilter` method from the `ExecutionSequencer` interface.

### Internal
- Sequencing orchestration moved out of the execution-side `SequencerTriggerer`
  into `TransactionStreamer`; the `ExecutionSequencer` interface now exposes
  `StartSequencing`/`EndSequencing`/`AppendLastSequencedBlock`/
  `ResequenceReorgedMessage` and a `SequencedMsg` result type. Numerous supporting
  refactors (sequencer pause/forwarder simplification, queue helpers, test helpers
  replacing `SequenceTransactionsForTest`).
