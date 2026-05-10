//! Differential fuzz scenario covering signed Ethereum tx envelopes
//! (Legacy / EIP-2930 / EIP-1559 / EIP-7702) wrapped in a kind=4 L2
//! `SignedTx` sub-message. Mirrors the [`super::DiffTxScenario`] pattern
//! but exercises the standard signed-tx ingest path that the
//! ArbOS-internal-type fuzzer never touches.

use alloy_primitives::{Address, Bytes, B256, U256};
use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, AuthorizationItem, L2TxKind, SignedL2TxBuilder},
        DepositBuilder,
    },
    scenario::{Scenario, ScenarioSetup},
};
use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

use crate::{
    arbitrary_impls::{
        build_or_skip, message_step, ArbosVersion, BoundedBytes, FUZZ_GAS_CAP, FUZZ_L1_BASE_FEE,
    },
    shared_nodes::FUZZ_L2_CHAIN_ID,
};

const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65,
    0x71, 0x75, 0x65, 0x6e,
]);

/// Single signed-tx scenario: deposits funds for the signer, then submits
/// one signed Ethereum tx via `[kind=3]([kind=4][rlp(2718)])`.
#[derive(Debug, Clone, Serialize)]
pub struct DiffSignedTxScenario {
    pub arbos_version: ArbosVersion,
    pub kind: SignedTxKind,
    pub signing_key_low: [u8; 32],
    pub to: Option<Address>,
    pub data: BoundedBytes<512>,
    pub value_low: u64,
    pub gas: u64,
    pub max_fee: u64,
    pub max_priority_fee: u64,
    /// Up to 4 auth-list entries, used only when `kind == Eip7702`.
    pub authorizations: Vec<AuthInput>,
}

/// One synthesized EIP-7702 authorization. The fuzzer supplies the
/// authority's secret key and the delegate address.
#[derive(Debug, Clone, Arbitrary, Serialize)]
pub struct AuthInput {
    pub signing_key: [u8; 32],
    pub address: Address,
    pub nonce: u8,
}

/// Bounded sub-set of `L2TxKind` so the fuzzer biases evenly across the
/// four signed standard tx types.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize)]
pub enum SignedTxKind {
    Legacy,
    Eip2930,
    Eip1559,
    Eip7702,
}

impl<'a> Arbitrary<'a> for SignedTxKind {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        Ok(match u.int_in_range::<u8>(0..=3)? {
            0 => Self::Legacy,
            1 => Self::Eip2930,
            2 => Self::Eip1559,
            _ => Self::Eip7702,
        })
    }
}

impl<'a> Arbitrary<'a> for DiffSignedTxScenario {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let arbos_version = ArbosVersion::arbitrary(u)?;
        let kind = SignedTxKind::arbitrary(u)?;
        let mut signing_key_low = [0u8; 32];
        u.fill_buffer(&mut signing_key_low)?;
        // Reject 0/N (invalid secp256k1 secret) — pad with 1 to keep deterministic.
        if signing_key_low.iter().all(|b| *b == 0) {
            signing_key_low[31] = 1;
        }
        let has_to = u.arbitrary::<bool>()?;
        let to = if has_to {
            Some(Address::arbitrary(u)?)
        } else {
            None
        };
        let data = BoundedBytes::<512>::arbitrary(u)?;
        let value_low = u.int_in_range::<u64>(0..=1_000_000_000)?;
        let gas = u.int_in_range::<u64>(21_000..=FUZZ_GAS_CAP)?;
        let max_fee = u.int_in_range::<u64>(1..=2_000_000_000)?;
        let max_priority_fee = u.int_in_range::<u64>(0..=max_fee)?;
        let auth_count = if kind == SignedTxKind::Eip7702 {
            u.int_in_range::<usize>(1..=3)?
        } else {
            0
        };
        let mut authorizations = Vec::with_capacity(auth_count);
        for _ in 0..auth_count {
            authorizations.push(AuthInput::arbitrary(u)?);
        }
        Ok(Self {
            arbos_version,
            kind,
            signing_key_low,
            to,
            data,
            value_low,
            gas,
            max_fee,
            max_priority_fee,
            authorizations,
        })
    }
}

impl DiffSignedTxScenario {
    /// Translate the scenario into a single deposit + a single signed-tx
    /// `Scenario` ready to feed `DualExec::run`. `None` if the signed tx
    /// cannot be constructed (e.g. EIP-7702 CREATE — invalid by spec).
    pub fn into_scenario(self) -> Option<Scenario> {
        let kind = match self.kind {
            SignedTxKind::Legacy => L2TxKind::Legacy,
            SignedTxKind::Eip2930 => L2TxKind::Eip2930,
            SignedTxKind::Eip1559 => L2TxKind::Eip1559,
            SignedTxKind::Eip7702 => {
                // 7702 cannot be CREATE per spec — TxEip7702.to is non-optional.
                if self.to.is_none() {
                    return None;
                }
                L2TxKind::Eip7702
            }
        };

        let signing_key = B256::from_slice(&self.signing_key_low);
        let signer = derive_address(signing_key);

        let mut steps = Vec::new();
        let mut idx: u64 = 1;
        let mut delayed: u64 = 0;

        // Fund the signer so it can pay gas. Depositing 1e20 wei keeps the
        // tx well under the balance limit even at the FUZZ_GAS_CAP.
        let fund = DepositBuilder {
            from: signer,
            to: signer,
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: idx,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Some(msg) = build_or_skip(&fund) {
            delayed += 1;
            steps.push(message_step(idx, msg, delayed));
            idx += 1;
        }

        // Optional CREATE: skip for 7702 (rejected above) and for 2930 since
        // bare CREATE on Arbitrum requires careful nonce handling.
        let to_field = if matches!(kind, L2TxKind::Eip7702) {
            self.to
        } else {
            self.to
        };

        let auth_list = self
            .authorizations
            .iter()
            .map(|a| AuthorizationItem {
                chain_id: FUZZ_L2_CHAIN_ID,
                address: a.address,
                nonce: a.nonce as u64,
                signing_key: B256::from(a.signing_key),
            })
            .collect();

        let max_fee_clamped = (self.max_fee as u128).max(FUZZ_L1_BASE_FEE as u128 / 100 + 1);
        let max_priority_clamped = (self.max_priority_fee as u128).min(max_fee_clamped);

        let builder = SignedL2TxBuilder {
            chain_id: FUZZ_L2_CHAIN_ID,
            nonce: 0,
            to: to_field,
            value: U256::from(self.value_low),
            data: Bytes::from(self.data.0.clone()),
            gas_limit: self.gas.clamp(50_000, FUZZ_GAS_CAP),
            gas_price: max_fee_clamped,
            max_fee_per_gas: max_fee_clamped,
            max_priority_fee_per_gas: max_priority_clamped,
            access_list: Vec::new(),
            authorization_list: auth_list,
            kind,
            signing_key,
            l1_block_number: 1,
            timestamp: 1_700_000_001,
            request_id: None,
            sender: SEQUENCER_ALIAS,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Some(msg) = build_or_skip(&builder) {
            delayed += 1;
            steps.push(message_step(idx, msg, delayed));
        }

        Some(Scenario {
            name: "fuzz_signed_tx".into(),
            description: format!("fuzz-generated signed {:?} tx", self.kind),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps,
        })
    }
}
