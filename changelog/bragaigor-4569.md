### Configuration
- Add `--node.dangerous.always-fallback-to-parent-chain-da` (DANGEROUS): makes a node operating against an AnyTrust chain (`ArbitrumChainParams.DataAvailabilityCommittee=true`) behave as if the DA committee is unavailable. The AnyTrust-required check is skipped, the batch poster always posts to the parent chain (calldata / 4844 blobs) even if a DAC writer is configured.
