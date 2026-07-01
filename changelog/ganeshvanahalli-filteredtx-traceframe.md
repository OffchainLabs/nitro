### Fixed
- Emit a balanced top-level call frame for EVM-skipped transactions (onchain-filtered txs, pre-recorded reverts, and retryable-redeem error paths) so tracing no longer fails with "incorrect number of top-level calls"
