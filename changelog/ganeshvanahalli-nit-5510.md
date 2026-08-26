### Fixed
- Fixed flaky `TestTimeboostRedisCoordinator`: the test slept 10ms before reading back updates the redis coordinator applies asynchronously, so a loaded CI runner could produce a stale read. It now polls for each update to land.
