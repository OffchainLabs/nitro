### Added
- Sign outbound webhook payloads from `cmd/filtering-report` with an Ed25519 leaf certificate issued by an internal CA, so receivers verify the signature against the CA root and rotate keys transparently.
