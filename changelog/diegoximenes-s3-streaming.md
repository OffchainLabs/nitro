### Changed
- The address-filter hash list is no longer buffered fully in memory: the S3 object is multipart-downloaded to a temporary file (new `--…s3.download-dir` option, defaulting to the OS temp dir) and stream-parsed straight into the hash store.
