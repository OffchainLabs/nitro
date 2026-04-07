### Fixed
- Fix deadlock in `StopWaiterSafe.stopAndWaitImpl` by releasing `RLock` before blocking on `waitChan`.
