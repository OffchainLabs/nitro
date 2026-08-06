### Added
- Address filter: new `plaintext` hashing scheme for S3 hash-list files — `hashes` entries are plain `0x`-prefixed addresses and `salt` is ignored.
- Address filter: new `--execution.transaction-filtering.address-filter.static-list` option — an inline hash-list JSON document merged with the S3 lists; when set, configuring S3 files becomes optional.

### Breaking changes
- The address filter now supports multiple S3 hash-list files, each with its own polling interval. The single-file options `--execution.transaction-filtering.address-filter.s3.*` and `--execution.transaction-filtering.address-filter.poll-interval` were removed; configure `transaction-filtering.address-filter.files` in a config file or `--execution.transaction-filtering.address-filter.files-list` (a JSON string with durations as nanosecond integers). An address is filtered if it appears in any list; the node only starts after every list downloads successfully. Per-file metrics are exported under `arb/addressfilter/file/<index>/…`.
