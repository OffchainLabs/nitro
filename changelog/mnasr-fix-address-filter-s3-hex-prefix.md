### Fixed
- Fix address filter S3 syncer failing to parse hash list JSON when salt or hash values use `0x`/`0X` hex prefix. Go's `encoding/hex.DecodeString` does not handle the prefix, so it is now stripped before decoding.
