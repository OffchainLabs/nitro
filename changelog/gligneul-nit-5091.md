### Configuration
- Add `execution.transaction-filtering.address-filter.s3.preallocate-memory` (default true) to preallocate and commit the hash-list memory at startup, engaging only when `s3.max-file-size-mb` is set.

### Changed
- The address filter now decodes the S3 hash list directly into preallocated, reused data structures committed at startup, so reloads perform no large allocation and cannot OOM the node.
