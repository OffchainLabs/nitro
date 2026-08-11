### Added
- Address filter: new `plaintext` hashing scheme for S3 hash-list files.
- Address filter: new `--execution.transaction-filtering.address-filter.static-list` option — an inline hash-list merged with the S3 lists.
- Address filter: new per-file `min-bytes-per-hash-entry` option sizing memory preallocation; defaults to 42 (the smallest entry any scheme allows), and lists holding only sha256 entries can set 66 to avoid preallocating for entries the list can never contain.

### Removed
- The address filter now supports multiple S3 hash-list files, each with its own polling interval. The options `--execution.transaction-filtering.address-filter.s3.*` and `--execution.transaction-filtering.address-filter.poll-interval` were removed; configure the `transaction-filtering.address-filter.files` array in a config file (`--conf.file`) or inline via `--conf.string` instead.
