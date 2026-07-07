use arbos::arbos_types::MessageWithMetadata;

use crate::{DelayedMessageDB, MelResult, MelState, SequencerMessage};

/// Extracts the L2 messages contained in a parsed sequencer batch.
///
/// TODO: not yet implemented. The full version must walk `seq_msg.segments`,
/// interleaving delayed messages read from `delayed_message_db` whenever the
/// batch's `after_delayed_messages` count runs ahead of
/// `mel_state.delayed_messages_read`, while advancing the per-message
/// timestamp / block-number cursors. For now it returns no messages.
pub fn extract_batch_messages<D: DelayedMessageDB>(
    _mel_state: &MelState,
    _seq_msg: SequencerMessage,
    _delayed_message_db: &D,
) -> MelResult<Vec<MessageWithMetadata>> {
    Ok(Vec::new())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::MockDelayedDb;

    #[test]
    fn extraction_stub_returns_no_messages() -> MelResult<()> {
        // The extractor is a stub; it yields no messages regardless of input.
        let out = extract_batch_messages(
            &MelState::default(),
            SequencerMessage::default(),
            &MockDelayedDb,
        )?;
        assert!(out.is_empty());
        Ok(())
    }
}
