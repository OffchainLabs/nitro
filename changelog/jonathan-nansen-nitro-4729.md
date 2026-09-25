### Fixed
- `OpenExistingExecutionDB` no longer treats a failed read-only probe of `l2chaindata` as fatal. If the probe fails for a reason other than the database not existing (e.g. an ancient/freezer table left torn by a write that failed under disk or inode exhaustion), it now retries read-write, which allows the freezer to repair the torn table instead of crash-looping.
