### Configuration
- Add `TransactionFiltering.Enable` master switch (default `false`), replaces `AddressFilter.Enable`.

### Changed
- `EnableETHCallFilter` now scopes to `eth_estimateGas` only; no longer gates prechecker filtering.
- Skip prechecker transaction-filter dry-run on sequencer nodes.
