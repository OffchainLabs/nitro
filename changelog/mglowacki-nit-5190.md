### Configuration
- Added `--conf.min-version` to set the minimum supported Nitro version.
- Added `--conf.max-version` to set the maximum supported Nitro version.
- Version checks and the version alerter only run for canonical SemVer-tagged release builds; consensus-tagged, branch, development, and local builds skip them.
- Version range errors take precedence over unknown configuration key errors.

### Changed
- Nitro binaries now report explicit build provenance: tagged releases as `tag+commit`, branch builds as `branch+commit`, and unstamped builds as `local+commit`, with `-modified` appended for dirty trees. This also changes the local devp2p and `web3_clientVersion` strings.
- `--version` output now includes the commit timestamp in compact UTC form (e.g. `v3.11.3-rc.1+abc1234-20260812T100000Z`).
- Git tags built by CI must be directly usable as Docker image tag prefixes.
