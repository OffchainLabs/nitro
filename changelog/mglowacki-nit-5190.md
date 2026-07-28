### Configuration
- New `--conf.min-version` and `--conf.max-version` (both empty by default,
  hot-reloadable): a configuration can declare the inclusive range of nitro
  versions it supports, and the node exits at startup if it falls outside it.
  Intended for configurations distributed to operators, so that using one
  written for a different nitro version reports `unsupported nitro version`
  rather than an opaque "invalid keys" parse failure. The bounds are evaluated
  before any other configuration field, so the version error always wins.
  Bounds must be full semantic versions (e.g. `v3.9.0`); malformed bounds are
  rejected rather than silently ignored.

  The check applies only to builds produced from a semantic version tag. Local
  builds, test binaries, and untagged CI builds report a note on stderr and are
  not gated. The commit hash the build appends to the version tag and the
  `-modified` marker of a dirty tree are ignored when comparing, so a
  `v3.9.9-<hash>` release build satisfies `min-version: v3.9.9`. Release
  candidates keep their semantic-version precedence and therefore rank below
  the release they lead up to: `v3.9.9-rc.2` does *not* satisfy
  `min-version: v3.9.9`. To admit every prerelease of a version, write
  `min-version: v3.9.9-0`.
