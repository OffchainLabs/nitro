//! Arbitrum node builder.
//!
//! Provides the node type definition and component builders
//! needed to launch an Arbitrum reth node.

pub mod addons;
pub mod args;
pub mod consensus;
pub mod genesis;
pub mod network;
pub mod payload;
pub mod pool;
pub mod producer;
pub mod validator;

use std::sync::Arc;

use arb_payload::ArbEngineTypes;
use arb_primitives::{ArbPrimitives, ArbTransactionSigned};
use arb_rpc::{ArbApiHandler, ArbApiServer, ArbEthApiBuilder, NitroExecutionApiServer, NitroExecutionHandler};
use reth_chainspec::ChainSpec;
use reth_node_builder::{
    components::{ComponentsBuilder, ConsensusBuilder, ExecutorBuilder},
    rpc::{BasicEngineApiBuilder, BasicEngineValidatorBuilder, RpcAddOns, RpcContext},
    BuilderContext, FullNodeComponents, FullNodeTypes, Node, NodeAdapter, NodeTypes,
};
use alloy_consensus::Header;
use reth_provider::{
    BlockNumReader, BlockReaderIdExt, DatabaseProviderFactory, HeaderProvider, StateProviderFactory,
};
use reth_rpc_eth_api::EthApiTypes;
use reth_storage_api::{BlockWriter, CanonChainTracker, DBProvider, EthStorage};

use arb_evm::ArbEvmConfig;

use crate::addons::ArbPayloadValidatorBuilder;
use crate::args::RollupArgs;
use crate::consensus::ArbConsensus;
use crate::network::ArbNetworkBuilder;
use crate::payload::ArbPayloadServiceBuilder;
use crate::pool::ArbPoolBuilder;
use crate::producer::ArbBlockProducer;

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
        N: FullNodeTypes<
            Types: NodeTypes<ChainSpec = ChainSpec, Primitives = ArbPrimitives>,
        >,
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
    N::Provider: DatabaseProviderFactory<
            ProviderRW: BlockWriter<
                Block = alloy_consensus::Block<ArbTransactionSigned>,
                Receipt = arb_primitives::ArbReceipt,
            > + reth_storage_api::StateWriter<Receipt = arb_primitives::ArbReceipt>
              + reth_storage_api::TrieWriter
              + DBProvider,
        > + CanonChainTracker<Header = Header>,
{
    type ComponentsBuilder = ComponentsBuilder<
        N,
        ArbPoolBuilder,
        ArbPayloadServiceBuilder,
        ArbNetworkBuilder,
        ArbExecutorBuilder,
        ArbConsensusBuilder,
    >;

    type AddOns = ArbAddOns<NodeAdapter<N, <Self::ComponentsBuilder as reth_node_builder::components::NodeComponentsBuilder<N>>::Components>>;

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
            + DatabaseProviderFactory<
                ProviderRW: BlockWriter<
                    Block = alloy_consensus::Block<ArbTransactionSigned>,
                    Receipt = arb_primitives::ArbReceipt,
                > + reth_storage_api::StateWriter<Receipt = arb_primitives::ArbReceipt>
                  + reth_storage_api::TrieWriter
                  + DBProvider,
            > + CanonChainTracker<Header = Header>,
    >,
    EthApi: EthApiTypes,
{
    let arb_api = ArbApiHandler::new(ctx.provider().clone());
    ctx.modules.merge_configured(arb_api.into_rpc())?;

    // Create the block producer with a persistence closure.
    let chain_spec: Arc<ChainSpec> = ctx.config().chain.clone();
    let evm_config = ArbEvmConfig::new(chain_spec.clone());
    let persist_provider = ctx.provider().clone();
    let persist_fn = move |sealed: &reth_primitives_traits::SealedBlock<
        alloy_consensus::Block<ArbTransactionSigned>,
    >,
                           receipts: Vec<arb_primitives::ArbReceipt>,
                           bundle_state: revm::database::BundleState,
                           hashed_state: reth_trie_common::HashedPostState,
                           trie_updates: reth_trie_common::updates::TrieUpdates| {
        use alloy_consensus::BlockHeader;
        use reth_execution_types::BlockExecutionOutput;
        use reth_primitives_traits::RecoveredBlock;
        use reth_storage_api::{StateWriter, TrieWriter};
        use alloy_evm::block::BlockExecutionResult;

        let block_number = sealed.header().number();
        let recovered = RecoveredBlock::new_sealed(sealed.clone(), vec![]);

        let provider_rw = persist_provider
            .database_provider_rw()
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        // Write the block (header, body, senders).
        provider_rw
            .insert_block(&recovered)
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        // Write state changes and receipts to plain state tables.
        let exec_output = BlockExecutionOutput {
            state: bundle_state,
            result: BlockExecutionResult {
                receipts,
                requests: Default::default(),
                gas_used: sealed.header().gas_used() as u64,
                blob_gas_used: 0,
            },
        };

        provider_rw
            .write_state(
                reth_storage_api::WriteStateInput::Single {
                    outcome: &exec_output,
                    block: block_number,
                },
                revm_database::OriginalValuesKnown::No,
                reth_storage_api::StateWriteConfig::default(),
            )
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        // Write hashed state (HashedAccounts, HashedStorages tables).
        // Required for state_by_block_hash() to see updated state.
        provider_rw
            .write_hashed_state(&hashed_state.into_sorted())
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        // Write trie intermediate nodes for incremental trie updates.
        provider_rw
            .write_trie_updates(trie_updates)
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        provider_rw
            .commit()
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        // Update the in-memory canonical head so header lookups find the new block.
        let sealed_header = reth_primitives_traits::SealedHeader::new(
            sealed.header().clone(),
            sealed.hash(),
        );
        persist_provider.set_canonical_head(sealed_header);

        Ok(())
    };

    // Genesis block number: read from chain spec genesis header.
    // 0 for Arbitrum Sepolia, 22207817 for Arbitrum One.
    let genesis_block_num = chain_spec.genesis_header().number;

    let block_producer = Arc::new(ArbBlockProducer::new(
        ctx.provider().clone(),
        chain_spec,
        evm_config,
        persist_fn,
    ));

    // Register the nitroexecution namespace on both the regular RPC and auth endpoints.
    // The consensus layer connects to the auth RPC port with JWT authentication.
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
