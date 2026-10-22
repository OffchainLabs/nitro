### Changed
- The sequencer rejects same-sender, same-nonce txs unless PGA is on and the tip is higher.

### Configuration
- Add `--execution.sequencer.reject-noop-replacements` (default true).
- Add `--execution.sequencer.pga.min-replacement-percentage` (default 10).

### Added
- Add metric `arb/sequencer/inflight/size`.
- Add metric `arb/sequencer/inflight/rejected/alreadyknown`.
- Add metric `arb/sequencer/inflight/rejected/nonceinflight`.
- Add metric `arb/sequencer/inflight/rejected/replaceunderpriced`.
- Add metric `arb/sequencer/inflight/replaced`.
