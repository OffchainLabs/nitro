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
use alloy_rpc_types_eth::{Block, Filter, Header, Log, Transaction, TransactionReceipt};
use alloy_transport::{RpcError, TransportErrorKind};

mod mock;
mod rpc;

#[cfg(any(test, feature = "test-utils"))]
pub mod test_utils;

pub use mock::MockParentChainReader;
pub use rpc::RpcParentChainReader;

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
