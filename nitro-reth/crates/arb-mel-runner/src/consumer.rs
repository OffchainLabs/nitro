//! The message consumer interface: the nitro node we push to.
//!
//! In the standalone-RPC-provider model, the MEL runner is the RPC client and the
//! nitro node is the server, serving the provider->node `nitromelconsumer`
//! namespace. This trait is that node from the runner's side: the runner pushes
//! extracted messages (`PushMessages` -> the node's transaction streamer) and
//! notifies it of reorgs (`ReorgedToParentChainBlock` -> the node's reorg
//! detector, which the MEL validator consumes to rewind).
//!
//! Ports `mel.MessageConsumer` plus the reorg callback that nitro's
//! `rpc_runner/rpc_server` exposes alongside it.

use std::sync::Mutex;

use arbos::arbos_types::MessageWithMetadata;
use async_trait::async_trait;

use crate::{MelRunnerError, Result};

/// The consumer (nitro node) the runner pushes extracted data to.
#[async_trait]
pub trait MessageConsumer: Send + Sync {
    /// Pushes `messages` beginning at `first_msg_idx`
    /// (`nitromelconsumer` `PushMessages`).
    async fn push_messages(
        &self,
        first_msg_idx: u64,
        messages: &[MessageWithMetadata],
    ) -> Result<()>;

    /// Notifies the consumer that the runner reorged to `parent_chain_block_number`,
    /// so its MEL validator can rewind (`nitromelconsumer` `ReorgedToParentChainBlock`).
    ///
    /// On the node side this is a no-op when no MEL validator is enabled; from the
    /// runner's side it is always called after a reorg.
    async fn reorged_to_parent_chain_block(&self, parent_chain_block_number: u64) -> Result<()>;
}

/// A [`MessageConsumer`] for tests, with an injectable error.
///
/// Mirrors nitro's `mockMessageConsumer{returnErr}`: set an error with
/// [`Self::set_error`] to make both methods fail. Records the last reorg
/// notification for assertions.
#[derive(Debug, Default)]
pub struct MockMessageConsumer {
    error: Mutex<Option<String>>,
    last_reorg_block: Mutex<Option<u64>>,
}

impl MockMessageConsumer {
    /// Creates a consumer that accepts everything.
    pub fn new() -> Self {
        Self::default()
    }

    /// Injects (or clears) an error returned by both methods.
    pub fn set_error(&self, err: Option<impl Into<String>>) {
        *self.error.lock().unwrap() = err.map(Into::into);
    }

    /// The parent-chain block of the most recent reorg notification, if any.
    pub fn last_reorg_block(&self) -> Option<u64> {
        *self.last_reorg_block.lock().unwrap()
    }

    fn check_error(&self) -> Result<()> {
        match self.error.lock().unwrap().clone() {
            Some(msg) => Err(MelRunnerError::Consumer(msg)),
            None => Ok(()),
        }
    }
}

#[async_trait]
impl MessageConsumer for MockMessageConsumer {
    async fn push_messages(
        &self,
        _first_msg_idx: u64,
        _messages: &[MessageWithMetadata],
    ) -> Result<()> {
        self.check_error()
    }

    async fn reorged_to_parent_chain_block(&self, parent_chain_block_number: u64) -> Result<()> {
        self.check_error()?;
        *self.last_reorg_block.lock().unwrap() = Some(parent_chain_block_number);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn accepts_by_default() {
        let c = MockMessageConsumer::new();
        assert!(c.push_messages(0, &[]).await.is_ok());
        assert!(c.reorged_to_parent_chain_block(42).await.is_ok());
        assert_eq!(c.last_reorg_block(), Some(42));
    }

    #[tokio::test]
    async fn injected_error_fails_both_methods_then_clears() {
        let c = MockMessageConsumer::new();
        c.set_error(Some("push failed"));
        assert!(
            c.push_messages(0, &[])
                .await
                .unwrap_err()
                .to_string()
                .contains("push failed")
        );
        assert!(c.reorged_to_parent_chain_block(1).await.is_err());
        c.set_error(None::<String>);
        assert!(c.push_messages(0, &[]).await.is_ok());
    }
}
