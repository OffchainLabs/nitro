### Configuration
- Added `--conf.min-version` to set the minimum supported Nitro version.
- Added `--conf.max-version` to set the maximum supported Nitro version.
- Comparable Nitro versions use stable `vMAJOR.MINOR.PATCH`, `dev.N`, `dev.N.private.N`, `rc.N`, or `rc.N.private.N` release tags, with positive counters.
- Version checks and the version alerter only run for release tags following that convention; opaque tags without a canonical release core, consensus-tagged, branch, development, and local builds skip them.
- Version range errors take precedence over unknown configuration key errors.

### Changed
- Nitro binaries now report explicit build provenance: tagged releases as `tag+commit`, branch builds as `branch+commit`, and unstamped builds as `local+commit`, with `-modified` appended for dirty trees. This also changes the local devp2p and `web3_clientVersion` strings.
- `--version` output now includes the commit timestamp in compact UTC form (e.g. `v3.11.3-rc.1+abc1234-20260812T100000Z`).
- Git tags built by CI must be directly usable as Docker image tag prefixes.
