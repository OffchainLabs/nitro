//! Arbitrum network builder.

use reth_chainspec::Hardforks;
use reth_network::{primitives::BasicNetworkPrimitives, NetworkHandle, PeersInfo};
use reth_node_builder::{components::NetworkBuilder, BuilderContext, FullNodeTypes, NodeTypes};
use reth_node_types::PrimitivesTy;
use reth_transaction_pool::{PoolPooledTx, PoolTransaction, TransactionPool};
use tracing::info;

/// Builder for Arbitrum P2P networking.
///
/// Delegates to reth's standard network stack.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbNetworkBuilder;

impl<Node, Pool> NetworkBuilder<Node, Pool> for ArbNetworkBuilder
where
    Node: FullNodeTypes<Types: NodeTypes<ChainSpec: Hardforks>>,
    Pool: TransactionPool<Transaction: PoolTransaction<Consensus = reth_node_types::TxTy<Node::Types>>>
        + Unpin
        + 'static,
{
    type Network = NetworkHandle<BasicNetworkPrimitives<PrimitivesTy<Node::Types>, PoolPooledTx<Pool>>>;

    async fn build_network(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
    ) -> eyre::Result<Self::Network> {
        let network = ctx.network_builder().await?;
        let handle = ctx.start_network(network, pool);
        info!(target: "reth::cli", enode=%handle.local_node_record(), "P2P networking initialized");
        Ok(handle)
    }
}
