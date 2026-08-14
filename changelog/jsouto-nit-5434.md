### Changed

- The sequencer now ends a block as soon as the remaining block gas falls below the intrinsic tx cost, instead of having the block processor discard the queued txs one by one. Replay and delayed-message block production are unchanged.
- The `arb/sequencer/block/gaslimited`, `arb/sequencer/block/datalimited`, and `arb/sequencer/block/txexhausted` counters are attributed more precisely: `txexhausted` counts only blocks that drained the tx queue without skipping any tx, and blocks ended by the PGA round schedule no longer count as `datalimited`. At most one of the three increments per block.
