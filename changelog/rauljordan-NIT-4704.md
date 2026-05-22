### Added
- Unified replay binary (`cmd/unified-replay/`) combining MEL message extraction and block production into a single WASM-compilable program
- `GetEndParentChainBlockHash` host I/O opcode for MEL proving
- `melwavmio` package providing WASM imports and native stubs for MEL
- Extended `GlobalState` to 4 bytes32 slots with backward-compatible hashing
- Makefile targets `build-unified-replay-env` and `build-unified-wasm-bin`
