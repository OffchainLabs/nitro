//! Arbitrum node builder.
//!
//! Provides the node type definition and component builders
//! needed to launch an Arbitrum reth node.

pub mod args;
pub mod consensus;

use std::sync::Arc;

use arb_payload::ArbEngineTypes;
use arb_primitives::{ArbPrimitives, ArbTransactionSigned};
use reth_chainspec::ChainSpec;
use reth_node_builder::{
    components::{ConsensusBuilder, ExecutorBuilder},
    BuilderContext, FullNodeTypes, NodeTypes,
};
use reth_storage_api::EthStorage;

use arb_evm::ArbEvmConfig;

use crate::args::RollupArgs;
use crate::consensus::ArbConsensus;

/// Arbitrum storage type.
pub type ArbStorage = EthStorage<ArbTransactionSigned>;

/// Arbitrum node configuration.
#[derive(Debug, Clone, Default)]
pub struct ArbNode {
    /// Rollup CLI arguments.
    pub args: RollupArgs,
}

impl ArbNode {
    /// Create a new Arbitrum node configuration.
    pub fn new(args: RollupArgs) -> Self {
        Self { args }
    }
}

impl NodeTypes for ArbNode {
    type Primitives = ArbPrimitives;
    type ChainSpec = ChainSpec;
    type Storage = ArbStorage;
    type Payload = ArbEngineTypes;
}

/// Builder for the Arbitrum EVM executor component.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbExecutorBuilder;

impl<N> ExecutorBuilder<N> for ArbExecutorBuilder
where
    N: FullNodeTypes<Types: NodeTypes<ChainSpec = ChainSpec, Primitives = ArbPrimitives>>,
{
    type EVM = ArbEvmConfig;

    async fn build_evm(self, ctx: &BuilderContext<N>) -> eyre::Result<Self::EVM> {
        Ok(ArbEvmConfig::new(ctx.chain_spec()))
    }
}

/// Builder for the Arbitrum consensus component.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbConsensusBuilder;

impl<N> ConsensusBuilder<N> for ArbConsensusBuilder
where
    N: FullNodeTypes<Types: NodeTypes<ChainSpec = ChainSpec, Primitives = ArbPrimitives>>,
{
    type Consensus = Arc<ArbConsensus<ChainSpec>>;

    async fn build_consensus(self, ctx: &BuilderContext<N>) -> eyre::Result<Self::Consensus> {
        Ok(Arc::new(ArbConsensus::new(ctx.chain_spec())))
    }
}
