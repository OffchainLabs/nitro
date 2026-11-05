### Configuration
- Added `tx-pre-checker.max-pending-conditional-txs`, hot-reloadable; 0 (the default) disables the cap.
### Added
- A node can cap conditional transactions in flight, rejecting extras with RPC error -32012.
- Added metric `arb/txprechecker/conditionaltx/pending/gauge`.
- Added metric `arb/txprechecker/conditionaltx/rejected/pendingcap/counter`.
