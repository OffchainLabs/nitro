### Internal
- `TestEthCallFilterPreservesResultWithScheduledTxes` now compares against a true unfiltered baseline from a second node with transaction filtering disabled, restoring the off-vs-on coverage lost when the test was migrated to the S3 pipeline (which enables filtering at node construction).
