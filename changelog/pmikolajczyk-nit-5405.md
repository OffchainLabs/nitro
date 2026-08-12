### Ignored
- nitro-reth: MEL batch decompression uses the vendored C brotli (same decoder as Go) instead of the pure-Rust brotli crate; decompression failures now yield an empty batch (Go parity) instead of silently truncating
