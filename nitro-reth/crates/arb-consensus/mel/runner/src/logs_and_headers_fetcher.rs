//! The logs-and-headers prefetcher backing `arb-mel`'s sync `LogsFetcher`.
//!
//! Ports nitro's `arbnode/mel/runner/logs_and_headers_fetcher.go`.
//! [`LogsAndHeadersFetcher::fetch`] bulk-fetches, for a block range, the logs
//! extraction needs and that range's headers; the sync [`LogsFetcher`] impl then
//! serves logs from cache, and [`LogsAndHeadersFetcher::get_header_by_number`]
//! serves headers from cache (falling back to the reader).

use std::{collections::HashMap, sync::Arc};

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::{Address, B256};
use alloy_rpc_types_eth::{Filter, Header, Log};
use alloy_sol_types::SolEvent;
use arb_mel::{
    InboxMessageDelivered, InboxMessageDeliveredFromOrigin, LogsFetcher, MELConfigSet, MelError,
    MelResult, MelState, MessageDelivered, SequencerBatchData, SequencerBatchDelivered,
};
use arb_parent_chain_client::ParentChainReader;

use crate::{MelRunnerError, Result};

/// Prefetches the parent-chain logs extraction needs for a block range and serves
/// them to the sync [`LogsFetcher`].
pub(crate) struct LogsAndHeadersFetcher<P: ParentChainReader> {
    parent_chain_reader: Arc<P>,
    blocks_to_fetch: u64,
    rollup_addr: Address,
    from_block: u64,
    to_block: u64,
    chain_height: u64,
    logs_by_block_hash: HashMap<B256, Vec<Log>>,
    logs_by_tx_index: HashMap<B256, HashMap<u64, Vec<Log>>>,
    /// Headers for `[from_block, to_block]`, one entry per block (`None` if the
    /// batch didn't return it). Indexed by `number - from_block`.
    headers: Vec<Option<Header>>,
}

impl<P: ParentChainReader> LogsAndHeadersFetcher<P> {
    /// Creates an empty prefetcher.
    pub(crate) fn new(
        parent_chain_reader: Arc<P>,
        blocks_to_fetch: u64,
        rollup_addr: Address,
    ) -> Self {
        Self {
            parent_chain_reader,
            blocks_to_fetch,
            rollup_addr,
            from_block: 0,
            to_block: 0,
            chain_height: 0,
            logs_by_block_hash: HashMap::new(),
            logs_by_tx_index: HashMap::new(),
            headers: Vec::new(),
        }
    }

    /// The parent-chain header at `number`, served from the prefetched range
    /// cache when present, otherwise fetched and written back into its slot.
    /// Mirrors nitro's `getHeaderByNumber`.
    pub(crate) async fn get_header_by_number(&mut self, number: u64) -> Result<Option<Header>> {
        // `None` unless the cache holds one slot per block of the range and
        // covers `number`; a cache out of step with the range would otherwise
        // serve another block's header.
        let range_len = self.to_block.saturating_sub(self.from_block) + 1;
        let slot = (self.headers.len() as u64 == range_len
            && (self.from_block..=self.to_block).contains(&number))
        .then(|| (number - self.from_block) as usize);

        if let Some(pos) = slot
            && let Some(header) = &self.headers[pos]
        {
            return Ok(Some(header.clone()));
        }

        let header = self
            .parent_chain_reader
            .header_by_number(BlockNumberOrTag::Number(number))
            .await?;

        // `None` means not posted yet; the caller retries and wants a fresh read.
        if let (Some(pos), Some(header)) = (slot, &header) {
            self.headers[pos] = Some(header.clone());
        }
        Ok(header)
    }

    /// Prefetches the logs for the next block range, unless it is already cached.
    /// Mirrors nitro's `fetch`, which splits headers from logs; here all five
    /// range reads are issued at once, since they're merged in a fixed order.
    pub(crate) async fn fetch(&mut self, pre_state: &MelState) -> Result<()> {
        let next = pre_state.parent_chain_block_number + 1;
        if next <= self.to_block {
            return Ok(());
        }
        self.reset();
        let mut to = next + self.blocks_to_fetch;
        if to > self.chain_height {
            let head = self
                .parent_chain_reader
                .header_by_number(BlockNumberOrTag::Latest)
                .await?
                .ok_or_else(|| MelRunnerError::NotFound("parent chain head".to_string()))?;
            if head.inner.number < next {
                return Err(MelRunnerError::InvalidState(
                    "reorg detected inside logs-and-headers fetcher".to_string(),
                ));
            }
            self.chain_height = head.inner.number;
            to = self.chain_height.min(to);
        }

        // Sequencer batch delivery + data (no address filter, like nitro).
        let seq_batch_filter = Filter::new()
            .from_block(next)
            .to_block(to)
            .event_signature(vec![
                SequencerBatchDelivered::SIGNATURE_HASH,
                SequencerBatchData::SIGNATURE_HASH,
            ]);
        // Delayed messages: `MessageDelivered` at the delayed-inbox target...
        let delayed_delivered_filter = Filter::new()
            .from_block(next)
            .to_block(to)
            .address(pre_state.delayed_message_posting_target_address)
            .event_signature(MessageDelivered::SIGNATURE_HASH);
        // ...and the inbox-message events (any address).
        let delayed_inbox_filter =
            Filter::new()
                .from_block(next)
                .to_block(to)
                .event_signature(vec![
                    InboxMessageDelivered::SIGNATURE_HASH,
                    InboxMessageDeliveredFromOrigin::SIGNATURE_HASH,
                ]);
        let mel_config_filter = Filter::new()
            .from_block(next)
            .to_block(to)
            .address(self.rollup_addr)
            .event_signature(MELConfigSet::SIGNATURE_HASH);

        let reader = &self.parent_chain_reader;
        let rollup_addr = self.rollup_addr;
        // Results stay in locals until all five succeed, so an error leaves the
        // fetcher in its reset (empty) state.
        let (headers, seq_batch_logs, delayed_delivered, delayed_inbox, mel_config) = tokio::try_join!(
            reader.headers_by_number_range(next, to),
            reader.filter_logs(&seq_batch_filter),
            reader.filter_logs(&delayed_delivered_filter),
            reader.filter_logs(&delayed_inbox_filter),
            // MEL config updates, only when a rollup address is configured.
            async {
                if rollup_addr != Address::ZERO {
                    reader.filter_logs(&mel_config_filter).await
                } else {
                    Ok(Vec::new())
                }
            },
        )?;

        // All fallible fetches succeeded; populate the caches together.
        self.headers = headers;
        // Index every fetched log by block hash and by tx index.
        for log in seq_batch_logs
            .into_iter()
            .chain(delayed_delivered)
            .chain(delayed_inbox)
            .chain(mel_config)
        {
            let (Some(block_hash), Some(tx_index)) = (log.block_hash, log.transaction_index) else {
                continue;
            };
            self.logs_by_block_hash
                .entry(block_hash)
                .or_default()
                .push(log.clone());
            self.logs_by_tx_index
                .entry(block_hash)
                .or_default()
                .entry(tx_index)
                .or_default()
                .push(log);
        }

        self.from_block = next;
        self.to_block = to;
        Ok(())
    }

    fn reset(&mut self) {
        self.from_block = 0;
        self.to_block = 0;
        self.logs_by_block_hash.clear();
        self.logs_by_tx_index.clear();
        self.headers.clear();
    }
}

impl<P: ParentChainReader> LogsFetcher for LogsAndHeadersFetcher<P> {
    fn logs_for_block_hash(&self, block_hash: B256) -> MelResult<Vec<Log>> {
        Ok(self
            .logs_by_block_hash
            .get(&block_hash)
            .cloned()
            .unwrap_or_default())
    }

    fn logs_for_tx_index(&self, block_hash: B256, tx_index: u64) -> MelResult<Vec<Log>> {
        self.logs_by_tx_index
            .get(&block_hash)
            .and_then(|by_index| by_index.get(&tx_index))
            .cloned()
            .ok_or(MelError::SequencerBatchData(
                "logs for tx index not in prefetch cache",
            ))
    }
}

#[cfg(test)]
mod tests {
    use std::sync::atomic::{AtomicUsize, Ordering};

    use alloy_primitives::{Address, B256, Bytes, LogData};
    use alloy_rpc_types_eth::{Block, Transaction, TransactionReceipt};
    use arb_parent_chain_client::{
        MockParentChainReader, Result as ParentChainResult, test_utils::header,
    };

    use super::*;

    fn mk_log(topic0: B256, block: u64, block_hash: B256, tx_index: u64) -> Log {
        Log {
            inner: alloy_primitives::Log {
                address: Address::ZERO,
                data: LogData::new_unchecked(vec![topic0], Bytes::new()),
            },
            block_hash: Some(block_hash),
            block_number: Some(block),
            block_timestamp: None,
            transaction_hash: Some(B256::repeat_byte(0xee)),
            transaction_index: Some(tx_index),
            log_index: None,
            removed: false,
        }
    }

    fn mk_log_at(
        topic0: B256,
        address: Address,
        block: u64,
        block_hash: B256,
        tx_index: u64,
    ) -> Log {
        let mut log = mk_log(topic0, block, block_hash, tx_index);
        log.inner.address = address;
        log
    }

    #[tokio::test]
    async fn prefetches_and_indexes_logs() {
        let batch_bh = B256::repeat_byte(0x1a);
        let delayed_bh = B256::repeat_byte(0x2b);
        let delayed_target = Address::repeat_byte(0xcc);
        let ignored = B256::repeat_byte(0xff);

        let mut mock = MockParentChainReader::new();
        mock.with_log(mk_log(
            SequencerBatchDelivered::SIGNATURE_HASH,
            5,
            batch_bh,
            1,
        ))
        .with_log(mk_log(SequencerBatchData::SIGNATURE_HASH, 5, batch_bh, 1))
        .with_log(mk_log(ignored, 5, batch_bh, 1))
        .with_log(mk_log_at(
            MessageDelivered::SIGNATURE_HASH,
            delayed_target,
            6,
            delayed_bh,
            2,
        ))
        .with_log(mk_log(
            InboxMessageDeliveredFromOrigin::SIGNATURE_HASH,
            6,
            delayed_bh,
            2,
        ))
        .with_log(mk_log(
            InboxMessageDelivered::SIGNATURE_HASH,
            6,
            delayed_bh,
            2,
        ))
        .with_log(mk_log(ignored, 6, delayed_bh, 2));

        let mut fetcher = LogsAndHeadersFetcher::new(Arc::new(mock), 10, Address::ZERO);
        fetcher.chain_height = 100; // skip the head lookup, as nitro's test does

        let state = MelState {
            parent_chain_block_number: 1,
            delayed_message_posting_target_address: delayed_target,
            ..Default::default()
        };
        fetcher.fetch(&state).await.unwrap();

        // Unrelated ("ignored") logs are filtered out by topic.
        assert_eq!(fetcher.logs_by_block_hash.len(), 2);
        assert_eq!(fetcher.logs_by_block_hash[&batch_bh].len(), 2);
        assert_eq!(fetcher.logs_by_block_hash[&delayed_bh].len(), 3);
        assert_eq!(fetcher.logs_by_tx_index[&batch_bh][&1].len(), 2);
        assert_eq!(fetcher.logs_by_tx_index[&delayed_bh][&2].len(), 3);

        // The sync LogsFetcher serves the same cache.
        assert_eq!(fetcher.logs_for_block_hash(batch_bh).unwrap().len(), 2);
        assert_eq!(fetcher.logs_for_tx_index(batch_bh, 1).unwrap().len(), 2);
    }

    #[tokio::test]
    async fn get_header_by_number_forwards_to_reader() {
        let mut mock = MockParentChainReader::new();
        mock.with_header(header(9, B256::repeat_byte(0x09)));
        let mut fetcher = LogsAndHeadersFetcher::new(Arc::new(mock), 10, Address::ZERO);

        let got = fetcher.get_header_by_number(9).await.unwrap();
        assert_eq!(got.unwrap().inner.number, 9);
        assert!(fetcher.get_header_by_number(999).await.unwrap().is_none());
    }

    #[tokio::test]
    async fn serves_headers_from_prefetch_cache() {
        let h5 = header(5, B256::repeat_byte(0x05));
        let mut mock = MockParentChainReader::new();
        mock.with_header(h5.clone());
        let mut fetcher = LogsAndHeadersFetcher::new(Arc::new(mock), 10, Address::ZERO);
        fetcher.chain_height = 100; // skip the head lookup

        // Prefetch range [2, 12]; header 5 is present, the rest missing.
        let state = MelState {
            parent_chain_block_number: 1,
            ..Default::default()
        };
        fetcher.fetch(&state).await.unwrap();

        // Header 5 comes from the cache — proven by erroring the reader first.
        fetcher.parent_chain_reader.set_error(Some("boom"));
        assert_eq!(fetcher.get_header_by_number(5).await.unwrap(), Some(h5));
        // A block outside the cached range forwards and hits the erroring reader.
        assert!(fetcher.get_header_by_number(50).await.is_err());
    }

    /// A [`ParentChainReader`] for asserting *how* the prefetcher reads.
    ///
    /// Its range batch always comes up empty, so header lookups fall through to
    /// `header_by_number`, which is counted. Gated reads yield once, so peak
    /// `in_flight` is the number issued concurrently (1 if awaited one by one).
    /// Everything else delegates to an empty [`MockParentChainReader`].
    #[derive(Default)]
    struct ProbeReader {
        inner: MockParentChainReader,
        in_flight: AtomicUsize,
        max_in_flight: AtomicUsize,
        single_header_reads: AtomicUsize,
    }

    impl ProbeReader {
        fn with_headers(headers: impl IntoIterator<Item = Header>) -> Self {
            let mut inner = MockParentChainReader::new();
            for header in headers {
                inner.with_header(header);
            }
            Self {
                inner,
                ..Default::default()
            }
        }

        async fn gate(&self) {
            let now = self.in_flight.fetch_add(1, Ordering::SeqCst) + 1;
            self.max_in_flight.fetch_max(now, Ordering::SeqCst);
            tokio::task::yield_now().await;
            self.in_flight.fetch_sub(1, Ordering::SeqCst);
        }
    }

    #[async_trait::async_trait]
    impl ParentChainReader for ProbeReader {
        async fn headers_by_number_range(
            &self,
            from: u64,
            to: u64,
        ) -> ParentChainResult<Vec<Option<Header>>> {
            self.gate().await;
            Ok(vec![None; (to - from + 1) as usize])
        }

        async fn filter_logs(&self, _q: &Filter) -> ParentChainResult<Vec<Log>> {
            self.gate().await;
            Ok(Vec::new())
        }

        async fn header_by_number(
            &self,
            num: BlockNumberOrTag,
        ) -> ParentChainResult<Option<Header>> {
            self.single_header_reads.fetch_add(1, Ordering::SeqCst);
            self.inner.header_by_number(num).await
        }
        async fn header_by_hash(&self, hash: B256) -> ParentChainResult<Option<Header>> {
            self.inner.header_by_hash(hash).await
        }
        async fn block_by_number(&self, num: BlockNumberOrTag) -> ParentChainResult<Option<Block>> {
            self.inner.block_by_number(num).await
        }
        async fn block_by_hash(&self, hash: B256) -> ParentChainResult<Option<Block>> {
            self.inner.block_by_hash(hash).await
        }
        async fn transaction_in_block(
            &self,
            block: B256,
            index: u64,
        ) -> ParentChainResult<Option<Transaction>> {
            self.inner.transaction_in_block(block, index).await
        }
        async fn transaction_receipt(
            &self,
            tx: B256,
        ) -> ParentChainResult<Option<TransactionReceipt>> {
            self.inner.transaction_receipt(tx).await
        }
        async fn transaction_by_hash(&self, hash: B256) -> ParentChainResult<Option<Transaction>> {
            self.inner.transaction_by_hash(hash).await
        }
    }

    #[tokio::test]
    async fn range_reads_are_issued_concurrently() {
        let probe = Arc::new(ProbeReader::default());
        // A non-zero rollup address, so the MEL-config query is issued too.
        let mut fetcher = LogsAndHeadersFetcher::new(probe.clone(), 10, Address::repeat_byte(0xaa));
        fetcher.chain_height = 100; // skip the head lookup

        let state = MelState {
            parent_chain_block_number: 1,
            ..Default::default()
        };
        fetcher.fetch(&state).await.unwrap();

        // The header batch plus the four log filters, all in flight at once.
        assert_eq!(probe.max_in_flight.load(Ordering::SeqCst), 5);
    }

    #[tokio::test]
    async fn header_missed_by_the_range_batch_is_cached() {
        let h5 = header(5, B256::repeat_byte(0x05));
        let probe = Arc::new(ProbeReader::with_headers([h5.clone()]));
        let mut fetcher = LogsAndHeadersFetcher::new(probe.clone(), 10, Address::ZERO);
        fetcher.chain_height = 100; // skip the head lookup

        let state = MelState {
            parent_chain_block_number: 1,
            ..Default::default()
        };
        fetcher.fetch(&state).await.unwrap();

        // The first lookup misses the empty slot and falls through to the
        // reader; the write-back serves the second.
        for _ in 0..2 {
            assert_eq!(
                fetcher.get_header_by_number(5).await.unwrap(),
                Some(h5.clone())
            );
        }
        assert_eq!(probe.single_header_reads.load(Ordering::SeqCst), 1);

        // A block that isn't posted yet must not be cached as absent.
        for _ in 0..2 {
            assert!(fetcher.get_header_by_number(6).await.unwrap().is_none());
        }
        assert_eq!(probe.single_header_reads.load(Ordering::SeqCst), 3);
    }

    #[tokio::test]
    async fn header_cache_out_of_step_with_range_falls_back() {
        let h5 = header(5, B256::repeat_byte(0x05));
        let mut mock = MockParentChainReader::new();
        mock.with_header(h5.clone());
        let mut fetcher = LogsAndHeadersFetcher::new(Arc::new(mock), 10, Address::ZERO);

        // Three cached headers against a one-block range: slot 0 must not be
        // read as block 5's header.
        fetcher.headers = vec![Some(header(100, B256::repeat_byte(0x64))), None, None];
        fetcher.from_block = 5;
        fetcher.to_block = 5;

        assert_eq!(fetcher.get_header_by_number(5).await.unwrap(), Some(h5));
    }

    #[tokio::test]
    async fn caches_range_and_skips_refetch() {
        let mut mock = MockParentChainReader::new();
        mock.with_header(header(50, B256::repeat_byte(0x32)));
        let mut fetcher = LogsAndHeadersFetcher::new(Arc::new(mock), 10, Address::ZERO);

        let state = MelState {
            parent_chain_block_number: 1,
            ..Default::default()
        };
        fetcher.fetch(&state).await.unwrap();
        assert_eq!(fetcher.from_block, 2);
        assert_eq!(fetcher.to_block, 12);
        assert_eq!(fetcher.chain_height, 50);
    }
}
