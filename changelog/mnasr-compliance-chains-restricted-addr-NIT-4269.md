### Added
- Add address filter service for compliance chains (`addressfilter` package). This feature enables sequencers to block transactions involving filtered addresses by polling a hashed address list from S3. Key capabilities include:
  - S3-based hashed list synchronization with ETag change detection for efficient polling
  - Lock-free HashStore using atomic pointer swaps for zero-blocking reads during updates
  - LRU cache (10k entries) for high-performance address lookups
  - Privacy-preserving design: addresses are never stored or transmitted in plaintext (SHA256 with salt)
  - Configurable via `--execution.address-filter.*` flags (enable,s3.bucket, s3.region, s3.object-key, s3.AccessKey, s3.SecretKey, poll-interval)

### Configuration
  - Add `--execution.address-filter.enable` flag to enable/disable address filtering 
  - Add `--execution.address-filter.poll-interval` flag to set the polling interval for the s3 syncer , e.g. 5s
  - Add `--execution.address-filter.s3.*` group of flags to configure S3 access:
    - Add `--execution.address-filter.s3.bucket` flag to specify the S3 bucket name for the hashed address list
    - Add `--execution.address-filter.s3.region` flag to specify the AWS region of
    - Add `--execution.address-filter.s3.object-key` flag to specify the S3 object key for the hashed address list
    - Add `--execution.address-filter.s3.access-key` flag to specify the AWS access
    - Add `--execution.address-filter.s3.secret-key` flag to specify the AWS secret key
