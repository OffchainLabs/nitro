### Changed
- Address-filter S3 hash list now expects the upstream `latest.json` schema: a flat `hashes` array of hex strings (replacing `address_hashes: [{hash}]`) and `hashing_scheme: "sha256-stringinput"` (replacing `"Sha256"`). Extra top-level fields such as `extract_uuid` and `issued_at` are ignored.
