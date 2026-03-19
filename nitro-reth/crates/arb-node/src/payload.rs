//! Arbitrum payload service builder.
//!
//! Spawns a minimal payload service that handles Subscribe commands
//! by returning a valid broadcast receiver. Block building is driven
//! externally by the sequencer via RPC, not by the payload builder.

use futures_util::{ready, StreamExt};
use reth_node_builder::{
    components::PayloadServiceBuilder, BuilderContext, FullNodeTypes, NodeTypes,
};
use reth_payload_builder::{PayloadBuilderHandle, PayloadServiceCommand};
use reth_payload_primitives::{PayloadBuilderAttributes, PayloadTypes};
use reth_transaction_pool::TransactionPool;
use std::{
    future::Future,
    pin::Pin,
    task::{Context, Poll},
};
use tokio::sync::{broadcast, mpsc};
use tokio_stream::wrappers::UnboundedReceiverStream;
use tracing::info;

/// Payload builder service that handles Subscribe commands properly.
///
/// The noop service drops Subscribe senders, which causes reth's engine
/// tree to fail with "ChannelClosed". This service keeps a broadcast
/// channel alive and responds to Subscribe with a valid receiver.
struct ArbPayloadService<T: PayloadTypes> {
    command_rx: UnboundedReceiverStream<PayloadServiceCommand<T>>,
    events_tx: broadcast::Sender<reth_payload_builder_primitives::Events<T>>,
}

impl<T: PayloadTypes> ArbPayloadService<T> {
    fn new() -> (Self, PayloadBuilderHandle<T>) {
        let (service_tx, command_rx) = mpsc::unbounded_channel();
        let (events_tx, _) = broadcast::channel(16);
        (
            Self {
                command_rx: UnboundedReceiverStream::new(command_rx),
                events_tx,
            },
            PayloadBuilderHandle::new(service_tx),
        )
    }
}

impl<T: PayloadTypes> Future for ArbPayloadService<T> {
    type Output = ();

    fn poll(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Self::Output> {
        let this = self.get_mut();
        loop {
            let Some(cmd) = ready!(this.command_rx.poll_next_unpin(cx)) else {
                return Poll::Ready(());
            };
            match cmd {
                PayloadServiceCommand::BuildNewPayload(attr, tx) => {
                    let id = attr.payload_id();
                    let _ = tx.send(Ok(id));
                }
                PayloadServiceCommand::BestPayload(_, tx) => {
                    let _ = tx.send(None);
                }
                PayloadServiceCommand::PayloadTimestamp(_, tx) => {
                    let _ = tx.send(None);
                }
                PayloadServiceCommand::Resolve(_, _, tx) => {
                    let _ = tx.send(None);
                }
                PayloadServiceCommand::Subscribe(tx) => {
                    let rx = this.events_tx.subscribe();
                    let _ = tx.send(rx);
                }
            }
        }
    }
}

/// Builder for the Arbitrum payload service.
///
/// Spawns a minimal payload builder service. Block building is driven
/// by the sequencer through RPC calls, not through the payload service.
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
        let (service, handle) = ArbPayloadService::<<Node::Types as NodeTypes>::Payload>::new();
        ctx.task_executor()
            .spawn_critical_task("payload builder service", Box::pin(service));
        info!(target: "reth::cli", "Payload builder service initialized");
        Ok(handle)
    }
}
