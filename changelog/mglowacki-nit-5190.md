### Configuration
- Added `--conf.min-version` to set the minimum supported Nitro version.
- Added `--conf.max-version` to set the maximum supported Nitro version.
- Version checks and the version alerter only run for tagged release builds; branch, development, and local builds skip them.
- Version range errors take precedence over unknown configuration key errors.

### Versioning
- Nitro binaries now report explicit build provenance: tagged releases as `tag+commit` (e.g. `v3.11.3-rc.1+abc1234`), branch builds as `branch+commit`, and unstamped builds as `local+commit`, with `-modified` appended for dirty trees. This also changes the local devp2p and `web3_clientVersion` strings.
- Release tags built by CI must be canonical SemVer (`vMAJOR.MINOR.PATCH` with optional prerelease); CI fails before building otherwise.
- Manual builders must replace the `NITRO_VERSION` build input with `NITRO_TAG`, `NITRO_BRANCH`, and `NITRO_COMMIT`.
