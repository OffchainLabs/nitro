### Changed
- Replace static batch poster compression configuration with dynamic, backlog-based compression level system

### Deprecated
- Deprecate `--node.batch-poster.compression-level` flag in favor of `--node.batch-poster.compression-levels`

### Added
- New `--node.batch-poster.compression-levels` configuration flag that accepts a JSON array of compression configurations based on batch backlog thresholds
- Support for defining compression level, recompression level, and backlog threshold combinations
- Validation rules to ensure compression levels don't increase with higher backlog thresholds

### Configuration
- The new `--node.batch-poster.compression-levels` flag allows operators to specify different compression strategies based 
on the current backlog of batches to be posted. The configuration is provided as a JSON array of objects, each containing:
  - `backlog`: The minimum backlog size (in number of batches) at which this configuration applies. First entry must be zero.
  - `level`: The initial compression level applied to messages when they are added to a batch once the backlog reaches or exceeds the configured threshold.
  - `recompression-level`: The recompression level to use for already compressed batches when the backlog meets or exceeds the threshold.
  - Example configuration:
    ```json 
    [ { "backlog": 0, "level": 3, "recompression-level": 5 }, { "backlogThreshold": 10, "level": 5, "recompression-level": 7 }, { "backlogThreshold": 20, "level": 7, "recompression-level": 9 } ]
    ```
- Validation rules:
  - The `backlog` values must be in strictly ascending order.
  - Both level and recompression-level must be weakly descending (non-increasing) across entries 
  - recompression-level must be greater than or equal to level within each entry (recompression should be at least as good as initial compression)
  - All levels must be in valid range: 0-11