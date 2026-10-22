### Internal
- Added explicit `timeout-minutes` to every CI job so a hung job fails fast instead of running to GitHub's 6h default. Caps are derived from historical p95 durations with margin.
