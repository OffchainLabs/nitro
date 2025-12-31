### Changed
- Replace static batch poster compression configuration with dynamic, backlog-based compression level system
- Deprecate `--node.batch-poster.compression-level` flag in favor of `--node.batch-poster.compression-levels`

### Added
- New `--node.batch-poster.compression-levels` configuration flag that accepts a JSON array of compression configurations based on batch backlog thresholds
- Support for defining compression level, recompression level, and backlog threshold combinations
- Validation rules to ensure compression levels don't increase with higher backlog thresholds
