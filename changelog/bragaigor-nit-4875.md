### Configuration
- Add `--execution.legacy-zero-base-fee-until <unix-ts>` opt-in compatibility flag (default 0 = disabled). When set, restores the pre-v3.7 behavior of treating headers with `ArbOSFormatVersion <= 40` and `BaseFee == 0` as non-arbitrum, for blocks with timestamp strictly less than the given value.
