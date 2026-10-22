### Changed
- The sequencer rejects same-sender, same-nonce txs unless PGA is on and the tip is higher.

### Configuration
- Add `--execution.sequencer.reject-noop-replacements` (default true).
- Add `--execution.sequencer.pga.min-replacement-percentage` (default 10).

### Added
- Add metric `arb/sequencer/inflight/count/gauge`.
- Add metric `arb/sequencer/inflight/count/histogram`.
- Add metric `arb/sequencer/inflight/rejected/alreadyknown/counter`.
- Add metric `arb/sequencer/inflight/rejected/nonceinflight/counter`.
- Add metric `arb/sequencer/inflight/rejected/replaceunderpriced/counter`.
- Add metric `arb/sequencer/inflight/replaced/counter`.
