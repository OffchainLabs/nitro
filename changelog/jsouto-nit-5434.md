### Changed

- The sequencer now ends a block as soon as the remaining block gas falls below the intrinsic tx cost, instead of having the block processor discard the queued txs one by one. Replay and delayed-message block production are unchanged.
- The `arb/sequencer/block/gaslimited`, `arb/sequencer/block/datalimited`, and `arb/sequencer/block/txexhausted` counters are no longer mutually exclusive: a block increments each counter whose limit it hit, and `txexhausted` counts only blocks that drained the queue without skipping any tx.
