### Configuration
- Added the `execution.sequencer.max-block-tx-candidates` option, which bounds how many transactions the sequencer drains from the pending queue to be considered for a block. This prevents the retry queue from growing without limit under sustained load.
