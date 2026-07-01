### Fixed
- `callTracer.OnTxEnd` now returns early if the top-level frame was never captured (e.g. on timeouts)
