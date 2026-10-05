### Fixed
- Bumped `WasmerSerializeVersion` to 17 to match wasmer's `MetadataHeader::CURRENT_VERSION`

### Ignored
- added a test that fails if the Go `WasmerSerializeVersion` diverges from `MetadataHeader::CURRENT_VERSION`
