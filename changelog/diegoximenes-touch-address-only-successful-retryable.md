### Fixed
- Address filter now touches the de-aliased L1 sender of retryable submissions and deposits. Previously a filtered L1 sender of a retryable was only caught through its auto-redeem (or not at all without one), and a filtered L1 contract depositor slipped through entirely.
- Early submission-failure paths (e.g. underfunded max submission fee) now return `ErrFilteredOnChain` for onchain-filtered retryables, so the delayed sequencer no longer re-halts forever on the same hash.
- `TxFailed` deduplicates `filteredTxHashes`: a filtered sender touched by both the submission tx and its auto-redeem no longer records the same originating hash twice in the halt state and reports.
- A redeem's `RefundTo` address is now reported to the address filter via a new `ReasonRetryableRefundTo`, only when refund funds actually reach it and at most once per transaction.

### Changed
- Retryable `Beneficiary`/`FeeRefundAddr`/`RetryTo` touching moved from `PostTxFilter` into `StartTxHook`, at the points funds move to them or the retryable is created. A submission that fails early no longer touches those addresses, so it can't be used to spam-halt the delayed sequencer with filtered addresses that never receive funds.
