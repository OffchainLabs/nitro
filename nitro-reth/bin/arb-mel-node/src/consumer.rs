use arb_mel_runner::{MessageConsumer, Result};
use arbos_types::MessageWithMetadata;
use tracing::info;

/// Placeholder sink until the real consumer exists: a JSON-RPC client pushing
/// to a nitro node's `nitromelconsumer` namespace (protocol: nitro-private#549).
pub struct LogConsumer;

#[async_trait::async_trait]
impl MessageConsumer for LogConsumer {
    async fn push_messages(
        &self,
        first_msg_idx: u64,
        messages: &[MessageWithMetadata],
    ) -> Result<()> {
        info!(first_msg_idx, count = messages.len(), "extracted messages");
        Ok(())
    }

    async fn reorged_to_parent_chain_block(&self, parent_chain_block_number: u64) -> Result<()> {
        info!(parent_chain_block_number, "reorged to parent chain block");
        Ok(())
    }
}
