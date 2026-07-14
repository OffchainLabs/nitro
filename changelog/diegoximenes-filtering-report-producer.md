### Changed
- Filtered-tx reports are now logged node-side when reported, tagged with the producer (`prechecker`/`sequencer`), with unbounded fields (raw transaction and event log data) truncated to keep entries compact.
- `cmd/filtering-report` forwarder no longer logs the full report body; its forward success/failure logs now include only the report id and tx hash.
