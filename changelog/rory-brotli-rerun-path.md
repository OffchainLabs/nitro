### Fixed
- Fixed the brotli build script watching a nonexistent path, which made every cargo build/check rebuild all crates above `brotli` (severe in `nitro-reth` since path-based deps landed in #598)
