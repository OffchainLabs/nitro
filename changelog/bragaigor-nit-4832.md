### Fixed
- Reduce log spam during node sync. `ExecuteNextMsg` now throttles `accumulator not found` errors via an ephemeral error handler.
