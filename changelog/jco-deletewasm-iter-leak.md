### Fixed
- Iterator leak in `deleteWasmEntries`: defer is now late-bound so the post-flush iterator is released.
