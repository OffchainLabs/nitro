### Fixed
- Address filter now records the L1 destination of `ArbSys.withdrawEth` / `ArbSys.sendTxToL1` via a new `ReasonToL1`, closing a gap where L2→L1 withdrawals to a filtered address were not caught.

### Changed
- `S3SyncManager` now includes `filterSetID` in the successful hash-list load log line so operators can correlate it with the `FilterSetID` reported on filtered tx reports.
