### Added
- Optional `--execution.transaction-filtering.address-filter.s3.max-file-size-mb` flag that skips downloading the filtered-addresses S3 file when it exceeds the configured size (0 disables the check). Adds prometheus metrics `arb/addressfilter/file/size`, `arb/addressfilter/file/toolarge_total`, and `arb/addressfilter/sync/failure_total`.
