//! Signed-L2-tx wrapping (kind 4) plus delayed/contract/unsigned variants.

use alloy_primitives::Bytes;

use crate::{
    error::HarnessError,
    messaging::{kinds, L1Message, MessageBuilder},
};

pub struct SignedTxBuilder {
    pub rlp_encoded_tx: Bytes,
    pub l1_block_number: u64,
    pub timestamp: u64,
}

impl MessageBuilder for SignedTxBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let _ = kinds::KIND_SIGNED_L2_TX;
        Err(HarnessError::NotImplemented {
            what: "SignedTxBuilder::build (Stage 2 / Agent B)",
        })
    }
}
