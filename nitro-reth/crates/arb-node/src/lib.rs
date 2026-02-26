//! Arbitrum node builder.
//!
//! Provides the node type definition and component builders
//! needed to launch an Arbitrum reth node.

pub mod consensus;

use reth_chainspec::ChainSpec;
use reth_ethereum_primitives::EthPrimitives;
use reth_node_builder::{
    components::ExecutorBuilder, BuilderContext, FullNodeTypes, NodeTypes,
};

use arb_evm::ArbEvmConfig;

/// Arbitrum node configuration.
#[derive(Debug, Clone, Default)]
pub struct ArbNode {
    /// Whether this node operates as a sequencer.
    pub sequencer: bool,
}

impl ArbNode {
    /// Create a new Arbitrum node configuration.
    pub fn new(sequencer: bool) -> Self {
        Self { sequencer }
    }
}

/// Builder for the Arbitrum EVM executor component.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbExecutorBuilder;

impl<N> ExecutorBuilder<N> for ArbExecutorBuilder
where
    N: FullNodeTypes<Types: NodeTypes<ChainSpec = ChainSpec, Primitives = EthPrimitives>>,
{
    type EVM = ArbEvmConfig;

    async fn build_evm(self, ctx: &BuilderContext<N>) -> eyre::Result<Self::EVM> {
        Ok(ArbEvmConfig::new(ctx.chain_spec()))
    }
}
