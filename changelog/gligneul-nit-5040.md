### Internal
- The execution engine's `SequenceTransactions` family now takes a `BlockSequencingHooks` interface instead of the concrete `*FullSequencingHooks`, decoupling the engine from the sequencer's hooks implementation.
- `FullSequencingHooks` moved to its own file and its filter function pointers were replaced by `TxFilter` and `BlockFilter` interfaces; the sequencer implements `TxFilter` directly.
