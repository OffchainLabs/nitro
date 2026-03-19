//! Arbitrum transaction pool builder.

use arb_primitives::ArbTransactionSigned;
use arb_txpool::ArbPooledTransaction;
use reth_chainspec::EthereumHardforks;
use reth_node_builder::{
    components::{PoolBuilder, TxPoolBuilder},
    BuilderContext, FullNodeTypes, NodeTypes,
};
use reth_primitives_traits::NodePrimitives;
use reth_transaction_pool::{
    CoinbaseTipOrdering, EthTransactionPool, TransactionValidationTaskExecutor,
};
use tracing::{debug, info};

/// Arbitrum transaction pool type alias.
pub type ArbTransactionPool<Client, Evm> =
    EthTransactionPool<Client, reth_transaction_pool::blobstore::DiskFileBlobStore, Evm>;

/// Builder for the Arbitrum transaction pool.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbPoolBuilder;

impl<Types, Node, Evm> PoolBuilder<Node, Evm> for ArbPoolBuilder
where
    Types: NodeTypes<
        ChainSpec: EthereumHardforks,
        Primitives: NodePrimitives<SignedTx = ArbTransactionSigned>,
    >,
    Node: FullNodeTypes<Types = Types>,
    Evm: reth_evm::ConfigureEvm<Primitives = Types::Primitives> + Clone + 'static,
{
    type Pool = reth_transaction_pool::Pool<
        TransactionValidationTaskExecutor<
            reth_transaction_pool::EthTransactionValidator<
                Node::Provider,
                ArbPooledTransaction,
                Evm,
            >,
        >,
        CoinbaseTipOrdering<ArbPooledTransaction>,
        reth_transaction_pool::blobstore::DiskFileBlobStore,
    >;

    async fn build_pool(
        self,
        ctx: &BuilderContext<Node>,
        evm_config: Evm,
    ) -> eyre::Result<Self::Pool> {
        let pool_config = ctx.pool_config();

        let blob_store = reth_node_builder::components::create_blob_store_with_cache(ctx, Some(0))?;

        let validator =
            TransactionValidationTaskExecutor::eth_builder(ctx.provider().clone(), evm_config)
                .set_eip4844(false)
                .with_max_tx_input_bytes(ctx.config().txpool.max_tx_input_bytes)
                .with_local_transactions_config(pool_config.local_transactions_config.clone())
                .set_tx_fee_cap(ctx.config().rpc.rpc_tx_fee_cap)
                .with_max_tx_gas_limit(ctx.config().txpool.max_tx_gas_limit)
                .with_minimum_priority_fee(ctx.config().txpool.minimum_priority_fee)
                .with_additional_tasks(ctx.config().txpool.additional_validation_tasks)
                .build_with_tasks(ctx.task_executor().clone(), blob_store.clone());

        let transaction_pool = TxPoolBuilder::new(ctx)
            .with_validator(validator)
            .build_and_spawn_maintenance_task(blob_store, pool_config)?;

        info!(target: "reth::cli", "Transaction pool initialized");
        debug!(target: "reth::cli", "Spawned txpool maintenance task");

        Ok(transaction_pool)
    }
}
