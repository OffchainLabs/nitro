### Fixed
- Suppress spurious block validator errors during shutdown
- Fix `Reorg` guard rejecting valid `count == 1` (reorg to genesis)
- Harden blocks reexecutor with panic recovery for concurrent trie access races
