### Added
- Execute onchain-filtered delayed transactions as no-ops: nonce is incremented, all gas is consumed, and a failed receipt is produced. The sender pays for the failed transaction as a penalty.
