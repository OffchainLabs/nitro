### Added
- Address filter: new `plaintext` hashing scheme for S3 hash-list files.
- Address filter: new `--execution.transaction-filtering.address-filter.static-list` option — an inline hash-list merged with the S3 lists.
- Address filter: new per-file `min-bytes-per-hash-entry` option sizing memory preallocation; defaults to 42 (the smallest entry any scheme allows), and lists holding only sha256 entries can set 66 to avoid preallocating for entries the list can never contain.

### Changed
- The address filter now supports multiple S3 hash-list files, each with its own polling interval; configure the `transaction-filtering.address-filter.files` array in a config file (`--conf.file`) or inline via `--conf.string`.
- The address filter S3 option `preallocate-memory` is renamed to `disable-preallocate-memory`, inverting its polarity; preallocation remains enabled by default.

### Removed
- The address filter options `--execution.transaction-filtering.address-filter.s3.*` and `--execution.transaction-filtering.address-filter.poll-interval`; use the `transaction-filtering.address-filter.files` array instead.
