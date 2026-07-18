### Fixed
- Fixed flaky TestGasEstimationWithRPCGasLimit by waiting for the 2nd nodes' execution heads to catch up before estimating gas, and bounding the estimation RPCs with a timeout so a stall fails fast instead of hanging into the package timeout.
