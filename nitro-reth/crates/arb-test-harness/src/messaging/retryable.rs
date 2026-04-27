//! Retryable submit / redeem L1 messages (kind 9).

use alloy_primitives::{Address, Bytes, U256};

use crate::{
    error::HarnessError,
    messaging::{kinds, L1Message, MessageBuilder},
};

pub struct RetryableBuilder {
    pub from: Address,
    pub to: Address,
    pub l2_call_value: U256,
    pub deposit: U256,
    pub max_submission_fee: U256,
    pub excess_fee_refund_address: Address,
    pub call_value_refund_address: Address,
    pub gas_limit: u64,
    pub max_fee_per_gas: U256,
    pub data: Bytes,
    pub l1_block_number: u64,
    pub timestamp: u64,
}

impl MessageBuilder for RetryableBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let _ = kinds::KIND_RETRYABLE_TX;
        Err(HarnessError::NotImplemented {
            what: "RetryableBuilder::build (Stage 2 / Agent B)",
        })
    }
}
