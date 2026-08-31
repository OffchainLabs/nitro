### Internal
- `arb-stylus` implements `arbutil`'s `EvmApi` trait instead of a local near-copy; WASM page accounting moved from the instance env into the API side (`PageTracker`), matching the upstream shape
- Fixed `arbutil` typos: `CreateRespone`/`Succes` → `CreateResponse`/`Success`
