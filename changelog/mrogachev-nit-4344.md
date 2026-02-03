### Added
- Add transaction prechecker address filtering, mirroring sequencer-side behaviour

### Configuration
- Add `execution.tx-prechecker.apply-transaction-filter` to enable address filtering during tx precheck
- Add `execution.tx-prechecker.event-filter` to configure event-based address extraction for filtering

### Changed
- When filtering is enabled, the prechecker executes transactions in a sandbox to discover touched addresses