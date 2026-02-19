### Fixed
- Fix race condition in event producer shutdown where receiver-side channel close could panic concurrent broadcast goroutines

### Ignored
- Speed up BOLD challenge protocol tests via subscription-based watcher triggering, execution run caching, and subscription-based WaitMined
- Extract shared helpers for BOLD challenge test node creation to deduplicate ~700 lines across test files
- Cache execution runs in Redis validation consumer to avoid redundant machine creation
