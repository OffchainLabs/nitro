### Fixed
- Speed up JSON-RPC request decoding, which stalled block production during large request bursts

### Changed
- JSON-RPC methods now reject `null` for a required argument, such as the transaction in `eth_call`
