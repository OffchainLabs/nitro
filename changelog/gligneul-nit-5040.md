### Internal
- The execution engine's `SequenceTransactions` family now takes a `BlockSequencingHooks` interface instead of the concrete `*FullSequencingHooks`, decoupling the engine from the sequencer's hooks implementation.
- The hooks report their sequenced transactions via `SequencedTxes`; building the L2 message from them (`MessageFromTxes`) is now a standalone function owned by the execution engine.
- `FullSequencingHooks` moved to its own file and its filter function pointers were replaced by `TxFilter` and `BlockFilter` interfaces; the sequencer implements `TxFilter` directly.
- The block processor now skips `PostTxFilter` for internal txs, so hooks implementations no longer special-case `ArbitrumInternalTxType`.
