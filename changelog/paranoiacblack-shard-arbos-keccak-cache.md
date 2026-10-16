### Changed
- The ArbOS keccak cache is now sharded, so concurrent `eth_call`s no longer serialize on a single cache mutex.
