### Added
- Feed clients can fill gaps from the feed's REST backlog instead of the websocket backlog.
- Add `arb/feed/backfill/runs`.
- Add `arb/feed/backfill/chunks`.
- Add `arb/feed/backfill/messages`.
- Add `arb/feed/backfill/chunks_missing`.
- Add `arb/feed/backfill/errors`.
- Add `arb/feed/backfill/permanent_errors`.

### Configuration
- Add `node.feed.input.rest.enable` (default false).
- Add `node.feed.input.rest.url`, required when `node.feed.input.rest.enable` is set.
- Add `node.feed.input.rest.timeout` (default 10s).
