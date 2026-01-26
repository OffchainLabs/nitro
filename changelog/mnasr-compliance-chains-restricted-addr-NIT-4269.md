### Added
- Add address filter service for compliance chains (`addressfilter` package). This feature enables sequencers to block transactions involving filtered addresses by polling a hashed address list from S3. Key capabilities include:
  - S3-based hashed list synchronization with ETag change detection for efficient polling
  - Lock-free HashStore using atomic pointer swaps for zero-blocking reads during updates
  - Configurable LRU cache for high-performance address lookups (default: 10k entries)
  - Privacy-preserving design: addresses are never stored or transmitted in plaintext (SHA256 with salt)
  - Forward-compatible hash list JSON format with `hashing_scheme` metadata field
  - Configurable S3 download settings (part size, concurrency, retries)

### Configuration
  - Add `--execution.address-filter.enable` flag to enable/disable address filtering
  - Add `--execution.address-filter.poll-interval` flag to set the polling interval for the s3 syncer, e.g. 5m
  - Add `--execution.address-filter.cache-size` flag to set the LRU cache size for address lookups (default: 10000)
  - Add `--execution.address-filter.s3.*` group of flags to configure S3 access:
    - Add `--execution.address-filter.s3.bucket` flag to specify the S3 bucket name for the hashed address list
    - Add `--execution.address-filter.s3.region` flag to specify the AWS region
    - Add `--execution.address-filter.s3.object-key` flag to specify the S3 object key for the hashed address list
    - Add `--execution.address-filter.s3.access-key` flag to specify the AWS access key
    - Add `--execution.address-filter.s3.secret-key` flag to specify the AWS secret key
  - Add `--execution.address-filter.download.*` group of flags to configure S3 download settings:
    - Add `--execution.address-filter.download.part-size-mb` flag to set S3 multipart download part size in MB (default: 32)
    - Add `--execution.address-filter.download.concurrency` flag to set S3 multipart download concurrency (default: 10)
    - Add `--execution.address-filter.download.max-retries` flag to set maximum retries for S3 part body download (default: 5)
