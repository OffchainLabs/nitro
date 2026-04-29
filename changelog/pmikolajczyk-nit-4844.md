### Fixed
- Wasmer now enforces the Stylus heap bound as a hard memory maximum, preventing `memory.grow(1<<16)` from succeeding in the JIT when it would fail in the prover
