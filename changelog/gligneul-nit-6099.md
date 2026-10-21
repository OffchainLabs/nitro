### Changed
- The sequencer rejects a second tx for an in-flight nonce unless it bumps the fees under PGA.

### Configuration
- Add `--execution.sequencer.reject-noop-replacements` (default true).
- Add `--execution.sequencer.replace-price-bump` (default 10).

### Added
- Add metric `arb/sequencer/inflight/size`.
- Add metric `arb/sequencer/inflight/rejected/alreadyknown`.
- Add metric `arb/sequencer/inflight/rejected/nonceinflight`.
- Add metric `arb/sequencer/inflight/rejected/replaceunderpriced`.
- Add metric `arb/sequencer/inflight/replaced`.
