use alloy_primitives::Address;

pub mod rlp;
pub mod serialization;

mod incoming_message;
mod message_with_meta;

pub use incoming_message::{
    BatchDataStats, BatchPostingReportFields, DEFAULT_INITIAL_L1_BASE_FEE,
    L1_MESSAGE_TYPE_BATCH_FOR_GAS_ESTIMATION, L1_MESSAGE_TYPE_BATCH_POSTING_REPORT,
    L1_MESSAGE_TYPE_END_OF_BLOCK, L1_MESSAGE_TYPE_ETH_DEPOSIT, L1_MESSAGE_TYPE_INITIALIZE,
    L1_MESSAGE_TYPE_INVALID, L1_MESSAGE_TYPE_L2_FUNDED_BY_L1, L1_MESSAGE_TYPE_L2_MESSAGE,
    L1_MESSAGE_TYPE_ROLLUP_EVENT, L1_MESSAGE_TYPE_SUBMIT_RETRYABLE, L1IncomingMessage,
    L1IncomingMessageHeader, MAX_L2_MESSAGE_SIZE, ParsedInitMessage, get_data_stats,
    invalid_l1_message, legacy_cost_for_stats, parse_batch_posting_report_fields,
    parse_incoming_l1_message, parse_init_message,
};
pub use message_with_meta::{MessageWithMetadata, MessageWithMetadataAndBlockInfo};

/// The well-known ArbOS batch-poster address (`0xa4b0…73657175656e636572`).
///
/// Defined here rather than in `arbos::l1_pricing` so wasm consumers (`arb-mel`)
/// can reach it without depending on `arbos`; `arbos::l1_pricing` re-exports it.
pub const BATCH_POSTER_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
