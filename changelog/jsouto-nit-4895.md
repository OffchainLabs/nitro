### Added
- Transaction feed websocket server that streams executed transactions and receipts to subscribers. Enabled via `--execution.transaction-feed.enable`.

### Configuration
- Add `--execution.transaction-feed.enable` enables the transaction feed websocket server (default `false`).
- Add `--execution.transaction-feed.addr` address to bind the transaction feed server (default: all interfaces).
- Add `--execution.transaction-feed.port` port for the transaction feed server (default `9646`).
- Add `--execution.transaction-feed.client-buf` per-client send buffer size (default `256`).
- Add `--execution.transaction-feed.broadcast-buf` broadcast channel buffer size (default `4096`).
- Add `--execution.transaction-feed.write-timeout` write timeout per client (default `2s`).
- Add `--execution.transaction-feed.ping-interval` websocket ping interval (default `30s`).
- Add `--execution.transaction-feed.handshake-timeout` websocket handshake timeout (default `5s`).
