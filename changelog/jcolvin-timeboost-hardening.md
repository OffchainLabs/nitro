### Fixed
- Hardened timeboost auctioneer against races, silent failures, and shutdown leaks
- Fixed `durationIntoRound` sub-second precision loss and unit mismatch
- Fixed rate-limit rollback so failed bids no longer consume rate-limit slots
- Fixed bid cache keying from `ExpressLaneController` to `Bidder` to prevent bid overwrites
### Changed
- Renamed `BidFloorAgent` to `ReserveOriginator` for clarity
- Replaced unbounded goroutine-per-bid persistence with bounded channel and dedicated worker
