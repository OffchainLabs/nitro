### Changed
- Small transactions can overtake large ones after consecutive data-limited FIFO blocks.

### Added
- Add metric `arb/sequencer/fifo/sizesampling/skipped`.
- Add metric `arb/sequencer/fifo/sizesampling/exponent`.

### Configuration
- Add `--execution.sequencer.fifo.size-sampling-max-exponent` (default 10, 0 disables, at most 100).
- Add `--execution.sequencer.fifo.size-sampling-grace-blocks` (default 1).
- Add `--execution.sequencer.fifo.size-sampling-min-tx-size` (default 1024).
