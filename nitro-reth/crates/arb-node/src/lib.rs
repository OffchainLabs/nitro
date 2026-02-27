//! Arbitrum node builder.
//!
//! Provides the node type definition and component builders
//! needed to launch an Arbitrum reth node.

pub mod args;
pub mod consensus;

use std::sync::Arc;

use reth_chainspec::ChainSpec;
use reth_ethereum_primitives::EthPrimitives;
use reth_node_builder::{
    components::{
        ComponentsBuilder, ConsensusBuilder, ExecutorBuilder, NoopPayloadBuilder,
        NodeComponentsBuilder,
    },
    BuilderContext, FullNodeTypes, Node, NodeAdapter, NodeTypes,
};
use reth_node_ethereum::{
    EthEngineTypes, EthereumAddOns, EthereumEthApiBuilder, EthereumEngineValidatorBuilder,
    EthereumNetworkBuilder, EthereumPoolBuilder,
};
use reth_storage_api::EthStorage;

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

impl NodeTypes for ArbNode {
    type Primitives = EthPrimitives;
    type ChainSpec = ChainSpec;
    type Storage = EthStorage;
    type Payload = EthEngineTypes;
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

/// Builder for the Arbitrum consensus component.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbConsensusBuilder;

impl<N> ConsensusBuilder<N> for ArbConsensusBuilder
where
    N: FullNodeTypes<Types: NodeTypes<ChainSpec = ChainSpec, Primitives = EthPrimitives>>,
{
    type Consensus = Arc<ArbConsensus<ChainSpec>>;

    async fn build_consensus(self, ctx: &BuilderContext<N>) -> eyre::Result<Self::Consensus> {
        Ok(Arc::new(ArbConsensus::new(ctx.chain_spec())))
    }
}

/// Component types for the Arbitrum node.
pub type ArbNodeComponents<N> = ComponentsBuilder<
    N,
    EthereumPoolBuilder,
    NoopPayloadBuilder,
    EthereumNetworkBuilder,
    ArbExecutorBuilder,
    ArbConsensusBuilder,
>;

impl<N> Node<N> for ArbNode
where
    N: FullNodeTypes<Types = Self>,
{
    type ComponentsBuilder = ArbNodeComponents<N>;

    type AddOns = EthereumAddOns<
        NodeAdapter<N, <Self::ComponentsBuilder as NodeComponentsBuilder<N>>::Components>,
        EthereumEthApiBuilder,
        EthereumEngineValidatorBuilder,
    >;

    fn components_builder(&self) -> Self::ComponentsBuilder {
        ComponentsBuilder::default()
            .node_types::<N>()
            .executor(ArbExecutorBuilder)
            .consensus(ArbConsensusBuilder)
            .pool(EthereumPoolBuilder::default())
            .network(EthereumNetworkBuilder::default())
            .noop_payload()
    }

    fn add_ons(&self) -> Self::AddOns {
        EthereumAddOns::default()
    }
}
