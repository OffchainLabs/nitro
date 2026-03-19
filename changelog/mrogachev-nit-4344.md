### Internal
- Tx pre-checker speculatively executes the transaction to detect filtered addresses.

### Configuration
- `--execution.tx-pre-checker.speculative-filter-gas-cap` (default: 50000000): gas cap for speculative address filter execution and cumulative redeem budget