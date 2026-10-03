### Fixed

- `eth_getProof` now rejects requests with more than 1024 storage keys instead of allocating unbounded memory, matching upstream geth's limit.
