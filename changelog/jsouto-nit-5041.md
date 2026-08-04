### Added

- Priority gas auction (PGA) transaction ordering: when the chain collects tips, the sequencer splits each block into rounds and orders transactions by priority instead of FIFO.
- A tx parked on a nonce gap re-enters the current block as soon as its predecessor is included, instead of waiting in the retry queue.

### Configuration

- Replace `execution.sequencer.experimental-pga.enable` with `execution.sequencer.experimental-pga.dangerous-force-fifo`, which forces FIFO ordering even when the chain collects tips (default `false`).
