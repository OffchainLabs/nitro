//! Arbitrum node builder.
//!
//! Provides the node type definition and component builders
//! needed to launch an Arbitrum reth node.

pub mod addons;
pub mod args;
pub mod consensus;
pub mod engine;
pub mod genesis;
pub mod launcher;
pub mod network;
pub mod payload;
pub mod pool;
pub mod producer;
pub mod validator;

use std::sync::Arc;

use alloy_consensus::Header;
use arb_payload::ArbEngineTypes;
use arb_primitives::{ArbPrimitives, ArbTransactionSigned};
use arb_rpc::{
    ArbApiHandler, ArbApiServer, ArbEthApiBuilder, NitroExecutionApiServer, NitroExecutionHandler,
};
use reth_chain_state::CanonicalInMemoryState;
use reth_chainspec::ChainSpec;
use reth_node_builder::{
    components::{ComponentsBuilder, ConsensusBuilder, ExecutorBuilder},
    rpc::{BasicEngineApiBuilder, BasicEngineValidatorBuilder, RpcAddOns, RpcContext},
    BuilderContext, FullNodeComponents, FullNodeTypes, Node, NodeAdapter, NodeTypes,
};
use reth_provider::{BlockNumReader, BlockReaderIdExt, HeaderProvider, StateProviderFactory};
use reth_rpc_eth_api::EthApiTypes;
use reth_storage_api::{CanonChainTracker, EthStorage};

use arb_evm::ArbEvmConfig;

use crate::{
    addons::ArbPayloadValidatorBuilder,
    args::RollupArgs,
    consensus::ArbConsensus,
    network::ArbNetworkBuilder,
    payload::ArbPayloadServiceBuilder,
    pool::ArbPoolBuilder,
    producer::{ArbBlockProducer, InMemoryStateAccess},
};

/// Arbitrum RPC add-ons type alias.
pub type ArbAddOns<N> = RpcAddOns<
    N,
    ArbEthApiBuilder,
    ArbPayloadValidatorBuilder,
    BasicEngineApiBuilder<ArbPayloadValidatorBuilder>,
    BasicEngineValidatorBuilder<ArbPayloadValidatorBuilder>,
>;

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

    /// Returns a [`ComponentsBuilder`] configured for Arbitrum.
    pub fn components<N>() -> ComponentsBuilder<
        N,
        ArbPoolBuilder,
        ArbPayloadServiceBuilder,
        ArbNetworkBuilder,
        ArbExecutorBuilder,
        ArbConsensusBuilder,
    >
    where
        N: FullNodeTypes<Types: NodeTypes<ChainSpec = ChainSpec, Primitives = ArbPrimitives>>,
    {
        ComponentsBuilder::default()
            .node_types::<N>()
            .pool(ArbPoolBuilder)
            .executor(ArbExecutorBuilder)
            .payload(ArbPayloadServiceBuilder)
            .network(ArbNetworkBuilder)
            .consensus(ArbConsensusBuilder)
    }
}

impl NodeTypes for ArbNode {
    type Primitives = ArbPrimitives;
    type ChainSpec = ChainSpec;
    type Storage = ArbStorage;
    type Payload = ArbEngineTypes;
}

impl<N> Node<N> for ArbNode
where
    N: FullNodeTypes<Types = Self>,
    N::Provider:
        CanonChainTracker<Header = Header> + InMemoryStateAccess<Primitives = ArbPrimitives>,
{
    type ComponentsBuilder = ComponentsBuilder<
        N,
        ArbPoolBuilder,
        ArbPayloadServiceBuilder,
        ArbNetworkBuilder,
        ArbExecutorBuilder,
        ArbConsensusBuilder,
    >;

    type AddOns =
        ArbAddOns<
            NodeAdapter<
                N,
                <Self::ComponentsBuilder as reth_node_builder::components::NodeComponentsBuilder<
                    N,
                >>::Components,
            >,
        >;

    fn components_builder(&self) -> Self::ComponentsBuilder {
        Self::components()
    }

    fn add_ons(&self) -> Self::AddOns {
        RpcAddOns::new(
            ArbEthApiBuilder::default(),
            ArbPayloadValidatorBuilder,
            BasicEngineApiBuilder::default(),
            BasicEngineValidatorBuilder::default(),
            Default::default(),
        )
        .extend_rpc_modules(register_arb_rpc)
    }
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

/// Registers the `arb_` and `nitroexecution_` RPC namespaces.
fn register_arb_rpc<N, EthApi>(ctx: RpcContext<'_, N, EthApi>) -> eyre::Result<()>
where
    N: FullNodeComponents<
        Types: NodeTypes<ChainSpec = ChainSpec, Primitives = ArbPrimitives>,
        Provider: BlockNumReader
                      + BlockReaderIdExt
                      + HeaderProvider
                      + StateProviderFactory
                      + InMemoryStateAccess<Primitives = ArbPrimitives>
                      + CanonChainTracker<Header = Header>,
    >,
    EthApi: EthApiTypes,
{
    let arb_api = ArbApiHandler::new(ctx.provider().clone());
    ctx.modules.merge_configured(arb_api.into_rpc())?;

    let chain_spec: Arc<ChainSpec> = ctx.config().chain.clone();
    let evm_config = ArbEvmConfig::new(chain_spec.clone());

    let in_memory_state: CanonicalInMemoryState<ArbPrimitives> =
        ctx.provider().canonical_in_memory_state();

    let genesis_block_num = chain_spec.genesis_header().number;

    let block_producer = Arc::new(ArbBlockProducer::new(
        ctx.provider().clone(),
        chain_spec,
        evm_config,
        in_memory_state,
    ));

    let nitro_exec =
        NitroExecutionHandler::new(ctx.provider().clone(), block_producer, genesis_block_num);
    let nitro_rpc = nitro_exec.into_rpc();
    ctx.modules.merge_configured(nitro_rpc.clone())?;
    ctx.auth_module.merge_auth_methods(nitro_rpc)?;

    Ok(())
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
