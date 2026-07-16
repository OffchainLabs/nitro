### Configuration
- Remove `execution.sequencer.read-from-tx-queue-timeout`; the sequencer drains a snapshot of the tx queues instead of waiting on them. Nodes still setting this flag will fail to start and should drop it from their config.

### Internal
- Simplify the sequencer queue drain loop: `drainQueueItems` snapshots each queue's length and drains exactly that many items, and `drainAndValidateQueueItems` additionally filters out the invalid ones before sequencing. Shutdown and inactive forwarding drain without validating and now also forward the txs parked in the nonce-failure cache.
