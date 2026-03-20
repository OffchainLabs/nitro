### Fixed
- Fix nil-dereference and log format in `cmd/nitro/nitro.go` when machine locator creation fails; return early instead of falling through to dereference nil locator
