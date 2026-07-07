use std::collections::HashMap;

use alloy_eips::{eip2718::Encodable2718, BlockNumberOrTag};
use alloy_primitives::B256;
use alloy_rpc_types_eth::{Block, Filter, Header, Log, Transaction, TransactionReceipt};

use super::{ParentChainReader, Result};

/// An in-memory [`ParentChainReader`] for tests.
///
/// Data is inserted up front with the `with_*` builders, then read back
/// through the trait. Values are indexed by the keys the trait looks them up
/// by (block number, block hash, transaction hash, ...), so lookups can happen
/// in any order.
#[derive(Debug, Default)]
pub struct MockParentChainReader {
    headers_by_number: HashMap<u64, Header>,
    headers_by_hash: HashMap<B256, Header>,
    blocks_by_number: HashMap<u64, Block>,
    blocks_by_hash: HashMap<B256, Block>,
    txs_by_hash: HashMap<B256, Transaction>,
    txs_by_block_index: HashMap<(B256, u64), Transaction>,
    receipts_by_tx_hash: HashMap<B256, TransactionReceipt>,
    logs: Vec<Log>,
}

impl MockParentChainReader {
    /// Creates an empty reader.
    pub fn new() -> Self {
        Self::default()
    }

    /// Registers a header, indexed by both its number and its hash.
    pub fn with_header(&mut self, header: Header) -> &mut Self {
        self.headers_by_number
            .insert(header.inner.number, header.clone());
        self.headers_by_hash.insert(header.hash, header);
        self
    }

    /// Registers a block along with its header and any full transactions it
    /// carries, indexed by both number and hash.
    pub fn with_block(&mut self, block: Block) -> &mut Self {
        self.with_header(block.header.clone());
        // `txns()` yields nothing if the block only carries transaction hashes.
        for tx in block.transactions.txns() {
            self.with_transaction(tx.clone());
        }
        self.blocks_by_number
            .insert(block.header.inner.number, block.clone());
        self.blocks_by_hash.insert(block.header.hash, block);
        self
    }

    /// Registers a transaction, indexed by its hash and by its `(block hash, index)` if it has
    /// been mined.
    pub fn with_transaction(&mut self, tx: Transaction) -> &mut Self {
        let hash = tx.inner.trie_hash();
        if let (Some(block_hash), Some(index)) = (tx.block_hash, tx.transaction_index) {
            self.txs_by_block_index
                .insert((block_hash, index), tx.clone());
        }
        self.txs_by_hash.insert(hash, tx);
        self
    }

    /// Registers a receipt, indexed by its transaction hash.
    pub fn with_receipt(&mut self, receipt: TransactionReceipt) -> &mut Self {
        self.receipts_by_tx_hash
            .insert(receipt.transaction_hash, receipt);
        self
    }

    /// Registers a log to be matched against filters in [`Self::filter_logs`].
    ///
    /// The log must carry `block_number` and `block_hash`. A log missing
    /// either never matches a filter (mirroring a real node, which only
    /// returns logs from mined blocks).
    pub fn with_log(&mut self, log: Log) -> &mut Self {
        self.logs.push(log);
        self
    }

    /// Resolves a [`BlockNumberOrTag`] to a concrete number against `known`,
    /// the set of numbers present in the map being queried. Tags that refer to
    /// the chain head (`latest`/`safe`/`finalized`/`pending`) resolve to the
    /// highest number present, `earliest` to the lowest. Resolving against the
    /// queried map means a header-only entry can't shadow the latest full block
    /// in `block_by_number`, and vice versa.
    fn resolve_number(num: BlockNumberOrTag, known: impl Iterator<Item = u64>) -> Option<u64> {
        match num {
            BlockNumberOrTag::Number(n) => Some(n),
            BlockNumberOrTag::Earliest => known.min(),
            BlockNumberOrTag::Latest
            | BlockNumberOrTag::Safe
            | BlockNumberOrTag::Finalized
            | BlockNumberOrTag::Pending => known.max(),
        }
    }
}

#[async_trait::async_trait]
impl ParentChainReader for MockParentChainReader {
    async fn header_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Header>> {
        Ok(
            Self::resolve_number(num, self.headers_by_number.keys().copied())
                .and_then(|n| self.headers_by_number.get(&n).cloned()),
        )
    }

    async fn header_by_hash(&self, hash: B256) -> Result<Option<Header>> {
        Ok(self.headers_by_hash.get(&hash).cloned())
    }

    async fn block_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Block>> {
        Ok(
            Self::resolve_number(num, self.blocks_by_number.keys().copied())
                .and_then(|n| self.blocks_by_number.get(&n).cloned()),
        )
    }

    async fn block_by_hash(&self, hash: B256) -> Result<Option<Block>> {
        Ok(self.blocks_by_hash.get(&hash).cloned())
    }

    async fn transaction_in_block(&self, block: B256, index: u64) -> Result<Option<Transaction>> {
        Ok(self.txs_by_block_index.get(&(block, index)).cloned())
    }

    async fn transaction_receipt(&self, tx: B256) -> Result<Option<TransactionReceipt>> {
        Ok(self.receipts_by_tx_hash.get(&tx).cloned())
    }

    async fn transaction_by_hash(&self, hash: B256) -> Result<Option<Transaction>> {
        Ok(self.txs_by_hash.get(&hash).cloned())
    }

    async fn filter_logs(&self, q: &Filter) -> Result<Vec<Log>> {
        Ok(self
            .logs
            .iter()
            .filter(|log| q.rpc_matches(log))
            .cloned()
            .collect())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{block, header, log_at, receipt, tx};
    use alloy_primitives::{Address, B256};
    use alloy_rpc_types_eth::BlockTransactions;

    #[tokio::test]
    async fn header_by_number_returns_matching_header() {
        let h5 = header(5, B256::repeat_byte(5));
        let mut mock = MockParentChainReader::new();
        mock.with_header(h5.clone());

        assert_eq!(
            mock.header_by_number(BlockNumberOrTag::Number(5))
                .await
                .unwrap(),
            Some(h5)
        );
    }

    #[tokio::test]
    async fn header_by_number_latest_returns_highest() {
        let h5 = header(5, B256::repeat_byte(5));
        let h6 = header(6, B256::repeat_byte(6));
        let mut mock = MockParentChainReader::new();
        mock.with_header(h5).with_header(h6.clone());

        assert_eq!(
            mock.header_by_number(BlockNumberOrTag::Latest)
                .await
                .unwrap(),
            Some(h6)
        );
    }

    #[tokio::test]
    async fn header_by_number_earliest_returns_lowest() {
        let h5 = header(5, B256::repeat_byte(5));
        let h6 = header(6, B256::repeat_byte(6));
        let mut mock = MockParentChainReader::new();
        mock.with_header(h5.clone()).with_header(h6);

        assert_eq!(
            mock.header_by_number(BlockNumberOrTag::Earliest)
                .await
                .unwrap(),
            Some(h5)
        );
    }

    #[tokio::test]
    async fn header_by_number_unknown_returns_none() {
        let mock = MockParentChainReader::new();

        assert!(mock
            .header_by_number(BlockNumberOrTag::Number(99))
            .await
            .unwrap()
            .is_none());
    }

    #[tokio::test]
    async fn header_by_hash_returns_matching_header() {
        let h5 = header(5, B256::repeat_byte(5));
        let mut mock = MockParentChainReader::new();
        mock.with_header(h5.clone());

        assert_eq!(mock.header_by_hash(h5.hash).await.unwrap(), Some(h5));
    }

    #[tokio::test]
    async fn header_by_hash_unknown_returns_none() {
        let mock = MockParentChainReader::new();

        assert!(mock
            .header_by_hash(B256::repeat_byte(0xff))
            .await
            .unwrap()
            .is_none());
    }

    #[tokio::test]
    async fn block_by_number_returns_matching_block() {
        let hash = B256::repeat_byte(5);
        let mut mock = MockParentChainReader::new();
        mock.with_block(block(5, hash, BlockTransactions::Full(vec![])));

        let got = mock
            .block_by_number(BlockNumberOrTag::Number(5))
            .await
            .unwrap()
            .unwrap();
        assert_eq!(got.header.hash, hash);
    }

    #[tokio::test]
    async fn block_by_number_unknown_returns_none() {
        let mock = MockParentChainReader::new();

        assert!(mock
            .block_by_number(BlockNumberOrTag::Number(99))
            .await
            .unwrap()
            .is_none());
    }

    #[tokio::test]
    async fn block_by_hash_unknown_returns_none() {
        let mock = MockParentChainReader::new();

        assert!(mock
            .block_by_hash(B256::repeat_byte(0xff))
            .await
            .unwrap()
            .is_none());
    }

    #[tokio::test]
    async fn transaction_in_block_unknown_returns_none() {
        let mock = MockParentChainReader::new();

        assert!(mock
            .transaction_in_block(B256::repeat_byte(5), 0)
            .await
            .unwrap()
            .is_none());
    }

    #[tokio::test]
    async fn transaction_receipt_returns_matching_receipt() {
        let tx_hash = B256::repeat_byte(0xcc);
        let r = receipt(tx_hash);

        let mut mock = MockParentChainReader::new();
        mock.with_receipt(r.clone());

        assert_eq!(mock.transaction_receipt(tx_hash).await.unwrap(), Some(r));
    }

    #[tokio::test]
    async fn transaction_receipt_unknown_returns_none() {
        let mock = MockParentChainReader::new();

        assert!(mock
            .transaction_receipt(B256::repeat_byte(0xff))
            .await
            .unwrap()
            .is_none());
    }

    #[tokio::test]
    async fn transaction_by_hash_returns_pending_transaction() {
        let t = tx(1, None); // pending: no block context
        let tx_hash = t.inner.trie_hash();

        let mut mock = MockParentChainReader::new();
        mock.with_transaction(t.clone());

        assert_eq!(mock.transaction_by_hash(tx_hash).await.unwrap(), Some(t));
    }

    #[tokio::test]
    async fn transaction_by_hash_unknown_returns_none() {
        let mock = MockParentChainReader::new();

        assert!(mock
            .transaction_by_hash(B256::repeat_byte(0xff))
            .await
            .unwrap()
            .is_none());
    }

    #[tokio::test]
    async fn filter_logs_respects_range_and_address() {
        let wanted = Address::repeat_byte(0xaa);

        let mut mock = MockParentChainReader::new();
        mock.with_log(log_at(10, wanted)) // match
            .with_log(log_at(30, wanted)) // out of range
            .with_log(log_at(12, Address::repeat_byte(0xbb))); // wrong address

        let filter = Filter::new().address(wanted).from_block(0).to_block(20);
        let got = mock.filter_logs(&filter).await.unwrap();

        assert_eq!(got.len(), 1);
        assert_eq!(got[0].block_number, Some(10));
    }

    #[tokio::test]
    async fn filter_logs_ignores_logs_without_block_context() {
        // A log with no block_number/block_hash does not match, even against a
        // filter with no constraints at all.
        let mut log = Log::default();
        log.inner.address = Address::repeat_byte(0xaa);

        let mut mock = MockParentChainReader::new();
        mock.with_log(log);

        let got = mock.filter_logs(&Filter::new()).await.unwrap();

        assert!(got.is_empty());
    }

    #[tokio::test]
    async fn with_block_registers_its_header() {
        let hash = B256::repeat_byte(5);
        let mut mock = MockParentChainReader::new();
        mock.with_block(block(5, hash, BlockTransactions::Full(vec![])));

        assert!(mock.header_by_hash(hash).await.unwrap().is_some());
        assert!(mock
            .header_by_number(BlockNumberOrTag::Number(5))
            .await
            .unwrap()
            .is_some());
    }

    #[tokio::test]
    async fn with_block_registers_full_transactions() {
        let block_hash = B256::repeat_byte(5);
        let t = tx(1, Some((block_hash, 0)));
        let tx_hash = t.inner.trie_hash();

        let mut mock = MockParentChainReader::new();
        mock.with_block(block(
            5,
            block_hash,
            BlockTransactions::Full(vec![t.clone()]),
        ));

        assert_eq!(
            mock.transaction_by_hash(tx_hash).await.unwrap(),
            Some(t.clone())
        );
        assert_eq!(
            mock.transaction_in_block(block_hash, 0).await.unwrap(),
            Some(t)
        );
    }

    #[tokio::test]
    async fn with_block_hashes_only_registers_no_transactions() {
        let block_hash = B256::repeat_byte(5);
        let tx_hash = B256::repeat_byte(0xee);

        let mut mock = MockParentChainReader::new();
        mock.with_block(block(
            5,
            block_hash,
            BlockTransactions::Hashes(vec![tx_hash]),
        ));

        assert!(mock.block_by_hash(block_hash).await.unwrap().is_some());
        assert!(mock.transaction_by_hash(tx_hash).await.unwrap().is_none());
    }

    #[tokio::test]
    async fn with_transaction_indexes_mined_tx_by_hash_and_position() {
        let block_hash = B256::repeat_byte(5);
        let t = tx(1, Some((block_hash, 0)));
        let tx_hash = t.inner.trie_hash();

        let mut mock = MockParentChainReader::new();
        mock.with_transaction(t.clone());

        assert_eq!(
            mock.transaction_by_hash(tx_hash).await.unwrap(),
            Some(t.clone())
        );
        assert_eq!(
            mock.transaction_in_block(block_hash, 0).await.unwrap(),
            Some(t)
        );
    }

    #[tokio::test]
    async fn header_only_entry_does_not_shadow_latest_block() {
        let mut mock = MockParentChainReader::new();
        // A full block at 5, plus a bare header at 7 (a higher number).
        mock.with_block(block(
            5,
            B256::repeat_byte(5),
            BlockTransactions::Full(vec![]),
        ));
        mock.with_header(header(7, B256::repeat_byte(7)));

        // `block_by_number(latest)` resolves against blocks only -> 5.
        let latest_block = mock
            .block_by_number(BlockNumberOrTag::Latest)
            .await
            .unwrap()
            .unwrap();
        assert_eq!(latest_block.header.inner.number, 5);

        // `header_by_number(latest)` resolves against headers -> 7.
        let latest_header = mock
            .header_by_number(BlockNumberOrTag::Latest)
            .await
            .unwrap()
            .unwrap();
        assert_eq!(latest_header.inner.number, 7);
    }
}
