//! Periodic block production progress logging.
//!
//! Producers update shared per window counters and carry over gauges. The reporter atomically
//! takes and resets the counters every [`SYNC_PROGRESS_LOG_INTERVAL`], while gauges retain their
//! latest values across reporting windows

use std::{sync::Arc, time::Duration};

use parking_lot::Mutex;
use reth_primitives_traits::constants::gas_units::{format_gas, format_gas_throughput};
use tokio::time::{Instant, MissedTickBehavior, interval_at};
use tracing::info;

const SYNC_PROGRESS_LOG_INTERVAL: Duration = Duration::from_secs(10);

#[derive(Debug, Default)]
struct ProgressWindow {
    blocks: u64,
    transactions: u64,
    gas: u64,
    flushes: u64,
    flushed_blocks: u64,
    commit_ms_sum: u64,
    commit_ms_max: u64,
}

#[derive(Clone, Copy, Debug, Default)]
struct ProgressGauges {
    latest_block: u64,
    last_persisted: u64,
    dirty_pages_mb: u64,
    flush_interval_current: u64,
    chain_length_unflushed: u64,
}

#[derive(Debug, Default)]
struct ProgressState {
    window: ProgressWindow,
    gauges: ProgressGauges,
}

struct ProgressSnapshot {
    window: ProgressWindow,
    gauges: ProgressGauges,
}

#[derive(Debug)]
pub(crate) struct ProgressCounters {
    state: Mutex<ProgressState>,
}

impl ProgressCounters {
    pub(crate) fn new(initial_block: u64) -> Self {
        Self {
            state: Mutex::new(ProgressState {
                gauges: ProgressGauges {
                    latest_block: initial_block,
                    last_persisted: initial_block,
                    ..Default::default()
                },
                ..Default::default()
            }),
        }
    }

    pub(crate) fn record_block(&self, number: u64, transactions: u64, gas: u64) {
        let mut state = self.state.lock();
        state.gauges.latest_block = number;
        state.window.blocks += 1;
        state.window.transactions += transactions;
        state.window.gas += gas;
    }

    pub(crate) fn record_flush(
        &self,
        last_block: u64,
        flushed_blocks: u64,
        commit_latency_ms: u64,
        dirty_pages_mb: u64,
        flush_interval: u64,
        chain_length_unflushed: u64,
    ) {
        let mut state = self.state.lock();
        state.gauges.last_persisted = last_block;
        state.window.flushes += 1;
        state.window.flushed_blocks += flushed_blocks;
        state.window.commit_ms_sum += commit_latency_ms;
        state.window.commit_ms_max = state.window.commit_ms_max.max(commit_latency_ms);
        state.gauges.dirty_pages_mb = dirty_pages_mb;
        state.gauges.flush_interval_current = flush_interval;
        state.gauges.chain_length_unflushed = chain_length_unflushed;
    }

    fn take_snapshot(&self) -> ProgressSnapshot {
        let mut state = self.state.lock();
        ProgressSnapshot {
            window: std::mem::take(&mut state.window),
            gauges: state.gauges,
        }
    }
}

struct ProgressReporter {
    counters: Arc<ProgressCounters>,
    window_start: Instant,
}

impl ProgressReporter {
    fn new(counters: Arc<ProgressCounters>) -> Self {
        Self {
            counters,
            window_start: Instant::now(),
        }
    }

    fn log_window(&mut self) {
        let ProgressSnapshot { window, gauges } = self.counters.take_snapshot();
        let elapsed = self.window_start.elapsed();
        self.window_start = Instant::now();
        if window.blocks == 0 && window.flushes == 0 {
            return;
        }

        info!(
            target: "block_producer",
            latest = gauges.latest_block,
            blocks = window.blocks,
            transactions = window.transactions,
            blocks_per_second = %format!("{:.1}", window.blocks as f64 / elapsed.as_secs_f64()),
            gas_used = %format_gas(window.gas),
            gas_throughput = %format_gas_throughput(window.gas, elapsed),
            "Produced blocks"
        );
        if window.flushes > 0 {
            info!(
                target: "block_producer",
                last_block = gauges.last_persisted,
                flushes = window.flushes,
                persisted_blocks = window.flushed_blocks,
                avg_commit_ms = window.commit_ms_sum / window.flushes,
                max_commit_ms = window.commit_ms_max,
                dirty_pages_mb = gauges.dirty_pages_mb,
                flush_interval_current = gauges.flush_interval_current,
                chain_length_unflushed = gauges.chain_length_unflushed,
                "Persisted blocks"
            );
        }
    }
}

pub(crate) async fn run_progress_reporter(counters: Arc<ProgressCounters>) {
    let start = Instant::now() + SYNC_PROGRESS_LOG_INTERVAL;
    let mut interval = interval_at(start, SYNC_PROGRESS_LOG_INTERVAL);
    interval.set_missed_tick_behavior(MissedTickBehavior::Delay);

    let mut reporter = ProgressReporter::new(counters);
    loop {
        interval.tick().await;
        reporter.log_window();
    }
}

#[cfg(test)]
mod tests {
    use tokio::task::yield_now;

    use super::*;

    #[test]
    fn window_counters_reset_and_gauges_carry_over() {
        let counters = ProgressCounters::new(9);
        counters.record_block(10, 3, 100);
        counters.record_block(11, 2, 50);
        counters.record_flush(11, 2, 40, 7, 256, 5);
        counters.record_flush(11, 1, 10, 8, 256, 4);

        let ProgressSnapshot { window, gauges } = counters.take_snapshot();
        assert_eq!(window.blocks, 2);
        assert_eq!(window.transactions, 5);
        assert_eq!(window.gas, 150);
        assert_eq!(window.flushes, 2);
        assert_eq!(window.flushed_blocks, 3);
        assert_eq!(window.commit_ms_sum, 50);
        assert_eq!(window.commit_ms_max, 40);
        assert_eq!(gauges.latest_block, 11);
        assert_eq!(gauges.last_persisted, 11);
        assert_eq!(gauges.dirty_pages_mb, 8);
        assert_eq!(gauges.flush_interval_current, 256);
        assert_eq!(gauges.chain_length_unflushed, 4);

        let ProgressSnapshot { window, gauges } = counters.take_snapshot();
        assert_eq!(window.blocks, 0);
        assert_eq!(window.transactions, 0);
        assert_eq!(window.gas, 0);
        assert_eq!(window.flushes, 0);
        assert_eq!(window.commit_ms_max, 0);
        assert_eq!(gauges.latest_block, 11);
        assert_eq!(gauges.last_persisted, 11);
        assert_eq!(gauges.dirty_pages_mb, 8);
        assert_eq!(gauges.flush_interval_current, 256);
        assert_eq!(gauges.chain_length_unflushed, 4);
    }

    #[tokio::test(start_paused = true)]
    async fn reporter_takes_window_on_interval() {
        let counters = Arc::new(ProgressCounters::new(0));
        counters.record_block(1, 2, 21_000);
        let reporter = tokio::spawn(run_progress_reporter(counters.clone()));

        yield_now().await;
        tokio::time::advance(SYNC_PROGRESS_LOG_INTERVAL).await;
        yield_now().await;

        let state = counters.state.lock();
        assert_eq!(state.window.blocks, 0);
        assert_eq!(state.window.transactions, 0);
        assert_eq!(state.window.gas, 0);
        assert_eq!(state.gauges.latest_block, 1);
        drop(state);

        reporter.abort();
    }
}
