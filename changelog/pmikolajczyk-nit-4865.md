### Fixed
- Fix ECRECOVER divergence between native execution and the WASM prover: k256 rejected high-S signatures in `recover_from_prehash` via its low-S canonicality check, while go-ethereum's ECRECOVER precompile accepts them. The fix normalizes `s → N−s` and flips the recovery point's y-parity before calling k256, recovering the identical public key without triggering the rejection.
