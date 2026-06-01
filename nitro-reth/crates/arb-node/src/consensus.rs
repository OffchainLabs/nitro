//! Arbitrum consensus implementation.
//!
//! L2 blocks are validated by the sequencer and posted to L1.
//! The consensus layer trusts the sequencer's block production
//! and performs only basic structural validation.

use std::{fmt::Debug, sync::Arc};

use alloy_consensus::{proofs::calculate_receipt_root, TxReceipt};
use alloy_primitives::Bloom;
use reth_chainspec::{EthChainSpec, EthereumHardforks};
use reth_consensus::{Consensus, ConsensusError, FullConsensus, HeaderValidator, ReceiptRootBloom};
use reth_execution_types::BlockExecutionResult;
use reth_primitives_traits::{
    receipt::gas_spent_by_transactions, Block, BlockHeader, GotExpected, NodePrimitives, Receipt,
    RecoveredBlock, SealedBlock, SealedHeader,
};

/// Arbitrum consensus engine.
///
/// Trusts the sequencer for block validity. Performs minimal
/// structural checks on headers.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ArbConsensus<CS> {
    chain_spec: Arc<CS>,
    verify_execution: bool,
}

impl<CS> ArbConsensus<CS> {
    /// Consensus engine that trusts sequencer-produced blocks (the node default).
    pub fn new(chain_spec: Arc<CS>) -> Self {
        Self {
            chain_spec,
            verify_execution: false,
        }
    }

    /// Consensus engine that re-checks execution output against the stored
    /// header, for offline commands that re-execute persisted blocks.
    pub fn new_verifying(chain_spec: Arc<CS>) -> Self {
        Self {
            chain_spec,
            verify_execution: true,
        }
    }
}

impl<H, CS> HeaderValidator<H> for ArbConsensus<CS>
where
    H: BlockHeader,
    CS: EthChainSpec<Header = H> + EthereumHardforks + Debug + Send + Sync,
{
    fn validate_header(&self, _header: &SealedHeader<H>) -> Result<(), ConsensusError> {
        Ok(())
    }

    fn validate_header_against_parent(
        &self,
        _header: &SealedHeader<H>,
        _parent: &SealedHeader<H>,
    ) -> Result<(), ConsensusError> {
        Ok(())
    }
}

impl<B, CS> Consensus<B> for ArbConsensus<CS>
where
    B: Block,
    CS: EthChainSpec<Header = B::Header> + EthereumHardforks + Debug + Send + Sync,
{
    fn validate_body_against_header(
        &self,
        _body: &B::Body,
        _header: &SealedHeader<B::Header>,
    ) -> Result<(), ConsensusError> {
        Ok(())
    }

    fn validate_block_pre_execution(&self, _block: &SealedBlock<B>) -> Result<(), ConsensusError> {
        Ok(())
    }
}

impl<N, CS> FullConsensus<N> for ArbConsensus<CS>
where
    N: NodePrimitives,
    CS: EthChainSpec<Header = N::BlockHeader> + EthereumHardforks + Debug + Send + Sync,
{
    fn validate_block_post_execution(
        &self,
        block: &RecoveredBlock<N::Block>,
        result: &BlockExecutionResult<N::Receipt>,
        receipt_root_bloom: Option<ReceiptRootBloom>,
    ) -> Result<(), ConsensusError> {
        if !self.verify_execution {
            return Ok(());
        }
        verify_block_execution(block.header(), &result.receipts, receipt_root_bloom)
    }
}

/// Check cumulative gas, receipts root, and logs bloom against the header.
/// ArbOS blocks carry no EIP-7685 requests, so that branch is omitted.
fn verify_block_execution<H, R>(
    header: &H,
    receipts: &[R],
    receipt_root_bloom: Option<ReceiptRootBloom>,
) -> Result<(), ConsensusError>
where
    H: BlockHeader,
    R: Receipt,
{
    let cumulative_gas_used = receipts
        .last()
        .map(|r| r.cumulative_gas_used())
        .unwrap_or(0);
    if header.gas_used() != cumulative_gas_used {
        return Err(ConsensusError::BlockGasUsed {
            gas: GotExpected {
                got: cumulative_gas_used,
                expected: header.gas_used(),
            },
            gas_spent_by_tx: gas_spent_by_transactions(receipts),
        });
    }

    let (receipts_root, logs_bloom) = receipt_root_bloom.unwrap_or_else(|| {
        let with_bloom = receipts
            .iter()
            .map(TxReceipt::with_bloom_ref)
            .collect::<Vec<_>>();
        let root = calculate_receipt_root(&with_bloom);
        let bloom = with_bloom
            .iter()
            .fold(Bloom::ZERO, |bloom, r| bloom | r.bloom_ref());
        (root, bloom)
    });

    if receipts_root != header.receipts_root() {
        return Err(ConsensusError::BodyReceiptRootDiff(
            GotExpected {
                got: receipts_root,
                expected: header.receipts_root(),
            }
            .into(),
        ));
    }
    if logs_bloom != header.logs_bloom() {
        return Err(ConsensusError::BodyBloomLogDiff(
            GotExpected {
                got: logs_bloom,
                expected: header.logs_bloom(),
            }
            .into(),
        ));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::verify_block_execution;
    use alloy_consensus::{
        proofs::calculate_receipt_root, Eip658Value, Header, Receipt as AlloyReceipt, TxReceipt,
    };
    use alloy_primitives::{Bloom, B256};
    use arb_primitives::{ArbReceipt, ArbReceiptKind};
    use reth_consensus::ConsensusError;

    fn receipts() -> Vec<ArbReceipt> {
        vec![ArbReceipt::new(ArbReceiptKind::Eip1559(AlloyReceipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21_000,
            logs: Vec::new(),
        }))]
    }

    fn matching_header(receipts: &[ArbReceipt]) -> Header {
        let with_bloom = receipts
            .iter()
            .map(TxReceipt::with_bloom_ref)
            .collect::<Vec<_>>();
        Header {
            gas_used: receipts
                .last()
                .map(|r| r.cumulative_gas_used())
                .unwrap_or(0),
            receipts_root: calculate_receipt_root(&with_bloom),
            logs_bloom: with_bloom
                .iter()
                .fold(Bloom::ZERO, |b, r| b | r.bloom_ref()),
            ..Default::default()
        }
    }

    #[test]
    fn accepts_matching_block() {
        let receipts = receipts();
        let header = matching_header(&receipts);
        assert!(verify_block_execution(&header, &receipts, None).is_ok());
    }

    #[test]
    fn rejects_gas_mismatch() {
        let receipts = receipts();
        let mut header = matching_header(&receipts);
        header.gas_used += 1;
        assert!(matches!(
            verify_block_execution(&header, &receipts, None),
            Err(ConsensusError::BlockGasUsed { .. })
        ));
    }

    #[test]
    fn rejects_receipts_root_mismatch() {
        let receipts = receipts();
        let mut header = matching_header(&receipts);
        header.receipts_root = B256::ZERO;
        assert!(matches!(
            verify_block_execution(&header, &receipts, None),
            Err(ConsensusError::BodyReceiptRootDiff(_))
        ));
    }

    #[test]
    fn rejects_logs_bloom_mismatch() {
        let receipts = receipts();
        let mut header = matching_header(&receipts);
        header.logs_bloom = Bloom::from([1u8; 256]);
        assert!(matches!(
            verify_block_execution(&header, &receipts, None),
            Err(ConsensusError::BodyBloomLogDiff(_))
        ));
    }
}
