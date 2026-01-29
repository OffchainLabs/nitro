### Changed
- Renamed `arbkeccak` wasm module to `arbcrypto`.
- For wasm compilation target, EC recovery is now expected to be provided by the external `arbcrypto` module. For JIT and arbitrator provers we inject Rust-based implementation.
