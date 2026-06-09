### Fixed
- Sequencer no longer rejects clean transactions following a filtered one in the same block. `PreTxFilter` now operates on a fresh per-tx `addressCheckerState` instead of inheriting the previous tx's filter state.
