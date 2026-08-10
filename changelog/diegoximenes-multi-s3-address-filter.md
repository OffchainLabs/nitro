### Added
- Address filter: new `plaintext` hashing scheme for S3 hash-list files.
- Address filter: new `--execution.transaction-filtering.address-filter.static-list` option — an inline hash-list merged with the S3 lists.

### Breaking changes
- The address filter now supports multiple S3 hash-list files, each with its own polling interval. The options `--execution.transaction-filtering.address-filter.s3.*` and `--execution.transaction-filtering.address-filter.poll-interval` were removed; configure `transaction-filtering.address-filter.files` in a config file or `--execution.transaction-filtering.address-filter.files-list` instead.
