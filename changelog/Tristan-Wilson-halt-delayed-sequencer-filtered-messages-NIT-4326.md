### Added
- Halt the delayed sequencer when a delayed message touches a filtered address, waiting for the tx hash to be added to the onchain filter before resuming. Includes fast onchain filter polling and configurable full-retry interval (`delayed-sequencer.filtered-tx-full-retry-interval`).
