### Changed

- The sequencer now ends a block as soon as the remaining block gas falls below the intrinsic tx cost, instead of having the block processor discard the queued txs one by one. Replay and delayed-message block production are unchanged.
