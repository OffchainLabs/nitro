### Added
 - Update geth pin to include commits from upstream geth's v1.17.2

### Configuration
- Rename `--persistent.pebble.experimental.mem-table-stop-writes-threshold` to `--persistent.pebble.experimental.mem-table-number`, following the upstream geth config rename. The option still determines the memtable size (cache/2/value, unchanged), but pebble's hard limit on queued memtables is now derived as twice the configured value instead of the value itself, to accommodate temporary spikes. Update any config files or scripts that set the old flag, as unknown flags prevent node startup.
- Change the default of `--persistent.pebble.experimental.read-sampling-multiplier` for nodes using the path state scheme from `-1` to `1`, following upstream geth. This re-enables read-triggered compactions on path-scheme nodes; hash-scheme nodes keep the previous default of `-1`. Set the flag explicitly to `-1` to restore the previous behavior.
