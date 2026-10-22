### Ignored
- nitro-reth: MEL batch decompression uses the vendored C brotli (same decoder as Go) instead of the pure-Rust brotli crate; decompression failures now yield an empty batch (Go parity) instead of silently truncating
- nitro-reth: the pure-Rust brotli crate is removed from the workspace entirely (bench fixtures now use the C encoder) and cargo-deny bans it for first-party crates
