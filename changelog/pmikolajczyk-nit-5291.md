### Configuration
- Renamed `--node.block-validator.dangerous.revalidation.start-block`/`end-block` to `start-batch`/`end-batch`; they always took batch numbers. The old names have no aliases.

### Fixed
- An unknown revalidation start batch no longer aborts node startup with an opaque database error; the node logs it and keeps validating from where it left off.
