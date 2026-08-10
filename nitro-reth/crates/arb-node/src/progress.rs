use std::time::{Duration, Instant};

use reth_primitives_traits::constants::gas_units::{format_gas, format_gas_throughput};
use tokio::sync::mpsc::UnboundedReceiver;
use tracing::info;

const SYNC_PROGRESS_INTERVAL: Duration = Duration::from_secs(10);

pub(crate) enum ProducerEvent {
    BlockProduced {
        number: u64,
        transactions: u64,
        gas: u64,
    },
    BlocksPersisted {
        last_block: u64,
        flushed_blocks: u64,
        commit_latency_ms: u64,
        dirty_pages_mb: u64,
        flush_interval: u64,
        chain_len_unflushed: u64,
    },
}

#[derive(Default)]
struct ReporterWindow {
    blocks: u64,
    transactions: u64,
    gas: u64,
    flushes: u64,
    flushed_blocks: u64,
    commit_ms_sum: u64,
    commit_ms_max: u64,
}

struct ProgressReporter {
    latest_block: u64,
    last_persisted: u64,
    dirty_pages_mb: u64,
    flush_interval_current: u64,
    chain_len_unflushed: u64,
    window: ReporterWindow,
    window_start: Instant,
}

impl ProgressReporter {
    fn new(initial_block: u64) -> Self {
        Self {
            latest_block: initial_block,
            last_persisted: initial_block,
            dirty_pages_mb: 0,
            flush_interval_current: 0,
            chain_len_unflushed: 0,
            window: ReporterWindow::default(),
            window_start: Instant::now(),
        }
    }

    fn handle_event(&mut self, event: ProducerEvent) {
        match event {
            ProducerEvent::BlockProduced {
                number,
                transactions,
                gas,
            } => {
                self.latest_block = number;
                self.window.blocks += 1;
                self.window.transactions += transactions;
                self.window.gas += gas;
            }
            ProducerEvent::BlocksPersisted {
                last_block,
                flushed_blocks,
                commit_latency_ms,
                dirty_pages_mb,
                flush_interval,
                chain_len_unflushed,
            } => {
                self.last_persisted = last_block;
                self.window.flushes += 1;
                self.window.flushed_blocks += flushed_blocks;
                self.window.commit_ms_sum += commit_latency_ms;
                self.window.commit_ms_max = self.window.commit_ms_max.max(commit_latency_ms);
                self.dirty_pages_mb = dirty_pages_mb;
                self.flush_interval_current = flush_interval;
                self.chain_len_unflushed = chain_len_unflushed;
            }
        }
    }

    fn log_window(&mut self) {
        let elapsed = self.window_start.elapsed();
        info!(
            target: "block_producer",
            latest = self.latest_block,
            blocks = self.window.blocks,
            transactions = self.window.transactions,
            blocks_per_second = %format!("{:.1}", self.window.blocks as f64 / elapsed.as_secs_f64()),
            gas_used = %format_gas(self.window.gas),
            gas_throughput = %format_gas_throughput(self.window.gas, elapsed),
            "Produced blocks"
        );
        if self.window.flushes > 0 {
            info!(
                target: "block_producer",
                last_block = self.last_persisted,
                flushes = self.window.flushes,
                persisted_blocks = self.window.flushed_blocks,
                avg_commit_ms = self.window.commit_ms_sum / self.window.flushes,
                max_commit_ms = self.window.commit_ms_max,
                dirty_pages_mb = self.dirty_pages_mb,
                flush_interval_current = self.flush_interval_current,
                chain_len_unflushed = self.chain_len_unflushed,
                "Persisted blocks"
            );
        }
        self.window = ReporterWindow::default();
        self.window_start = Instant::now();
    }
}

pub(crate) async fn run_progress_reporter(
    mut events: UnboundedReceiver<ProducerEvent>,
    initial_block: u64,
) {
    let start = tokio::time::Instant::now() + SYNC_PROGRESS_INTERVAL;
    let mut interval = tokio::time::interval_at(start, SYNC_PROGRESS_INTERVAL);
    interval.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);

    let mut reporter = ProgressReporter::new(initial_block);
    loop {
        tokio::select! {
            event = events.recv() => match event {
                Some(event) => reporter.handle_event(event),
                None => {
                    if reporter.window.blocks > 0 || reporter.window.flushes > 0 {
                        reporter.log_window();
                    }
                    return;
                }
            },
            _ = interval.tick() => reporter.log_window(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn window_counters_reset_and_gauges_carry_over() {
        let mut reporter = ProgressReporter::new(9);
        reporter.handle_event(ProducerEvent::BlockProduced {
            number: 10,
            transactions: 3,
            gas: 100,
        });
        reporter.handle_event(ProducerEvent::BlockProduced {
            number: 11,
            transactions: 2,
            gas: 50,
        });
        reporter.handle_event(ProducerEvent::BlocksPersisted {
            last_block: 11,
            flushed_blocks: 2,
            commit_latency_ms: 40,
            dirty_pages_mb: 7,
            flush_interval: 256,
            chain_len_unflushed: 5,
        });
        reporter.handle_event(ProducerEvent::BlocksPersisted {
            last_block: 11,
            flushed_blocks: 1,
            commit_latency_ms: 10,
            dirty_pages_mb: 8,
            flush_interval: 256,
            chain_len_unflushed: 4,
        });

        assert_eq!(reporter.window.blocks, 2);
        assert_eq!(reporter.window.transactions, 5);
        assert_eq!(reporter.window.gas, 150);
        assert_eq!(reporter.window.flushes, 2);
        assert_eq!(reporter.window.flushed_blocks, 3);
        assert_eq!(reporter.window.commit_ms_sum, 50);
        assert_eq!(reporter.window.commit_ms_max, 40);

        reporter.log_window();

        assert_eq!(reporter.window.blocks, 0);
        assert_eq!(reporter.window.transactions, 0);
        assert_eq!(reporter.window.gas, 0);
        assert_eq!(reporter.window.flushes, 0);
        assert_eq!(reporter.window.commit_ms_max, 0);
        assert_eq!(reporter.latest_block, 11);
        assert_eq!(reporter.last_persisted, 11);
        assert_eq!(reporter.dirty_pages_mb, 8);
        assert_eq!(reporter.flush_interval_current, 256);
        assert_eq!(reporter.chain_len_unflushed, 4);
    }

    #[test]
    fn log_window_without_activity_does_not_panic() {
        let mut reporter = ProgressReporter::new(0);
        reporter.log_window();
        reporter.handle_event(ProducerEvent::BlockProduced {
            number: 1,
            transactions: 2,
            gas: 21_000,
        });
        reporter.log_window();
    }
}
