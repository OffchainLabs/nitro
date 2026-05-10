//! Multi-message-per-iteration scenario covering arbitrary mixes of L1
//! message kinds (Deposit, SubmitRetryable, SignedL2Tx, UnsignedUserTx,
//! ContractTx) in a single iteration.

use alloy_primitives::{Address, Bytes, B256, U256};
use arb_test_harness::{
    messaging::{
        retryable::{apply_l1_to_l2_alias, RetryableSubmitBuilder},
        signed_tx::{derive_address, AuthorizationItem, L2TxKind, SignedL2TxBuilder},
        ContractTxBuilder, DepositBuilder, MessageBuilder, UnsignedUserTxBuilder,
    },
    scenario::{Scenario, ScenarioSetup},
};
use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

use crate::{
    arbitrary_impls::{build_or_skip, message_step, ArbosVersion, BoundedBytes, FUZZ_GAS_CAP, FUZZ_L1_BASE_FEE},
    shared_nodes::{next_msg_idx, FUZZ_L2_CHAIN_ID},
};

const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65,
    0x71, 0x75, 0x65, 0x6e,
]);

#[derive(Debug, Clone, Serialize)]
pub enum MessageStep {
    Deposit {
        to: Address,
        amount: u64,
    },
    SubmitRetryable {
        l1_sender: Address,
        to: Option<Address>,
        l2_call_value: u64,
        deposit_value: u64,
        max_submission_fee: u64,
        gas_limit: u64,
        max_fee_per_gas: u64,
        fee_refund: Address,
        cvalue_refund: Address,
        data: BoundedBytes<256>,
    },
    SignedTx {
        kind: SignedKind,
        signing_key: [u8; 32],
        to: Option<Address>,
        value: u64,
        gas: u64,
        max_fee: u64,
        priority_fee: u64,
        data: BoundedBytes<256>,
        auth_count: u8,
    },
    UnsignedUserTx {
        from: Address,
        to: Option<Address>,
        value: u64,
        gas: u64,
        max_fee: u64,
        nonce: u8,
        data: BoundedBytes<256>,
    },
    ContractTx {
        from: Address,
        to: Option<Address>,
        value: u64,
        gas: u64,
        max_fee: u64,
        data: BoundedBytes<256>,
    },
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
pub enum SignedKind {
    Legacy,
    Eip2930,
    Eip1559,
    Eip7702,
}

impl<'a> Arbitrary<'a> for SignedKind {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(match u.int_in_range::<u8>(0..=3)? {
            0 => Self::Legacy,
            1 => Self::Eip2930,
            2 => Self::Eip1559,
            _ => Self::Eip7702,
        })
    }
}

impl<'a> Arbitrary<'a> for MessageStep {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        match u.int_in_range::<u8>(0..=4)? {
            0 => Ok(Self::Deposit {
                to: Address::arbitrary(u)?,
                amount: u.int_in_range::<u64>(0..=10_000_000_000_000)?,
            }),
            1 => {
                let mut sk = [0u8; 32];
                u.fill_buffer(&mut sk)?;
                if sk.iter().all(|b| *b == 0) {
                    sk[31] = 1;
                }
                let kind = SignedKind::arbitrary(u)?;
                let has_to = u.arbitrary::<bool>()?;
                let to = if has_to {
                    Some(Address::arbitrary(u)?)
                } else {
                    None
                };
                let auth_count = if kind == SignedKind::Eip7702 {
                    u.int_in_range::<u8>(1..=3)?
                } else {
                    0
                };
                Ok(Self::SignedTx {
                    kind,
                    signing_key: sk,
                    to,
                    value: u.int_in_range::<u64>(0..=1_000_000_000)?,
                    gas: u.int_in_range::<u64>(50_000..=FUZZ_GAS_CAP)?,
                    max_fee: u.int_in_range::<u64>(1..=2_000_000_000)?,
                    priority_fee: u.int_in_range::<u64>(0..=2_000_000_000)?,
                    data: BoundedBytes::<256>::arbitrary(u)?,
                    auth_count,
                })
            }
            2 => Ok(Self::UnsignedUserTx {
                from: Address::arbitrary(u)?,
                to: if u.arbitrary::<bool>()? {
                    Some(Address::arbitrary(u)?)
                } else {
                    None
                },
                value: u.int_in_range::<u64>(0..=1_000_000_000)?,
                gas: u.int_in_range::<u64>(50_000..=FUZZ_GAS_CAP)?,
                // Floor at L2 base fee (~0.1 gwei in genesis) so under-priced
                // txs don't drift between nodes' validation paths. Fuzzing
                // under-pricing semantics is a separate scenario class.
                max_fee: u.int_in_range::<u64>(200_000_000..=2_000_000_000)?,
                // Always nonce 0 for an unsigned-user-tx — we don't track
                // sender nonce growth across iterations.
                nonce: 0,
                data: BoundedBytes::<256>::arbitrary(u)?,
            }),
            3 => Ok(Self::ContractTx {
                from: Address::arbitrary(u)?,
                to: if u.arbitrary::<bool>()? {
                    Some(Address::arbitrary(u)?)
                } else {
                    None
                },
                value: u.int_in_range::<u64>(0..=1_000_000_000)?,
                gas: u.int_in_range::<u64>(50_000..=FUZZ_GAS_CAP)?,
                max_fee: u.int_in_range::<u64>(200_000_000..=2_000_000_000)?,
                data: BoundedBytes::<256>::arbitrary(u)?,
            }),
            _ => Ok(Self::SubmitRetryable {
                l1_sender: Address::arbitrary(u)?,
                to: if u.arbitrary::<bool>()? {
                    Some(Address::arbitrary(u)?)
                } else {
                    None
                },
                l2_call_value: u.int_in_range::<u64>(0..=1_000_000_000_000)?,
                deposit_value: u.int_in_range::<u64>(0..=10_000_000_000_000)?,
                max_submission_fee: u.int_in_range::<u64>(0..=10_000_000_000_000)?,
                gas_limit: u.int_in_range::<u64>(50_000..=FUZZ_GAS_CAP)?,
                max_fee_per_gas: u.int_in_range::<u64>(1..=2_000_000_000)?,
                fee_refund: Address::arbitrary(u)?,
                cvalue_refund: Address::arbitrary(u)?,
                data: BoundedBytes::<256>::arbitrary(u)?,
            }),
        }
    }
}

#[derive(Debug, Clone, Serialize)]
pub struct DiffMultiMsgScenario {
    pub arbos_version: ArbosVersion,
    pub messages: Vec<MessageStep>,
}

impl<'a> Arbitrary<'a> for DiffMultiMsgScenario {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let arbos_version = ArbosVersion::arbitrary(u)?;
        let n = u.int_in_range::<usize>(1..=5)?;
        let mut messages = Vec::with_capacity(n);
        for _ in 0..n {
            messages.push(MessageStep::arbitrary(u)?);
        }
        Ok(Self {
            arbos_version,
            messages,
        })
    }
}

impl DiffMultiMsgScenario {
    pub fn into_scenario(self) -> Option<Scenario> {
        let mut steps = Vec::new();

        // Pre-fund a known address so signed-tx senders that haven't been
        // pre-funded by Deposit messages can still pay gas.
        let pre_fund_idx = next_msg_idx();
        let pre_fund = DepositBuilder {
            from: Address::repeat_byte(0xa1),
            to: Address::repeat_byte(0xa2),
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: pre_fund_idx,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Ok(msg) = pre_fund.build() {
            steps.push(message_step(pre_fund_idx, msg, pre_fund_idx));
        }

        for step in self.messages.iter() {
            self.emit_step(step, &mut steps);
        }

        if steps.len() <= 1 {
            return None;
        }
        Some(Scenario {
            name: "fuzz_multi_msg".into(),
            description: "fuzz-generated multi-message scenario".into(),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps,
        })
    }

    fn emit_step(&self, step: &MessageStep, steps: &mut Vec<arb_test_harness::scenario::ScenarioStep>) {
        match step {
            MessageStep::Deposit { to, amount } => {
                let dep = DepositBuilder {
                    from: Address::repeat_byte(0xa1),
                    to: *to,
                    amount: U256::from(*amount),
                    l1_block_number: 1,
                    timestamp: 1_700_000_000,
                    request_seq: 0,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                if let Some(msg) = build_or_skip(&dep) {
                    let idx = next_msg_idx();
                    steps.push(message_step(idx, msg, idx));
                }
            }
            MessageStep::SubmitRetryable {
                l1_sender,
                to,
                l2_call_value,
                deposit_value,
                max_submission_fee,
                gas_limit,
                max_fee_per_gas,
                fee_refund,
                cvalue_refund,
                data,
            } => {
                // Pre-fund the aliased L1 sender so the retryable can pay.
                let aliased = apply_l1_to_l2_alias(*l1_sender);
                let pre_idx = next_msg_idx();
                let pre = DepositBuilder {
                    from: Address::repeat_byte(0xa1),
                    to: aliased,
                    amount: U256::from(10u128).pow(U256::from(21u64)),
                    l1_block_number: 1,
                    timestamp: 1_700_000_000,
                    request_seq: pre_idx,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                if let Ok(msg) = pre.build() {
                    steps.push(message_step(pre_idx, msg, pre_idx));
                }
                let builder = RetryableSubmitBuilder {
                    l1_sender: *l1_sender,
                    to: to.unwrap_or(Address::ZERO),
                    l2_call_value: U256::from(*l2_call_value),
                    deposit_value: U256::from(*deposit_value),
                    max_submission_fee: U256::from(*max_submission_fee),
                    excess_fee_refund_address: *fee_refund,
                    call_value_refund_address: *cvalue_refund,
                    gas_limit: *gas_limit,
                    max_fee_per_gas: U256::from(*max_fee_per_gas),
                    data: Bytes::from(data.0.clone()),
                    l1_block_number: 1,
                    timestamp: 1_700_000_001,
                    request_id: None,
                };
                if let Some(msg) = build_or_skip(&builder) {
                    let idx = next_msg_idx();
                    steps.push(message_step(idx, msg, idx));
                }
            }
            MessageStep::SignedTx {
                kind,
                signing_key,
                to,
                value,
                gas,
                max_fee,
                priority_fee,
                data,
                auth_count,
            } => {
                let signing_key = B256::from(*signing_key);
                let signer = derive_address(signing_key);
                // Pre-fund the signer.
                let pre_idx = next_msg_idx();
                let pre = DepositBuilder {
                    from: Address::repeat_byte(0xa1),
                    to: signer,
                    amount: U256::from(10u128).pow(U256::from(20u64)),
                    l1_block_number: 1,
                    timestamp: 1_700_000_000,
                    request_seq: pre_idx,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                if let Ok(msg) = pre.build() {
                    steps.push(message_step(pre_idx, msg, pre_idx));
                }
                let kind_l2 = match kind {
                    SignedKind::Legacy => L2TxKind::Legacy,
                    SignedKind::Eip2930 => L2TxKind::Eip2930,
                    SignedKind::Eip1559 => L2TxKind::Eip1559,
                    SignedKind::Eip7702 => {
                        if to.is_none() {
                            return;
                        }
                        L2TxKind::Eip7702
                    }
                };
                let auth_list: Vec<AuthorizationItem> = if *auth_count > 0 {
                    (0..*auth_count)
                        .map(|i| AuthorizationItem {
                            chain_id: FUZZ_L2_CHAIN_ID,
                            address: Address::repeat_byte(0xb0 + i),
                            nonce: 0,
                            signing_key,
                        })
                        .collect()
                } else {
                    Vec::new()
                };
                let max_fee_clamped = (*max_fee as u128).max(FUZZ_L1_BASE_FEE as u128 / 100 + 1);
                let priority_clamped = (*priority_fee as u128).min(max_fee_clamped);
                let builder = SignedL2TxBuilder {
                    chain_id: FUZZ_L2_CHAIN_ID,
                    nonce: 0,
                    to: *to,
                    value: U256::from(*value),
                    data: Bytes::from(data.0.clone()),
                    gas_limit: (*gas).clamp(50_000, FUZZ_GAS_CAP),
                    gas_price: max_fee_clamped,
                    max_fee_per_gas: max_fee_clamped,
                    max_priority_fee_per_gas: priority_clamped,
                    access_list: Vec::new(),
                    authorization_list: auth_list,
                    kind: kind_l2,
                    signing_key,
                    l1_block_number: 1,
                    timestamp: 1_700_000_001,
                    request_id: None,
                    sender: SEQUENCER_ALIAS,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                if let Some(msg) = build_or_skip(&builder) {
                    let idx = next_msg_idx();
                    steps.push(message_step(idx, msg, idx));
                }
            }
            MessageStep::UnsignedUserTx {
                from,
                to,
                value,
                gas,
                max_fee,
                nonce,
                data,
            } => {
                let pre_idx = next_msg_idx();
                let pre = DepositBuilder {
                    from: Address::repeat_byte(0xa1),
                    to: *from,
                    amount: U256::from(10u128).pow(U256::from(20u64)),
                    l1_block_number: 1,
                    timestamp: 1_700_000_000,
                    request_seq: pre_idx,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                if let Ok(msg) = pre.build() {
                    steps.push(message_step(pre_idx, msg, pre_idx));
                }
                let builder = UnsignedUserTxBuilder {
                    from: *from,
                    gas_limit: (*gas).clamp(50_000, FUZZ_GAS_CAP),
                    max_fee_per_gas: U256::from(*max_fee),
                    nonce: *nonce as u64,
                    to: to.unwrap_or(Address::ZERO),
                    value: U256::from(*value),
                    data: Bytes::from(data.0.clone()),
                    l1_block_number: 1,
                    timestamp: 1_700_000_001,
                    request_seq: 0,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                if let Some(msg) = build_or_skip(&builder) {
                    let idx = next_msg_idx();
                    steps.push(message_step(idx, msg, idx));
                }
            }
            MessageStep::ContractTx {
                from,
                to,
                value,
                gas,
                max_fee,
                data,
            } => {
                let builder = ContractTxBuilder {
                    from: *from,
                    gas_limit: (*gas).clamp(50_000, FUZZ_GAS_CAP),
                    max_fee_per_gas: U256::from(*max_fee),
                    to: to.unwrap_or(Address::ZERO),
                    value: U256::from(*value),
                    data: Bytes::from(data.0.clone()),
                    l1_block_number: 1,
                    timestamp: 1_700_000_001,
                    request_seq: 0,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                if let Some(msg) = build_or_skip(&builder) {
                    let idx = next_msg_idx();
                    steps.push(message_step(idx, msg, idx));
                }
            }
        }
    }
}
