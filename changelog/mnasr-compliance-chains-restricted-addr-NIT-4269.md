### Added
- Add restricted address filtering for compliance chains (`restrictedaddr` package). This feature enables sequencers to block transactions involving restricted addresses by polling a hashed address list from S3. Key capabilities include:
  - S3-based hash list synchronization with ETag change detection for efficient polling
  - Lock-free HashStore using atomic pointer swaps for zero-blocking reads during updates
  - LRU cache (10k entries) for high-performance address lookups
  - Privacy-preserving design: addresses are never stored or transmitted in plaintext (SHA256 with salt)
  - Configurable via `--node.restricted-addr.*` flags (bucket, region, object-key, poll-interval)
