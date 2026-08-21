### Configuration
- Added the `execution.sequencer.max-block-tx-candidates` option, which bounds how many transactions the sequencer drains from the pending queue to be considered for a block. This prevents the retry queue from growing without limit under sustained load.

### Changed
- The sequencer now rejects configurations where `execution.sequencer.max-block-speed` divided by `execution.sequencer.pga.rounds-per-block` yields a PGA round length below 5ms. Such configurations were previously accepted.
