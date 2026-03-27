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
use reth_provider::{
    BlockNumReader, BlockReaderIdExt, DatabaseProviderFactory, HeaderProvider, StateProviderFactory,
};
use reth_rpc_eth_api::EthApiTypes;
use reth_storage_api::{BlockWriter, CanonChainTracker, DBProvider, EthStorage, HistoryWriter};

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
    N::Provider: DatabaseProviderFactory<
            ProviderRW: BlockWriter<
                Block = alloy_consensus::Block<ArbTransactionSigned>,
                Receipt = arb_primitives::ArbReceipt,
            > + reth_storage_api::StateWriter<Receipt = arb_primitives::ArbReceipt>
                            + reth_storage_api::TrieWriter
                            + HistoryWriter
                            + DBProvider,
        > + CanonChainTracker<Header = Header>
        + InMemoryStateAccess<Primitives = ArbPrimitives>,
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
                      + DatabaseProviderFactory<
            ProviderRW: BlockWriter<
                Block = alloy_consensus::Block<ArbTransactionSigned>,
                Receipt = arb_primitives::ArbReceipt,
            > + reth_storage_api::StateWriter<Receipt = arb_primitives::ArbReceipt>
                            + reth_storage_api::TrieWriter
                            + HistoryWriter
                            + DBProvider,
        > + CanonChainTracker<Header = Header>,
    >,
    EthApi: EthApiTypes,
{
    let arb_api = ArbApiHandler::new(ctx.provider().clone());
    ctx.modules.merge_configured(arb_api.into_rpc())?;

    let chain_spec: Arc<ChainSpec> = ctx.config().chain.clone();
    let evm_config = ArbEvmConfig::new(chain_spec.clone());

    // Get the in-memory state handle from the provider (BlockchainProvider).
    let in_memory_state: CanonicalInMemoryState<ArbPrimitives> =
        ctx.provider().canonical_in_memory_state();

    // Batch persist closure: writes multiple ExecutedBlocks in one DB transaction
    // using reth's storage API including history indices.
    let persist_provider = ctx.provider().clone();
    let batch_persist_fn = move |blocks: &[reth_chain_state::ExecutedBlock<ArbPrimitives>]| {
        use alloy_consensus::BlockHeader;
        use reth_storage_api::{StateWriter, TrieWriter, WriteStateInput};

        if blocks.is_empty() {
            return Ok(());
        }

        let first_number = blocks.first().unwrap().recovered_block().number();
        let last_number = blocks.last().unwrap().recovered_block().number();

        let provider_rw = persist_provider
            .database_provider_rw()
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        for block in blocks {
            let block_number = block.recovered_block().sealed_block().header().number();

            // Write block (header, body, senders).
            provider_rw
                .insert_block(block.recovered_block())
                .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

            // Write state changes, receipts, and changesets.
            provider_rw
                .write_state(
                    WriteStateInput::Single {
                        outcome: block.execution_outcome(),
                        block: block_number,
                    },
                    revm_database::OriginalValuesKnown::No,
                    reth_storage_api::StateWriteConfig::default(),
                )
                .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

            // Write hashed state and trie updates.
            let trie_data = block.trie_data();
            provider_rw
                .write_hashed_state(&trie_data.hashed_state)
                .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

            provider_rw
                .write_trie_updates_sorted(&trie_data.trie_updates)
                .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;
        }

        // Build history indices for all blocks in this batch.
        // This populates AccountsHistory and StorageHistory tables,
        // which are required for HistoricalStateProvider to work correctly.
        provider_rw
            .update_history_indices(first_number..=last_number)
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        // Single commit for all blocks.
        provider_rw
            .commit()
            .map_err(|e| arb_rpc::BlockProducerError::Storage(e.to_string()))?;

        // Update canonical head to the last flushed block.
        let last = blocks.last().unwrap();
        let sealed = last.recovered_block().sealed_block();
        let sealed_header =
            reth_primitives_traits::SealedHeader::new(sealed.header().clone(), sealed.hash());
        persist_provider.set_canonical_head(sealed_header);

        Ok(())
    };

    // Buffer threshold from env or default.
    let buffer_threshold = std::env::var("ARB_BLOCK_BUFFER_SIZE")
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(producer::DEFAULT_BUFFER_THRESHOLD);

    // Genesis block number: read from chain spec genesis header.
    // 0 for Arbitrum Sepolia, 22207817 for Arbitrum One.
    let genesis_block_num = chain_spec.genesis_header().number;

    let block_producer = Arc::new(ArbBlockProducer::new(
        ctx.provider().clone(),
        chain_spec,
        evm_config,
        in_memory_state,
        batch_persist_fn,
        buffer_threshold,
    ));

    // Register the nitroexecution namespace on both the regular RPC and auth endpoints.
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
