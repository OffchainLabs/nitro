//! Multi-message-per-iteration scenario covering arbitrary mixes of L1
//! message kinds (Deposit, SubmitRetryable, SignedL2Tx, UnsignedUserTx,
//! ContractTx) in a single iteration.

use std::sync::atomic::{AtomicU64, Ordering};

use alloy_primitives::{Address, Bytes, B256, U256};

/// Per-process counter for `request_seq` on Arbitrum-internal txs
/// (UnsignedUserTx / ContractTx) so each submission's derived tx hash is
/// unique even when the user-facing fields collide across iterations.
/// Distinct from `GLOBAL_MSG_IDX` (which tracks L1 message index and must
/// match Nitro's expected sequence).
static UNIQUE_REQUEST_SEQ: AtomicU64 = AtomicU64::new(1);
fn next_request_seq() -> u64 {
    UNIQUE_REQUEST_SEQ.fetch_add(1, Ordering::SeqCst)
}
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
    arbitrary_impls::{
        build_or_skip, message_step, ArbosVersion, BoundedBytes, FUZZ_GAS_CAP, FUZZ_L1_BASE_FEE,
    },
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
    /// Read-only call into ArbWasm precompile (programVersion / programInitGas
    /// / programMemoryFootprint / programTimeLeft / programCodehash). These
    /// methods all do codehash + StylusParams reads — historically bug-dense
    /// for gas accounting (see commit 77a226b).
    ArbWasmRead {
        method: ArbWasmReadMethod,
        target: Address,
        signing_key: [u8; 32],
        gas: u64,
        max_fee: u64,
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

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
pub enum ArbWasmReadMethod {
    StylusVersion,
    InkPrice,
    MaxStackDepth,
    FreePages,
    PageGas,
    PageRamp,
    PageLimit,
    MinInitGas,
    InitCostScalar,
    ExpiryDays,
    KeepaliveDays,
    BlockCacheSize,
    ActivationGas,
    CodehashVersion,
    CodehashAsmSize,
    ProgramVersion,
    ProgramInitGas,
    ProgramMemoryFootprint,
    ProgramTimeLeft,
}

impl<'a> Arbitrary<'a> for ArbWasmReadMethod {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(match u.int_in_range::<u8>(0..=18)? {
            0 => Self::StylusVersion,
            1 => Self::InkPrice,
            2 => Self::MaxStackDepth,
            3 => Self::FreePages,
            4 => Self::PageGas,
            5 => Self::PageRamp,
            6 => Self::PageLimit,
            7 => Self::MinInitGas,
            8 => Self::InitCostScalar,
            9 => Self::ExpiryDays,
            10 => Self::KeepaliveDays,
            11 => Self::BlockCacheSize,
            12 => Self::ActivationGas,
            13 => Self::CodehashVersion,
            14 => Self::CodehashAsmSize,
            15 => Self::ProgramVersion,
            16 => Self::ProgramInitGas,
            17 => Self::ProgramMemoryFootprint,
            _ => Self::ProgramTimeLeft,
        })
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ArbWasmArgKind {
    NoArg,
    Codehash,
    Address,
}

impl ArbWasmReadMethod {
    pub fn selector(&self) -> [u8; 4] {
        // keccak256("<sig>")[0..4]
        match self {
            Self::StylusVersion => [0xa9, 0x96, 0xe0, 0xc2],
            Self::InkPrice => [0xd1, 0xc1, 0x7a, 0xbc],
            Self::MaxStackDepth => [0x8c, 0xcf, 0xaa, 0x70],
            Self::FreePages => [0x44, 0x90, 0xc1, 0x9d],
            Self::PageGas => [0x7a, 0xf4, 0xba, 0x49],
            Self::PageRamp => [0x11, 0xc8, 0x2a, 0xe8],
            Self::PageLimit => [0x97, 0x86, 0xf9, 0x6e],
            Self::MinInitGas => [0x99, 0xd0, 0xb3, 0x8d],
            Self::InitCostScalar => [0x5f, 0xc9, 0x4c, 0x0b],
            Self::ExpiryDays => [0x30, 0x9f, 0x65, 0x55],
            Self::KeepaliveDays => [0x0a, 0x93, 0x64, 0x55],
            Self::BlockCacheSize => [0x7a, 0xf6, 0xe8, 0x19],
            Self::ActivationGas => [0x22, 0x78, 0xc2, 0x78],
            Self::CodehashVersion => [0xd7, 0x0c, 0x0c, 0xa7],
            Self::CodehashAsmSize => [0x40, 0x89, 0x26, 0x7f],
            Self::ProgramVersion => [0xcc, 0x8f, 0x4e, 0x88],
            Self::ProgramInitGas => [0x62, 0xb6, 0x88, 0xaa],
            Self::ProgramMemoryFootprint => [0xae, 0xf3, 0x6b, 0xe3],
            Self::ProgramTimeLeft => [0xc7, 0x75, 0xa6, 0x2a],
        }
    }

    pub fn arg_kind(&self) -> ArbWasmArgKind {
        match self {
            Self::StylusVersion
            | Self::InkPrice
            | Self::MaxStackDepth
            | Self::FreePages
            | Self::PageGas
            | Self::PageRamp
            | Self::PageLimit
            | Self::MinInitGas
            | Self::InitCostScalar
            | Self::ExpiryDays
            | Self::KeepaliveDays
            | Self::BlockCacheSize
            | Self::ActivationGas => ArbWasmArgKind::NoArg,
            Self::CodehashVersion | Self::CodehashAsmSize => ArbWasmArgKind::Codehash,
            Self::ProgramVersion
            | Self::ProgramInitGas
            | Self::ProgramMemoryFootprint
            | Self::ProgramTimeLeft => ArbWasmArgKind::Address,
        }
    }
}

impl<'a> Arbitrary<'a> for MessageStep {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        match u.int_in_range::<u8>(0..=5)? {
            0 => Ok(Self::Deposit {
                to: Address::arbitrary(u)?,
                amount: u.int_in_range::<u64>(0..=10_000_000_000_000)?,
            }),
            5 => {
                let mut sk = [0u8; 32];
                u.fill_buffer(&mut sk)?;
                if sk.iter().all(|b| *b == 0) {
                    sk[31] = 1;
                }
                Ok(Self::ArbWasmRead {
                    method: ArbWasmReadMethod::arbitrary(u)?,
                    target: Address::arbitrary(u)?,
                    signing_key: sk,
                    gas: u.int_in_range::<u64>(50_000..=FUZZ_GAS_CAP)?,
                    max_fee: u.int_in_range::<u64>(200_000_000..=2_000_000_000)?,
                })
            }
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

    fn emit_step(
        &self,
        step: &MessageStep,
        steps: &mut Vec<arb_test_harness::scenario::ScenarioStep>,
    ) {
        match step {
            MessageStep::ArbWasmRead {
                method,
                target,
                signing_key,
                gas,
                max_fee,
            } => {
                let signing_key_b256 = B256::from(*signing_key);
                let signer = derive_address(signing_key_b256);
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
                // ArbWasm precompile address.
                let arbwasm = Address::new([
                    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
                ]);
                let mut data = Vec::with_capacity(36);
                data.extend_from_slice(&method.selector());
                match method.arg_kind() {
                    ArbWasmArgKind::NoArg => {}
                    ArbWasmArgKind::Address => {
                        let mut padded = [0u8; 32];
                        padded[12..].copy_from_slice(target.as_slice());
                        data.extend_from_slice(&padded);
                    }
                    ArbWasmArgKind::Codehash => {
                        // Use target bytes (left-padded) as the codehash. The
                        // actual hash doesn't need to resolve to an active
                        // program — only the gas accounting through the read
                        // path matters for differential testing.
                        let mut hash = [0u8; 32];
                        hash[12..].copy_from_slice(target.as_slice());
                        // Also fill the first 12 bytes with a deterministic
                        // pattern so the codehash isn't always EVM-address-shaped.
                        for (i, b) in hash[..12].iter_mut().enumerate() {
                            *b = target.as_slice()[i % 20];
                        }
                        data.extend_from_slice(&hash);
                    }
                }
                let max_fee_clamped = (*max_fee as u128).max(FUZZ_L1_BASE_FEE as u128 / 100 + 1);
                let builder = SignedL2TxBuilder {
                    chain_id: FUZZ_L2_CHAIN_ID,
                    nonce: 0,
                    to: Some(arbwasm),
                    value: U256::ZERO,
                    data: Bytes::from(data),
                    gas_limit: (*gas).clamp(50_000, FUZZ_GAS_CAP),
                    gas_price: max_fee_clamped,
                    max_fee_per_gas: max_fee_clamped,
                    max_priority_fee_per_gas: 0,
                    access_list: Vec::new(),
                    authorization_list: Vec::new(),
                    kind: L2TxKind::Eip1559,
                    signing_key: signing_key_b256,
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
                // Use a unique request_seq per submission so the
                // derived ArbitrumUnsignedTx hash is unique even when
                // (from, nonce, gas, fee, to, value, data) collide
                // across iterations. Otherwise the same hash gets
                // sequenced into multiple different L2 blocks on each
                // node, and `eth_getTransactionReceipt(hash)` returns
                // whichever block each node disambiguated to — yielding
                // a phantom `effective_gas_price` divergence even though
                // both nodes are correct.
                let unique_seq = next_request_seq();
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
                    request_seq: unique_seq,
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
                // Unique request_seq so ArbitrumContractTx hash differs
                // across iterations even when other fields collide (see
                // the matching comment on UnsignedUserTxBuilder above).
                let unique_seq = next_request_seq();
                let builder = ContractTxBuilder {
                    from: *from,
                    gas_limit: (*gas).clamp(50_000, FUZZ_GAS_CAP),
                    max_fee_per_gas: U256::from(*max_fee),
                    to: to.unwrap_or(Address::ZERO),
                    value: U256::from(*value),
                    data: Bytes::from(data.0.clone()),
                    l1_block_number: 1,
                    timestamp: 1_700_000_001,
                    request_seq: unique_seq,
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
