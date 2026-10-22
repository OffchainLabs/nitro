### Internal
- `make` now installs safe-smart-account's dependencies with `npm ci` instead of `npm install`, so the submodule's `package-lock.json` can never be rewritten as a build side effect; a manifest/lockfile mismatch now fails loudly instead.
