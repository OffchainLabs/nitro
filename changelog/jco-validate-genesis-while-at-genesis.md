### Fixed
- Re-run genesis assertion validation on startup while the chain head is still at genesis, so a previously failed validation is not silently skipped on subsequent restarts.
