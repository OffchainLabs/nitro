### Fixed
- Fix nil-dereference and log format in `cmd/nitro/nitro.go` when machine locator creation fails; return early instead of falling through to dereference nil locator
- Harden blocks reexecutor with panic recovery for concurrent trie access races
- Suppress spurious validator shutdown errors from context cancellation (`Canceled` at Trace, `DeadlineExceeded` at Warn)
- Extract `handleValidationResult` to deduplicate validation progress error handling; skip reorg attempts during shutdown
- Fix `Reorg` guard rejecting valid `count == 1` (reorg to genesis)
