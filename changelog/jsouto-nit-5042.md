### Changed

- After producing a block, the sequencer throttles to the next PGA round boundary instead of a full `max-block-speed`, so a block that closes early opens the next one at the following round boundary. FIFO pacing is unchanged.
