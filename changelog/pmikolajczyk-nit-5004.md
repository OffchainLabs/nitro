### Changed
- Replace `sha3`/`digest` keccak with `tiny-keccak` across `prover` and `caller-env`.

### Internal
- Make `GuestPtr` field private with checked addition (panics on overflow instead of silent wraparound).
- Remove JIT wrappers for arbrypto and arbcompress
