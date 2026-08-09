### Changed
- Simplify Docker CI workflow: remove the local registry service, drop image targets not exercised by CI (nitro-node, nitro-node-slim-stripped, nitro-node-stripped), and load nitro-node-dev directly instead of pushing to and pulling from localhost:5000
- Remove the Docker layer cache (actions/cache tarballs and the associated move/clear steps); a cold build is faster than the cache export overhead
- Speed up detect-changes by skipping submodule init; the contracts submodule is matched as a bare gitlink path
