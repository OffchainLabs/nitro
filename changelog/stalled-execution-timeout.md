### Added
- Add `--node.transaction-streamer.stalled-execution-timeout` flag: when set to a non-zero duration, shuts down gracefully if block execution repeatedly fails on the same message index, preventing the node from hanging indefinitely as a zombie process on unrecoverable state or trie errors.
- Escalate `ExecuteNextMsg` execution engine errors to error level when execution remains stalled on the same message index for more than one minute.
