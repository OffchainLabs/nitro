use alloy_primitives::{Address, B256};

use crate::MelError;

pub type MelResult<T> = Result<T, MelError>;

/// Time bounds for a sequencer batch.
#[derive(Debug, Clone, Copy, Default)]
pub struct TimeBounds {
    pub min_timestamp: u64,
    pub max_timestamp: u64,
    pub min_block_number: u64,
    pub max_block_number: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DataLocation {
    TxInput,
    SeparateEvent,
    BlobHashes,
}

impl DataLocation {
    /// Decodes the on-chain `dataLocation` byte. Unknown values (e.g. a
    /// force-inclusion batch that carries no data) map to `None`.
    pub fn from_byte(byte: u8) -> Option<Self> {
        match byte {
            0 => Some(DataLocation::TxInput),
            1 => Some(DataLocation::SeparateEvent),
            2 => Some(DataLocation::BlobHashes),
            _ => None,
        }
    }
}

/// A batch parsed from a `SequencerBatchDelivered` log.
pub struct Batch {
    pub block_hash: B256,
    pub parent_chain_block_number: u64,
    pub sequence_number: u64,
    pub before_inbox_acc: B256,
    pub after_inbox_acc: B256,
    pub after_delayed_acc: B256,
    pub after_delayed_count: u64,
    pub time_bounds: TimeBounds,
    pub data_location: Option<DataLocation>,
    pub bridge_address: Address,
    pub raw_log: alloy_rpc_types_eth::Log,
    pub cached_serialized: Option<Vec<u8>>,
}

pub struct BatchMeta {
    pub accumulator: B256,
    pub message_count: u64,
    pub delayed_message_count: u64,
    pub parent_chain_block: u64,
}

#[derive(Debug, PartialEq)]
pub enum BatchSegmentKind {
    L2Message,
    L2MessageBrotli,
    DelayedMessages,
    AdvanceTimestamp,
    AdvanceL1BlockNumber,
    Unknown,
}

impl From<BatchSegmentKind> for u8 {
    fn from(value: BatchSegmentKind) -> Self {
        use BatchSegmentKind::*;
        match value {
            L2Message => 0,
            L2MessageBrotli => 1,
            DelayedMessages => 2,
            AdvanceTimestamp => 3,
            AdvanceL1BlockNumber => 4,
            Unknown => 5,
        }
    }
}

impl From<u8> for BatchSegmentKind {
    fn from(value: u8) -> Self {
        use BatchSegmentKind::*;
        match value {
            0 => L2Message,
            1 => L2MessageBrotli,
            2 => DelayedMessages,
            3 => AdvanceTimestamp,
            4 => AdvanceL1BlockNumber,
            _ => Unknown,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn data_location_from_byte() {
        assert_eq!(DataLocation::from_byte(0), Some(DataLocation::TxInput));
        assert_eq!(
            DataLocation::from_byte(1),
            Some(DataLocation::SeparateEvent)
        );
        assert_eq!(DataLocation::from_byte(2), Some(DataLocation::BlobHashes));
        // Unknown bytes (e.g. a force-inclusion batch with no data) decode to None.
        assert_eq!(DataLocation::from_byte(3), None);
        assert_eq!(DataLocation::from_byte(255), None);
    }
}
