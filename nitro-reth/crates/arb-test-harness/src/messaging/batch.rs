//! Sequencer-batch message builder (kind 13: BatchPostingReport).

use alloy_primitives::Address;

use crate::{
    error::HarnessError,
    messaging::{kinds, L1Message, MessageBuilder},
};

pub struct BatchBuilder {
    pub batch_index: u64,
    pub batch_poster: Address,
    pub data_size: u64,
    pub l1_block_number: u64,
    pub timestamp: u64,
}

impl MessageBuilder for BatchBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let _ = kinds::KIND_BATCH_POSTING_REPORT;
        Err(HarnessError::NotImplemented {
            what: "BatchBuilder::build (Stage 2 / Agent B)",
        })
    }
}
