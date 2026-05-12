### Changed
- Replaced bincode-based Stylus module cache with a stable positional WAVM wire format owned by the prover.
- Existing nodes have their cached entries purged on first start; a missing version key is treated as incompatible.

### Added
- Added `WavmSerializeVersion = 1` (Go `uint32`, Rust `u8`); the two consts are guarded against drift by a CI test.
