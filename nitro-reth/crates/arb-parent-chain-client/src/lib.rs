//! A client for reading Arbitrum's parent chain over JSON-RPC.
//!
//! Provides [`ParentChainReader`], a small trait that wraps the standard
//! Ethereum `eth_*` calls for fetching blocks, transactions, receipts, and
//! logs, plus a ready-to-use implementation backed by a live RPC provider.
//!
//! This only reads data; it doesn't interpret any of it. Code that needs to do
//! deterministic, no-IO processing is expected to pull what it needs through
//! this reader first and then work on the results.

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_types_eth::{Block, Filter, Header, Log, Transaction, TransactionReceipt};
use alloy_transport::{RpcError, TransportErrorKind};

/// Something went wrong while reading from the parent chain.
#[derive(Debug, thiserror::Error)]
pub enum ParentChainError {
    /// RPC call failure
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
}

/// Return type for [`ParentChainReader`]
pub type Result<T, E = ParentChainError> = std::result::Result<T, E>;

/// Reads blocks, transactions, receipts, and logs from the parent chain.
///
/// Each method is a wrapper around one Ethereum JSON-RPC call. A lookup
/// that finds nothing returns `Ok(None)` or an empty `Vec`. An `Err(_)`
/// means the request itself failed.
#[async_trait::async_trait]
pub trait ParentChainReader: Send + Sync {
    /// Returns the header of the block with the given number.
    async fn header_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Header>>;
    /// Returns the header of the block with the given hash.
    async fn header_by_hash(&self, hash: B256) -> Result<Option<Header>>;
    /// Returns the full block, including its transactions, with the given number.
    async fn block_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Block>>;
    /// Returns the full block, including its transactions, with the given hash.
    async fn block_by_hash(&self, hash: B256) -> Result<Option<Block>>;
    /// Returns the transaction at the given index within the given block.
    async fn transaction_in_block(&self, block: B256, index: u64) -> Result<Option<Transaction>>;
    /// Returns the receipt for the transaction with the given hash.
    async fn transaction_receipt(&self, tx: B256) -> Result<Option<TransactionReceipt>>;
    /// Returns the transaction with the given hash.
    async fn transaction_by_hash(&self, hash: B256) -> Result<Option<Transaction>>;
    /// Returns every log matching the given filter.
    async fn filter_logs(&self, q: &Filter) -> Result<Vec<Log>>;
}

/// A [`ParentChainReader`] that talks to a node over JSON-RPC.
#[derive(Debug)]
pub struct RpcParentChainReader {
    provider: RootProvider,
}

#[async_trait::async_trait]
impl ParentChainReader for RpcParentChainReader {
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
