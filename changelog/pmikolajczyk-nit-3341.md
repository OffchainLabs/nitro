### Fixed
 - Block validator no longer crashes on timeout errors during validation. Timeout errors are retried separately from other validation failures, up to a configurable limit.

### Configuration
 - Added `--node.block-validator.validation-spawning-allowed-timeouts` (default `3`): maximum number of timeout errors allowed per validation before treating it as fatal. Timeout errors have their own counter, separate from `--node.block-validator.validation-spawning-allowed-attempts`.
