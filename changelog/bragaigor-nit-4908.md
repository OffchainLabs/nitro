### Changed
- Replaced bincode-based Stylus module cache with a stable positional WAVM wire format owned by the prover.
- Existing nodes have their cached entries purged on first start; a missing version key is treated as incompatible.

### Added
- Added `WavmSerializeVersion = 1` (`uint32` in Go, `u32` in Rust); the two consts are guarded against drift by a CI test.
