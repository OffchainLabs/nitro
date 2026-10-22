### Fixed
- Fixed flaky `TestSequencerInboxReader`: after a simulated L1 reorg the test could check L2 state while the node was still asynchronously rewinding and re-digesting messages; it now waits for the consensus batch count and exact execution head before checking.
