//! Block producer implementation.
//!
//! Produces blocks from L1 incoming messages by parsing transactions,
//! executing them against the current state, and persisting the results.

use std::sync::Arc;

use alloy_consensus::{Block, BlockBody, BlockHeader, Header, proofs, EMPTY_OMMER_ROOT_HASH};
use alloy_primitives::{B64, B256, U256};
use alloy_rpc_types_eth::BlockNumberOrTag;
use parking_lot::Mutex;
use reth_chainspec::ChainSpec;
use reth_provider::{
    BlockNumReader, BlockReaderIdExt, HeaderProvider, StateProviderFactory,
};
use tracing::{info, warn};

use arb_rpc::block_producer::{BlockProducer, BlockProducerError, BlockProductionInput, ProducedBlock};
use arbos::parse_l2::parse_l2_transactions;

/// Concrete block producer backed by reth's database.
pub struct ArbBlockProducer<Provider> {
    provider: Provider,
    #[allow(dead_code)]
    chain_spec: Arc<ChainSpec>,
    /// Mutex to serialize block production.
    produce_lock: Mutex<()>,
}

impl<Provider> ArbBlockProducer<Provider> {
    pub fn new(provider: Provider, chain_spec: Arc<ChainSpec>) -> Self {
        Self {
            provider,
            chain_spec,
            produce_lock: Mutex::new(()),
        }
    }
}

impl<Provider> ArbBlockProducer<Provider>
where
    Provider: BlockNumReader + BlockReaderIdExt + HeaderProvider + StateProviderFactory + Send + Sync + 'static,
    <Provider as HeaderProvider>::Header: BlockHeader,
{
    /// Get the current head block number.
    fn head_block_number(&self) -> Result<u64, BlockProducerError> {
        self.provider
            .best_block_number()
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))
    }

    /// Produce a minimal block from a message that generates no transactions.
    fn produce_empty_block(
        &self,
        input: &BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError> {
        let head_num = self.head_block_number()?;
        let parent_header = self
            .provider
            .sealed_header_by_number_or_tag(BlockNumberOrTag::Number(head_num))
            .map_err(|e| BlockProducerError::StateAccess(e.to_string()))?
            .ok_or_else(|| {
                BlockProducerError::StateAccess(format!("Parent block {head_num} not found"))
            })?;

        let parent = parent_header.header();
        let l2_block_number = parent.number().saturating_add(1);

        // Timestamp: max(l1_timestamp, parent_timestamp)
        let timestamp = input.l1_timestamp.max(parent.timestamp());

        // Copy extra_data from parent (contains send root)
        let extra_data = parent.extra_data().to_vec();

        // Copy mix_hash from parent (preserves arbos_version, l1_block_number, send_count)
        let mix_hash = parent.mix_hash().unwrap_or_default();

        // State root unchanged (no state changes)
        let state_root = parent.state_root();

        let header = Header {
            parent_hash: parent_header.hash(),
            ommers_hash: EMPTY_OMMER_ROOT_HASH,
            beneficiary: input.sender,
            state_root,
            transactions_root: proofs::calculate_transaction_root::<
                arb_primitives::ArbTransactionSigned,
            >(&[]),
            receipts_root: proofs::calculate_receipt_root::<arb_primitives::ArbReceipt>(&[]),
            withdrawals_root: None,
            logs_bloom: Default::default(),
            timestamp,
            mix_hash,
            nonce: B64::from(input.delayed_messages_read.to_be_bytes()),
            base_fee_per_gas: parent.base_fee_per_gas(),
            number: l2_block_number,
            gas_limit: parent.gas_limit(),
            difficulty: U256::from(1),
            gas_used: 0,
            extra_data: extra_data.into(),
            parent_beacon_block_root: None,
            blob_gas_used: None,
            excess_blob_gas: None,
            requests_hash: None,
        };

        let block = Block::<arb_primitives::ArbTransactionSigned> {
            header,
            body: BlockBody {
                transactions: vec![],
                ommers: Default::default(),
                withdrawals: None,
            },
        };

        // Seal the block
        let sealed = reth_primitives_traits::SealedBlock::seal_slow(block);
        let block_hash = sealed.hash();

        // Extract send root from extra_data
        let send_root = if sealed.header().extra_data.len() >= 32 {
            B256::from_slice(&sealed.header().extra_data[..32])
        } else {
            B256::ZERO
        };

        info!(
            target: "block_producer",
            block_num = l2_block_number,
            ?block_hash,
            ?send_root,
            "Produced empty block"
        );

        Ok(ProducedBlock {
            block_hash,
            send_root,
        })
    }
}

#[async_trait::async_trait]
impl<Provider> BlockProducer for ArbBlockProducer<Provider>
where
    Provider: BlockNumReader + BlockReaderIdExt + HeaderProvider + StateProviderFactory + Send + Sync + 'static,
    <Provider as HeaderProvider>::Header: BlockHeader,
{
    async fn produce_block(
        &self,
        msg_idx: u64,
        input: BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError> {
        let _lock = self.produce_lock.lock();

        // Validate that this message is the next expected one
        let head_num = self.head_block_number()?;
        let expected_block = head_num + 1;
        let actual_block = msg_idx; // For genesis_block_num=0: block_num = msg_idx

        if expected_block != actual_block {
            return Err(BlockProducerError::Unexpected(format!(
                "Expected block {expected_block} but got msg_idx {msg_idx} (block {actual_block})"
            )));
        }

        // Parse L2 transactions from the message
        let parsed_txs = parse_l2_transactions(
            input.kind,
            input.sender,
            &input.l2_msg,
            input.request_id,
            input.l1_base_fee,
        )
        .map_err(|e| BlockProducerError::Parse(e.to_string()))?;

        info!(
            target: "block_producer",
            msg_idx,
            kind = input.kind,
            num_txs = parsed_txs.len(),
            "Parsed L1 message"
        );

        // For now, produce empty blocks for all messages.
        // Full execution will be implemented next.
        if !parsed_txs.is_empty() {
            warn!(
                target: "block_producer",
                msg_idx,
                kind = input.kind,
                num_txs = parsed_txs.len(),
                "Full execution not yet implemented, producing empty block"
            );
        }

        self.produce_empty_block(&input)
    }
}
