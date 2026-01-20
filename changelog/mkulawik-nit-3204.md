### Added
 - Add `execution.caching.trie-cap-batch-size` option that sets batch size in bytes used in the TrieDB Cap operation (0 = use geth default)
 - Add `execution.caching.trie-commit-batch-size` option that sets batch size in bytes used in the TrieDB Commit operation (0 = use geth default)
 - Add database batch size checks to prevent panic on pebble batch overflow
