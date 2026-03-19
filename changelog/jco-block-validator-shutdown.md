### Fixed
- Suppress spurious validator shutdown errors from context cancellation (`Canceled` at Trace, `DeadlineExceeded` at Warn)
- Extract `handleValidationResult` to deduplicate validation progress error handling; skip reorg attempts during shutdown
- Fix `Reorg` guard rejecting valid `count == 1` (reorg to genesis)
