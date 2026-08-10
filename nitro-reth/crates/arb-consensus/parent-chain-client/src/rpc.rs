use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_client::BatchRequest;
use alloy_rpc_types_eth::{Block, Filter, Header, Log, Transaction, TransactionReceipt};

use super::{ParentChainReader, Result};

/// A [`ParentChainReader`] that talks to a node over JSON-RPC.
#[derive(Debug)]
pub struct RpcParentChainReader {
    provider: RootProvider,
}

impl From<RootProvider> for RpcParentChainReader {
    fn from(provider: RootProvider) -> Self {
        Self { provider }
    }
}

#[async_trait::async_trait]
impl ParentChainReader for RpcParentChainReader {
    // NOTE: Implemented using get_header_by_number() so only works with geth/reth upstream.
    async fn header_by_number(&self, num: BlockNumberOrTag) -> Result<Option<Header>> {
        let header = self.provider.get_header_by_number(num).await?;
        Ok(header)
    }

    async fn headers_by_number_range(&self, from: u64, to: u64) -> Result<Vec<Option<Header>>> {
        if from > to {
            return Ok(Vec::new());
        }
        // One JSON-RPC batch (geth's `BatchCallContext` equivalent): queue an
        // `eth_getHeaderByNumber` per block, send once, then collect in order.
        // Responses are matched to calls by request id, so `waiters` stays aligned
        // with `from..=to`. This is a geth/reth specific RPC call
        let mut batch = BatchRequest::new(self.provider.client());
        let mut waiters = Vec::with_capacity((to - from + 1) as usize);
        for n in from..=to {
            waiters.push(batch.add_call::<_, Option<Header>>(
                "eth_getHeaderByNumber",
                &(BlockNumberOrTag::Number(n),),
            )?);
        }
        batch.send().await?;
        let mut headers = Vec::with_capacity(waiters.len());
        for waiter in waiters {
            headers.push(waiter.await?);
        }
        Ok(headers)
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

#[cfg(test)]
mod tests {
    use alloy_primitives::Address;
    use alloy_provider::mock::Asserter;
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::BlockTransactions;

    use super::*;
    use crate::{
        ParentChainError,
        test_utils::{block, header, log_at, receipt, tx},
    };

    /// A reader backed by a mock transport that replays the queued responses.
    fn reader(asserter: Asserter) -> RpcParentChainReader {
        RootProvider::new(RpcClient::mocked(asserter)).into()
    }

    #[tokio::test]
    async fn header_by_number_deserializes_response() {
        let h = header(7, B256::repeat_byte(7));
        let asserter = Asserter::new();
        asserter.push_success(&h);

        let got = reader(asserter)
            .header_by_number(BlockNumberOrTag::Number(7))
            .await
            .unwrap();
        assert_eq!(got, Some(h));
    }

    #[tokio::test]
    async fn header_by_hash_deserializes_response() {
        let h = header(7, B256::repeat_byte(7));
        let asserter = Asserter::new();
        asserter.push_success(&h);

        let got = reader(asserter).header_by_hash(h.hash).await.unwrap();
        assert_eq!(got, Some(h));
    }

    #[tokio::test]
    async fn block_by_number_deserializes_response() {
        let b = block(5, B256::repeat_byte(5), BlockTransactions::Full(vec![]));
        let asserter = Asserter::new();
        asserter.push_success(&b);

        let got = reader(asserter)
            .block_by_number(BlockNumberOrTag::Number(5))
            .await
            .unwrap();
        assert_eq!(got, Some(b));
    }

    #[tokio::test]
    async fn block_by_hash_deserializes_response() {
        let b = block(5, B256::repeat_byte(5), BlockTransactions::Full(vec![]));
        let asserter = Asserter::new();
        asserter.push_success(&b);

        let got = reader(asserter)
            .block_by_hash(B256::repeat_byte(5))
            .await
            .unwrap();
        assert_eq!(got, Some(b));
    }

    #[tokio::test]
    async fn transaction_in_block_deserializes_response() {
        let block_hash = B256::repeat_byte(5);
        let t = tx(1, Some((block_hash, 0)));
        let asserter = Asserter::new();
        asserter.push_success(&t);

        let got = reader(asserter)
            .transaction_in_block(block_hash, 0)
            .await
            .unwrap();
        assert_eq!(got, Some(t));
    }

    #[tokio::test]
    async fn transaction_by_hash_deserializes_response() {
        let t = tx(1, None);
        let asserter = Asserter::new();
        asserter.push_success(&t);

        let got = reader(asserter)
            .transaction_by_hash(B256::repeat_byte(1))
            .await
            .unwrap();
        assert_eq!(got, Some(t));
    }

    #[tokio::test]
    async fn transaction_receipt_deserializes_response() {
        let r = receipt(B256::repeat_byte(0xcc));
        let asserter = Asserter::new();
        asserter.push_success(&r);

        let got = reader(asserter)
            .transaction_receipt(B256::repeat_byte(0xcc))
            .await
            .unwrap();
        assert_eq!(got, Some(r));
    }

    #[tokio::test]
    async fn filter_logs_deserializes_response() {
        let logs = vec![log_at(10, Address::repeat_byte(0xaa))];
        let asserter = Asserter::new();
        asserter.push_success(&logs);

        let got = reader(asserter).filter_logs(&Filter::new()).await.unwrap();
        assert_eq!(got, logs);
    }

    #[tokio::test]
    async fn missing_header_returns_none() {
        let asserter = Asserter::new();
        asserter.push_success(&Option::<Header>::None);

        let got = reader(asserter)
            .header_by_number(BlockNumberOrTag::Number(1))
            .await
            .unwrap();
        assert!(got.is_none());
    }

    #[tokio::test]
    async fn empty_filter_logs_returns_empty_vec() {
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Log>::new());

        let got = reader(asserter).filter_logs(&Filter::new()).await.unwrap();
        assert!(got.is_empty());
    }

    #[tokio::test]
    async fn transport_error_is_propagated() {
        let asserter = Asserter::new();
        asserter.push_failure_msg("boom");

        let err = reader(asserter)
            .header_by_number(BlockNumberOrTag::Number(1))
            .await
            .unwrap_err();
        assert!(matches!(err, ParentChainError::Transport(_)));
    }
}
