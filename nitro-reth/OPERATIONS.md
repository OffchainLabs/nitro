# Operations Guide

## Recommended Linux sysctl for sustained high-throughput sync

For nodes syncing at high block rates, the default Linux dirty-page settings can let
the OS writeback backlog grow into double-digit GBs before throttling kicks in,
which produces long stalls. Recommended overrides:

```
# /etc/sysctl.d/99-arbreth.conf
vm.dirty_ratio = 10
vm.dirty_background_ratio = 5
vm.dirty_writeback_centisecs = 3000
vm.dirty_expire_centisecs = 6000
```

Apply with `sudo sysctl --system`.

## Reading the producer's flush metrics

The producer logs a `block flush` event at `target=block_producer` per flush. Fields:

- `mdbx_commit_latency_ms` — duration of the MDBX commit. Healthy: under 1000 ms p99.
  Concerning: sustained > 2000 ms suggests disk or page-cache pressure.
- `dirty_pages_mb` — OS dirty page count sampled at flush. Healthy: under 200 MB.
  If this grows into the GBs, increase `vm.dirty_ratio` headroom or reduce
  `flush_interval_current`.
- `flush_interval_current` — adaptive flush interval (blocks per commit). The
  scheduler halves it when commit latency exceeds target and doubles when
  comfortable. Default bounds: 32-256 blocks.
- `chain_len_unflushed` — depth of the in-memory chain waiting for persistence.
  Bounded by `ARB_RETH_MAX_INFLIGHT` (default 512).

## MDBX configuration

MDBX tuning is exposed via reth's CLI. See reth documentation for:

- `--db.sync-mode` (default in deployment configs: `safe-no-sync`)
- `--db.growth-step`
- `--db.exclusive`
- `--db.page-size` (note: fixed at DB creation; requires fresh sync to change)

For sustained high-throughput sync, `safe-no-sync` is recommended (matches geth /
erigon production defaults). Crash recovery relies on chain finality.
