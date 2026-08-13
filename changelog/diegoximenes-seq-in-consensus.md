### Configuration
- The periodic full re-execution interval used while the sequencer is halted on a
  filtered delayed message moved from
  `--node.delayed-sequencer.filtered-tx-full-retry-interval` to
  `--execution.transaction-filtering.filtered-tx-full-retry-interval` (default 30s,
  now validated to be positive). The dead field on `DelayedSequencerConfig` was
  removed.
- New `--execution.sequencer.poll-interval` (default 10ms, hot-reloadable,
  validated to be positive): the interval an idle sequencer waits before
  re-checking for pending work, instead of busy-looping. Capped at
  `MaxBlockSpeed`.

### Changed
- Block production is now driven by consensus (`TransactionStreamer`) and gated on
  the execution sequencer's active state (`IsActive()`) instead of the
  coordinator's `ExpectChosenSequencer` check. The authoritative redis lockout
  check still runs downstream in `WriteSequencedMsg`, so correctness is unchanged;
  `IsActive()` additionally accounts for forwarding state and the
  activation-readiness window.
- Delayed messages are now enqueued by consensus (`EnqueueDelayedMessages`, sent in
  batches) and sequenced by execution, replacing the previous one-at-a-time
  `SequenceDelayedMessage` path. Batches misaligned with the expected next delayed
  index (e.g. from a concurrent enqueue during sequencer handoff) are trimmed or
  dropped instead of poisoning the queue.
- `arb_checkPublisherHealth` still reports "not chosen" for an active node whose
  lockout expired, without the exec→consensus query: the coordinator mirrors its
  lockout deadline into the execution sequencer (`SetActiveUntil`), and
  `CheckHealth` reports unhealthy past that deadline.
- If a sequenced message is durably written by consensus but the exec-chain append
  fails, its txs are reported successful (they are part of the canonical message)
  and the execution chain heals by re-digesting the message; sequencing pauses
  while the execution head lags the consensus head, and commit failures retry with
  a backoff instead of immediately.
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
  `ResequenceReorgedMessage`/`SetActiveUntil` and a `SequencedMsg` result type.
  Numerous supporting refactors (sequencer pause/forwarder simplification, queue
  helpers, test helpers replacing `SequenceTransactionsForTest`).
