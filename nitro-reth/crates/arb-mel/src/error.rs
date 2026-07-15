use alloy_primitives::B256;

#[derive(Debug, thiserror::Error)]
pub enum MelError {
    #[error("parent chain block hash mismatch: expected {expected}, got {got}")]
    ParentHashMismatch { expected: B256, got: B256 },

    #[error("batch posting reports {reports} exceed batches {batches}")]
    TooManyBatchPostingReports { reports: usize, batches: usize },

    #[error(transparent)]
    Storage(#[from] arb_storage_errors::StorageError),

    #[error("failed to ABI-decode {event} log: {source}")]
    AbiDecode {
        event: &'static str,
        source: alloy_sol_types::Error,
    },

    #[error("sequencer inbox event has non-uint64 {0}")]
    NonU64(&'static str),

    #[error("sequencer batches out of order; after batch {after} got batch {got}")]
    BatchesOutOfOrder { after: u64, got: u64 },

    #[error("message {message_index} data not found")]
    MessageDataNotFound { message_index: B256 },

    #[error("found message {message_index} data with mismatched hash")]
    MessageDataHashMismatch { message_index: B256 },

    #[error("unexpected delayed-message log type")]
    UnexpectedLogType,

    #[error("tx data too short to decode")]
    TxDataTooShort,

    #[error("failed to parse batch posting report: {0}")]
    BatchPostingReportParse(String),

    #[error("failed to fetch sequencer batch data: {0}")]
    SequencerBatchData(&'static str),

    #[error("sequencer message missing L1 header")]
    SequencerMessageTooShort,

    #[error(
        "batch {batch_num} has unsupported authenticated header byte {header_byte:#04x}; node may be out of date"
    )]
    NodeOutOfDate { batch_num: u64, header_byte: u8 },

    #[error(
        "batch {batch_num} requires an unsupported data-availability provider (header byte {header_byte:#04x})"
    )]
    UnsupportedDaHeaderByte { batch_num: u64, header_byte: u8 },

    #[error("unsupported sequencer message encoding: {0}")]
    UnsupportedEncoding(&'static str),

    #[error("failed to decompress batch payload")]
    BatchDecompressionFailed,

    #[error("unknown error")]
    Unknown,
}
