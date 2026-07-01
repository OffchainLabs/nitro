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
