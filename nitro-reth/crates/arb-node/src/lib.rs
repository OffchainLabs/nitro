//! Arbitrum node builder.
//!
//! Provides the node type definition and component builders
//! needed to launch an Arbitrum reth node.

pub mod args;
pub mod consensus;

use std::sync::Arc;

use arb_primitives::ArbPrimitives;
use reth_chainspec::ChainSpec;
use reth_node_builder::{
    components::{ConsensusBuilder, ExecutorBuilder},
    BuilderContext, FullNodeTypes, NodeTypes,
};

use arb_evm::ArbEvmConfig;

use crate::args::RollupArgs;
use crate::consensus::ArbConsensus;

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

// TODO: Implement NodeTypes once ArbPrimitives-compatible engine/payload types exist.
// EthEngineTypes requires EthPrimitives; need ArbEngineTypes with ArbPrimitives payloads.

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

// TODO: Implement Node trait once ArbPrimitives-compatible pool builder and add-ons are available.
// The Ethereum pool/network/add-ons builders hard-code EthPrimitives and cannot be used directly.
// Need: ArbPoolBuilder (EthPoolTransaction for ArbTransactionSigned), ArbAddOns, ArbEngineValidator.
