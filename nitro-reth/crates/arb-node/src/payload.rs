//! Arbitrum payload service builder.
//!
//! Uses a noop payload service initially. The sequencer drives
//! block building externally via the engine API, so the payload
//! builder service is a pass-through.

use reth_node_builder::{
    components::PayloadServiceBuilder, BuilderContext, FullNodeTypes, NodeTypes,
};
use reth_payload_builder::PayloadBuilderHandle;
use reth_transaction_pool::TransactionPool;
use tracing::info;

/// Builder for the Arbitrum payload service.
///
/// Spawns a noop payload builder service. Block building is driven
/// by the sequencer through engine API calls.
#[derive(Debug, Default, Clone, Copy)]
pub struct ArbPayloadServiceBuilder;

impl<Node, Pool, Evm> PayloadServiceBuilder<Node, Pool, Evm> for ArbPayloadServiceBuilder
where
    Node: FullNodeTypes,
    Pool: TransactionPool + Unpin + 'static,
    Evm: Send + 'static,
{
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        _pool: Pool,
        _evm_config: Evm,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let (service, handle) =
            reth_payload_builder::noop::NoopPayloadBuilderService::<<Node::Types as NodeTypes>::Payload>::new();
        ctx.task_executor()
            .spawn_critical_task("payload builder service", Box::pin(service));
        info!(target: "reth::cli", "Payload builder service initialized (noop)");
        Ok(handle)
    }
}
