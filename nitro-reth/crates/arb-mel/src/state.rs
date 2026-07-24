//! MEL state-transition operations over [`MelState`].

use arb_mel_types::{DelayedInboxMessage, MelState};
use arbos::types::MessageWithMetadata;

use crate::{DelayedMessageDB, MelResult};

/// TODO: not yet implemented. Will fold `message` into the delayed-inbox
/// accumulator and advance `delayed_messages_read`.
pub fn accumulate_delayed_message(
    _state: &mut MelState,
    _message: &DelayedInboxMessage,
) -> MelResult<()> {
    Ok(())
}

/// TODO: not yet implemented. Will fold `message` into the message accumulator
/// that seeds the L2 chain.
pub fn accumulate_message(_state: &mut MelState, _message: &MessageWithMetadata) -> MelResult<()> {
    Ok(())
}

/// TODO: not yet implemented. Will drain any unread delayed messages into the
/// inbox accumulator when transitioning MEL versions.
pub fn move_unread_delayed_messages_to_inbox_accumulator(
    _state: &mut MelState,
    _delayed_msg_db: &impl DelayedMessageDB,
) -> MelResult<()> {
    Ok(())
}
