### Fixed
- The batch poster no longer treats a reverted gas estimation as a successful estimate of zero gas when the follow-up `eth_call` (used only to get a detailed error) happens to succeed, which posted a batch with an insufficient gas limit. A zero gas estimate is now an estimation failure everywhere nitro estimates gas: the batch poster, the validator wallet and BOLD transactions.
