### Fixed
 - Use defer to release createBlocksMutex in sequencerWrapper to prevent deadlock on panic
