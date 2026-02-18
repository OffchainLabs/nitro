### Added
- `execution.rpc.error-when-triedb-busy` configuration option: when enabled, RPC calls requiring state access return an immediate error instead of waiting (and potentially timing out) when the HashDB TrieDB is busy committing state
