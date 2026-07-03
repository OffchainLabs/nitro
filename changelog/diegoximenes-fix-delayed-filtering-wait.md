### Fixed
- The delayed sequencer no longer keeps a filtered delayed message halted while waiting for a *later* delayed message to finalize. Once the filtered (and already-finalized) message's hash is added to the onchain filter, it resumes immediately instead of being blocked by `waitingForFinalizedBlock`, which previously inflated `delayedsequencer/filtered_tx_wait_seconds` by up to a parent-chain finality lag.

### Changed
- Lowered the "Delayed message filtered - HALTING delayed sequencer" log from `Error` to `Info`, since halting on a filtered delayed message is expected operation.
