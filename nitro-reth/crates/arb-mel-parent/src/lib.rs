use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_types_eth::{Block, Filter, Header, Log, Transaction, TransactionReceipt};
use alloy_transport::{RpcError, TransportErrorKind};

#[derive(Debug, thiserror::Error)]
pub enum ParentChainError {
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
}

pub type Result<T, E = ParentChainError> = std::result::Result<T, E>;

#[async_trait::async_trait]
pub trait ParentChainReader: Send + Sync {
    async fn header_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Header>>;
    async fn header_by_hash(&self, hash: B256) -> Result<Option<Header>>;
    async fn block_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Block>>;
    async fn block_by_hash(&self, hash: B256) -> Result<Option<Block>>;
    async fn transaction_in_block(&self, block: B256, index: u64) -> Result<Option<Transaction>>;
    async fn transaction_receipt(&self, tx: B256) -> Result<Option<TransactionReceipt>>;
    async fn transaction_by_hash(&self, hash: B256) -> Result<Option<Transaction>>;
    async fn filter_logs(&self, q: &Filter) -> Result<Vec<Log>>;
}

#[derive(Debug)]
pub struct AlloyParentChainReader {
    provider: RootProvider,
}

#[async_trait::async_trait]
impl ParentChainReader for AlloyParentChainReader {
    async fn header_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Header>> {
        // TODO: could potentially use get_header_by_number if we know the parent is geth or reth
        let block = self.provider.get_block_by_number(num).await?;
        Ok(block.map(|b| b.header))
    }

    async fn header_by_hash(&self, hash: B256) -> Result<Option<Header>> {
        // TODO: could potentially use get_header_by_number if we know the parent is geth or reth
        let block = self.provider.get_block_by_hash(hash).await?;
        Ok(block.map(|b| b.header))
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
