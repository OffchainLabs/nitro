//! Differential fuzz scenario for SubmitRetryable (kind=9) messages.

use alloy_primitives::{Address, Bytes, U256};
use arb_test_harness::{
    messaging::{
        retryable::RetryableSubmitBuilder, DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup},
};
use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

use crate::{
    arbitrary_impls::{message_step, ArbosVersion, BoundedBytes, FUZZ_GAS_CAP, FUZZ_L1_BASE_FEE},
    shared_nodes::{next_msg_idx, FUZZ_L2_CHAIN_ID},
};

#[derive(Debug, Clone, Serialize)]
pub struct DiffRetryableScenario {
    pub arbos_version: ArbosVersion,
    pub l1_sender: Address,
    pub to: Option<Address>,
    pub data: BoundedBytes<512>,
    pub l2_call_value: u64,
    pub deposit_value: u64,
    pub max_submission_fee: u64,
    pub gas_limit: u64,
    pub max_fee_per_gas: u64,
    pub fee_refund_addr: Address,
    pub call_value_refund_addr: Address,
}

impl<'a> Arbitrary<'a> for DiffRetryableScenario {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let arbos_version = ArbosVersion::arbitrary(u)?;
        let l1_sender = Address::arbitrary(u)?;
        let has_to = u.arbitrary::<bool>()?;
        let to = if has_to {
            Some(Address::arbitrary(u)?)
        } else {
            None
        };
        let data = BoundedBytes::<512>::arbitrary(u)?;
        let l2_call_value = u.int_in_range::<u64>(0..=1_000_000_000_000)?;
        let max_submission_fee = u.int_in_range::<u64>(0..=10_000_000_000_000)?;
        let gas_limit = u.int_in_range::<u64>(50_000..=FUZZ_GAS_CAP)?;
        let max_fee_per_gas = u.int_in_range::<u64>(1..=2_000_000_000)?;
        // Cover deposit > total cost (auto-redeem path) and < total cost
        // (zero-out and stay queued).
        let deposit_value = u.int_in_range::<u64>(0..=10_000_000_000_000)?;
        let fee_refund_addr = Address::arbitrary(u)?;
        let call_value_refund_addr = Address::arbitrary(u)?;
        Ok(Self {
            arbos_version,
            l1_sender,
            to,
            data,
            l2_call_value,
            deposit_value,
            max_submission_fee,
            gas_limit,
            max_fee_per_gas,
            fee_refund_addr,
            call_value_refund_addr,
        })
    }
}

impl DiffRetryableScenario {
    pub fn into_scenario(self) -> Option<Scenario> {
        let mut steps = Vec::new();

        let aliased = arb_test_harness::messaging::retryable::apply_l1_to_l2_alias(self.l1_sender);
        let l1_poster = Address::repeat_byte(0xa1);
        for to in [aliased, self.fee_refund_addr, self.call_value_refund_addr] {
            let idx = next_msg_idx();
            let dep = DepositBuilder {
                from: l1_poster,
                to,
                amount: U256::from(10u128).pow(U256::from(20u64)),
                l1_block_number: 1,
                timestamp: 1_700_000_000,
                request_seq: idx,
                base_fee_l1: FUZZ_L1_BASE_FEE,
            };
            if let Ok(msg) = dep.build() {
                steps.push(message_step(idx, msg, idx));
            }
        }

        let builder = RetryableSubmitBuilder {
            l1_sender: self.l1_sender,
            to: self.to.unwrap_or(Address::ZERO),
            l2_call_value: U256::from(self.l2_call_value),
            deposit_value: U256::from(self.deposit_value),
            max_submission_fee: U256::from(self.max_submission_fee),
            excess_fee_refund_address: self.fee_refund_addr,
            call_value_refund_address: self.call_value_refund_addr,
            gas_limit: self.gas_limit,
            max_fee_per_gas: U256::from(self.max_fee_per_gas),
            data: Bytes::from(self.data.0.clone()),
            l1_block_number: 1,
            timestamp: 1_700_000_001,
            request_id: None,
        };
        let msg = builder.build().ok()?;
        let idx = next_msg_idx();
        steps.push(message_step(idx, msg, idx));

        Some(Scenario {
            name: "fuzz_retryable".into(),
            description: "fuzz-generated SubmitRetryable scenario".into(),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps,
        })
    }
}
