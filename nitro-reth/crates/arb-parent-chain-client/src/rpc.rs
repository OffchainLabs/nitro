use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_types_eth::{Block, Filter, Header, Log, Transaction, TransactionReceipt};

use super::{ParentChainReader, Result};

/// A [`ParentChainReader`] that talks to a node over JSON-RPC.
#[derive(Debug)]
pub struct RpcParentChainReader {
    provider: RootProvider,
}

#[async_trait::async_trait]
impl ParentChainReader for RpcParentChainReader {
    // NOTE: Implemented using get_header_by_number() so only works with geth/reth upstream.
    async fn header_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Header>> {
        let header = self.provider.get_header_by_number(num).await?;
        Ok(header)
    }

    // NOTE: Implemented using get_header_by_hash() so only works with geth/reth upstream.
    async fn header_by_hash(&self, hash: B256) -> Result<Option<Header>> {
        let header = self.provider.get_header_by_hash(hash).await?;
        Ok(header)
    }

    async fn block_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Block>> {
        let block = self.provider.get_block_by_number(num).full().await?;
        Ok(block)
    }

    async fn block_by_hash(&self, hash: B256) -> Result<Option<Block>> {
        let block = self.provider.get_block_by_hash(hash).full().await?;
        Ok(block)
    }

    async fn transaction_in_block(&self, block: B256, index: u64) -> Result<Option<Transaction>> {
        let tx = self
            .provider
            .get_transaction_by_block_hash_and_index(block, index as usize)
            .await?;
        Ok(tx)
    }

    async fn transaction_receipt(&self, tx: B256) -> Result<Option<TransactionReceipt>> {
        Ok(self.provider.get_transaction_receipt(tx).await?)
    }

    async fn transaction_by_hash(&self, hash: B256) -> Result<Option<Transaction>> {
        let tx = self.provider.get_transaction_by_hash(hash).await?;
        Ok(tx)
    }

    async fn filter_logs(&self, q: &Filter) -> Result<Vec<Log>> {
        let logs = self.provider.get_logs(q).await?;
        Ok(logs)
    }
}
