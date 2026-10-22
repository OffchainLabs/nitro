### Internal
- Seeded the `arb-replay` crate (nitro-reth): a wasm binary that reaches brotli through `arbcompress` host imports resolved by the runner
- The `brotli` crate gained a `link` feature so consumers choose between linking an implementation and declaring wasm imports.
