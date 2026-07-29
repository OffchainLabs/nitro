### Changed
- An inactive sequencer forwards pending txs only from the background forwarder poll; the block-creation loop no longer forwards inline.

### Fixed
- When a block fails to commit, txs drained but never attempted are no longer failed back to their submitters; they stay in the retry queue.

### Internal
- Introduce the `txOrderer` interface and its `fifoTxOrderer` implementation: the sequencer creates an orderer per block and the sequencing hooks pull the block's txs from it one at a time.
