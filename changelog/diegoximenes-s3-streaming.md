### Changed
- The address-filter hash list is no longer buffered fully in memory: the S3 object is multipart-downloaded to a temporary file (new required `--…s3.download-dir` option) and stream-parsed straight into the hash store.
