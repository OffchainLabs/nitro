### Added
- Add address filter service for compliance chains (`addressfilter` package). This feature enables sequencers to block transactions involving filtered addresses by polling a hashed address list from S3. Key capabilities include:
  - S3-based hashed list synchronization with ETag change detection for efficient polling
  - Lock-free HashStore using atomic pointer swaps for zero-blocking reads during updates
  - LRU cache (10k entries) for high-performance address lookups
  - Privacy-preserving design: addresses are never stored or transmitted in plaintext (SHA256 with salt)
  - Configurable via `--execution.address-filter.*` flags (enable,s3.bucket, s3.region, s3.object-key, s3.AccessKey, s3.SecretKey, poll-interval)
