### Ignored
- Test harness: `waitForNodeToCatchUpWithParentChain` now reads batch progress from the MEL message extractor when it replaces the inbox tracker, instead of silently skipping the catch-up wait for every MEL node (fixes `TestAnyTrustRekey`)
