### Changed
- The sequencer rejects same-sender, same-nonce txs unless PGA is on and the tip is higher.

### Configuration
- Add `--execution.sequencer.reject-noop-replacements` (default true).
- Add `--execution.sequencer.pga.min-replacement-percentage` (default 10).

### Added
- Add metric `arb/sequencer/pending/count/gauge`.
- Add metric `arb/sequencer/pending/count/histogram`.
- Add metric `arb/sequencer/pending/rejected/alreadyknown/counter`.
- Add metric `arb/sequencer/pending/rejected/noncepending/counter`.
- Add metric `arb/sequencer/pending/rejected/replaceunderpriced/counter`.
- Add metric `arb/sequencer/pending/replaced/counter`.
