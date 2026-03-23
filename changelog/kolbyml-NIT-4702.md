### Fixed
- Recover panics in `StopWaiterSafe` tracked goroutines and log the panic message so delayed-sequencer panics do not take down the node.
