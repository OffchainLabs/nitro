//! Arbitrum consensus implementation.
//!
//! L2 blocks are validated by the sequencer and posted to L1.
//! The consensus layer trusts the sequencer's block production
//! and performs only basic structural validation.

use std::fmt::Debug;
use std::sync::Arc;

use reth_chainspec::{EthChainSpec, EthereumHardforks};
use reth_consensus::{Consensus, ConsensusError, FullConsensus, HeaderValidator};
use reth_execution_types::BlockExecutionResult;
use reth_primitives_traits::{
    Block, BlockHeader, NodePrimitives, RecoveredBlock, SealedBlock, SealedHeader,
};

/// Arbitrum consensus engine.
///
/// Trusts the sequencer for block validity. Performs minimal
/// structural checks on headers.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ArbConsensus<CS> {
    chain_spec: Arc<CS>,
}

impl<CS> ArbConsensus<CS> {
    /// Create a new consensus engine.
    pub fn new(chain_spec: Arc<CS>) -> Self {
        Self { chain_spec }
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
        _block: &RecoveredBlock<N::Block>,
        _result: &BlockExecutionResult<N::Receipt>,
        _receipt_root_bloom: Option<reth_consensus::ReceiptRootBloom>,
    ) -> Result<(), ConsensusError> {
        Ok(())
    }
}
