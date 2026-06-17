### Added
- Active sequencer now periodically reports the current address-filter set ID to the filtering-report service, which forwards it to a configured external HTTP endpoint

### Configuration
- `cmd/filtering-report`: the request signer is now configured via top-level `--signer.*` flags instead of `--report-forwarder.signer.*`, since the signer is shared by the report forwarder and the filter-set-id reporter.
