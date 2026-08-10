//! MEL state-transition operations over [`MelState`].

use alloy_primitives::{B256, keccak256};
use arb_mel_types::{DelayedInboxMessage, MelState};
use arbos_types::MessageWithMetadata;

use crate::{DelayedMessageDB, MelError, MelResult};

fn chain_accumulator(prev: B256, msg_hash: B256) -> B256 {
    let mut preimage = [0u8; 64];
    preimage[..32].copy_from_slice(prev.as_slice());
    preimage[32..].copy_from_slice(msg_hash.as_slice());
    keccak256(preimage)
}

// TODO: not yet at parity with Nitro's AccumulateDelayedMessage. Missing
// initMsg capture (when delayed_messages_seen == 0) and preimage recording
// that the pour/pop FIFO and MEL validation depend on.
pub fn accumulate_delayed_message(
    state: &mut MelState,
    message: &DelayedInboxMessage,
) -> MelResult<()> {
    state.delayed_message_inbox_acc =
        chain_accumulator(state.delayed_message_inbox_acc, message.abi_hash());
    Ok(())
}

// TODO: not yet at parity with Nitro's AccumulateMessage. Missing preimage
// recording required for MEL validation-mode replay.
pub fn accumulate_message(state: &mut MelState, message: &MessageWithMetadata) -> MelResult<()> {
    state.local_msg_accumulator =
        chain_accumulator(state.local_msg_accumulator, message.abi_hash());
    Ok(())
}

pub fn move_unread_delayed_messages_to_inbox_accumulator(
    state: &mut MelState,
    delayed_msg_db: &impl DelayedMessageDB,
) -> MelResult<()> {
    let mut unread = Vec::new();
    for i in state.delayed_messages_read..state.delayed_messages_seen {
        let msg = delayed_msg_db
            .read_delayed_message(state, i)
            .map_err(|e| MelError::DelayedAccumulatorCreation(e.to_string()))?
            .ok_or_else(|| {
                MelError::DelayedAccumulatorCreation(format!(
                    "no delayed message in db at index {i}"
                ))
            })?;
        unread.push(msg);
    }
    if state.delayed_message_inbox_acc != B256::ZERO
        || state.delayed_message_outbox_acc != B256::ZERO
    {
        return Err(MelError::NonZeroDelayedAccumulator {
            inbox: state.delayed_message_inbox_acc,
            outbox: state.delayed_message_outbox_acc,
        });
    }
    for msg in &unread {
        accumulate_delayed_message(state, msg)?;
    }
    Ok(())
}
