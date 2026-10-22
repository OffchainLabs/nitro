### Changed
- Default `--file-logging.max-backups` increased from 20 to 40.

### Fixed
- With file logging enabled, Go runtime panics and fatal errors are now also written to a `<log-file>.crash` file, not only to stderr.
