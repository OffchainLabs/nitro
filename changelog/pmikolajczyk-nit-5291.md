### Configuration
- Renamed `--node.block-validator.dangerous.revalidation.start-block`/`end-block` to `start-batch`/`end-batch`; they always took batch numbers. The old names have no aliases.

### Changed
- Revalidation now only moves the last validated state backwards. A start batch ahead of it used to mark everything in between as validated without validating it; use `--node.bold.dangerous.assume-valid` for that instead.

### Fixed
- An unusable revalidation start batch, one that is unknown or not behind the last validated state, no longer aborts node startup; the node logs it and keeps validating from where it left off.
