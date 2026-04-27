//! ETH-deposit message builder (kinds 3 and 12).

use alloy_primitives::{Address, U256};

use crate::{
    error::HarnessError,
    messaging::{kinds, L1Message, MessageBuilder},
};

pub struct DepositBuilder {
    pub from: Address,
    pub to: Address,
    pub amount: U256,
    pub l1_block_number: u64,
    pub timestamp: u64,
}

impl MessageBuilder for DepositBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let _ = (
            self.from,
            self.to,
            self.amount,
            self.l1_block_number,
            self.timestamp,
            kinds::KIND_ETH_DEPOSIT,
        );
        Err(HarnessError::NotImplemented {
            what: "DepositBuilder::build (Stage 2 / Agent B)",
        })
    }
}
