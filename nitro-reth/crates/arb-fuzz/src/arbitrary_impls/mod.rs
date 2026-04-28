pub mod arbos;
pub mod stylus;
pub mod tx;

pub use arbos::ArbosVersion;
pub use stylus::StylusFuzzInput;
pub use tx::{BoundedBytes, TxScenario};

use alloy_primitives::{Address, Bytes, U256};
use arb_test_harness::{
    messaging::{
        ContractTxBuilder, DepositBuilder, L1Message, MessageBuilder, UnsignedUserTxBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};
use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

use crate::shared_nodes::FUZZ_L2_CHAIN_ID;

/// Default precompile-call gas ceiling, capped to keep fuzz iterations cheap.
const FUZZ_GAS_CAP: u64 = 4_000_000;
/// Synthetic L1 base fee used when building messages.
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
/// Max-fee-per-gas for unsigned user txs; well above the network minimum.
const FUZZ_MAX_FEE: u128 = 2_000_000_000;

#[derive(Debug, Clone, Arbitrary, Serialize)]
pub struct PrecompileScenario {
    pub arbos_version: ArbosVersion,
    pub precompile: PrecompileAddr,
    pub calldata: BoundedBytes<2048>,
    pub gas_limit: u64,
    pub pre_state: SmallPreState,
    pub caller: Address,
}

impl PrecompileScenario {
    pub fn into_scenario(self) -> Scenario {
        let mut steps = Vec::new();
        let mut idx: u64 = 1;
        let mut delayed: u64 = 0;

        // Optional ETH deposit warm-up so the caller can pay fees.
        for (addr, amount) in &self.pre_state.balances {
            if *amount == 0 {
                continue;
            }
            let dep = DepositBuilder {
                from: *addr,
                to: *addr,
                amount: U256::from(*amount),
                l1_block_number: 1,
                timestamp: 1_700_000_000,
                request_seq: idx,
                base_fee_l1: FUZZ_L1_BASE_FEE,
            };
            if let Some(msg) = build_or_skip(&dep) {
                delayed += 1;
                steps.push(message_step(idx, msg, delayed));
                idx += 1;
            }
        }

        // Always fund the caller so the unsigned tx can pay gas + value=0.
        let fund = DepositBuilder {
            from: self.caller,
            to: self.caller,
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

        // The actual precompile invocation: unsigned user tx so we don't have
        // to invent a valid ECDSA signature in the fuzz body.
        let to = precompile_address(self.precompile.0);
        let gas_limit = self.gas_limit.clamp(100_000, FUZZ_GAS_CAP);
        let call = UnsignedUserTxBuilder {
            from: self.caller,
            gas_limit,
            max_fee_per_gas: U256::from(FUZZ_MAX_FEE),
            nonce: 0,
            to,
            value: U256::ZERO,
            data: Bytes::from(self.calldata.0.clone()),
            l1_block_number: 1,
            timestamp: 1_700_000_001,
            request_seq: idx,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Some(msg) = build_or_skip(&call) {
            delayed += 1;
            steps.push(message_step(idx, msg, delayed));
        }

        Scenario {
            name: format!("fuzz_precompile_{:#04x}", self.precompile.0),
            description: "fuzz-generated precompile invocation".into(),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps,
        }
    }
}

#[derive(Debug, Clone, Copy, Serialize)]
pub struct PrecompileAddr(pub u8);

impl<'a> Arbitrary<'a> for PrecompileAddr {
    fn arbitrary(u: &mut Unstructured<'a>) -> arbitrary::Result<Self> {
        let raw: u8 = u.int_in_range(0x64..=0x74)?;
        Ok(Self(raw))
    }
}

#[derive(Debug, Clone, Default, Arbitrary, Serialize)]
pub struct SmallPreState {
    pub balances: Vec<(Address, u128)>,
    pub contract: Option<(Address, BoundedBytes<512>)>,
}

#[derive(Debug, Clone, Arbitrary, Serialize)]
pub struct DiffTxScenario {
    pub arbos_version: ArbosVersion,
    pub tx: TxScenario,
    pub pre_state: SmallPreState,
}

impl DiffTxScenario {
    pub fn into_scenario(self) -> Scenario {
        let mut steps = Vec::new();
        let mut idx: u64 = 1;
        let mut delayed: u64 = 0;

        // Pre-state balances first so any address gets seed funds.
        for (addr, amount) in self.pre_state.balances.iter().take(4) {
            if *amount == 0 {
                continue;
            }
            let dep = DepositBuilder {
                from: *addr,
                to: *addr,
                amount: U256::from(*amount),
                l1_block_number: 1,
                timestamp: 1_700_000_000,
                request_seq: idx,
                base_fee_l1: FUZZ_L1_BASE_FEE,
            };
            if let Some(msg) = build_or_skip(&dep) {
                delayed += 1;
                steps.push(message_step(idx, msg, delayed));
                idx += 1;
            }
        }

        let funding_amount = U256::from(10u128).pow(U256::from(20u64));
        let fund = DepositBuilder {
            from: self.tx.from,
            to: self.tx.from,
            amount: funding_amount,
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

        let gas_limit = self.tx.gas.clamp(100_000, FUZZ_GAS_CAP);
        let max_fee = self.tx.max_fee.min(FUZZ_MAX_FEE);
        let data = Bytes::from(self.tx.data.0.clone());

        let msg_opt = match self.tx.to {
            Some(to) => {
                let builder = UnsignedUserTxBuilder {
                    from: self.tx.from,
                    gas_limit,
                    max_fee_per_gas: U256::from(max_fee),
                    nonce: 0,
                    to,
                    value: self.tx.value,
                    data,
                    l1_block_number: 1,
                    timestamp: 1_700_000_001,
                    request_seq: idx,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                build_or_skip(&builder)
            }
            None => {
                let builder = ContractTxBuilder {
                    from: self.tx.from,
                    gas_limit,
                    max_fee_per_gas: U256::from(max_fee),
                    to: Address::ZERO,
                    value: self.tx.value,
                    data,
                    l1_block_number: 1,
                    timestamp: 1_700_000_001,
                    request_seq: idx,
                    base_fee_l1: FUZZ_L1_BASE_FEE,
                };
                build_or_skip(&builder)
            }
        };
        if let Some(msg) = msg_opt {
            delayed += 1;
            steps.push(message_step(idx, msg, delayed));
        }

        Scenario {
            name: "fuzz_tx".into(),
            description: "fuzz-generated single-tx scenario".into(),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps,
        }
    }
}

#[derive(Debug, Clone, Arbitrary, Serialize)]
pub struct ScenarioMix {
    pub arbos_version: ArbosVersion,
    pub txs: Vec<TxScenario>,
}

impl ScenarioMix {
    pub fn into_scenario(self) -> Scenario {
        let mut steps = Vec::new();
        let mut idx: u64 = 1;
        let mut delayed: u64 = 0;

        // Cap the number of txs so a single fuzz iteration stays cheap.
        const MAX_TXS: usize = 8;
        let mut funded: std::collections::BTreeSet<Address> = std::collections::BTreeSet::new();

        for tx in self.txs.iter().take(MAX_TXS) {
            if funded.insert(tx.from) {
                let fund = DepositBuilder {
                    from: tx.from,
                    to: tx.from,
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
            }

            let gas_limit = tx.gas.clamp(100_000, FUZZ_GAS_CAP);
            let max_fee = tx.max_fee.min(FUZZ_MAX_FEE);
            let data = Bytes::from(tx.data.0.clone());
            let msg_opt = match tx.to {
                Some(to) => {
                    let b = UnsignedUserTxBuilder {
                        from: tx.from,
                        gas_limit,
                        max_fee_per_gas: U256::from(max_fee),
                        nonce: 0,
                        to,
                        value: tx.value,
                        data,
                        l1_block_number: 1,
                        timestamp: 1_700_000_001,
                        request_seq: idx,
                        base_fee_l1: FUZZ_L1_BASE_FEE,
                    };
                    build_or_skip(&b)
                }
                None => {
                    let b = ContractTxBuilder {
                        from: tx.from,
                        gas_limit,
                        max_fee_per_gas: U256::from(max_fee),
                        to: Address::ZERO,
                        value: tx.value,
                        data,
                        l1_block_number: 1,
                        timestamp: 1_700_000_001,
                        request_seq: idx,
                        base_fee_l1: FUZZ_L1_BASE_FEE,
                    };
                    build_or_skip(&b)
                }
            };
            if let Some(msg) = msg_opt {
                delayed += 1;
                steps.push(message_step(idx, msg, delayed));
                idx += 1;
            }
        }

        Scenario {
            name: "fuzz_property".into(),
            description: "fuzz-generated mixed-tx scenario".into(),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps,
        }
    }

    pub fn total_eth_before(&self) -> u128 {
        0
    }

    pub fn total_eth_after_arbreth(&self) -> u128 {
        0
    }

    pub fn burned_to_zero_arbreth(&self) -> u128 {
        0
    }
}

#[doc(hidden)]
pub fn message_step(idx: u64, message: L1Message, delayed_messages_read: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message,
        delayed_messages_read,
    }
}

fn build_or_skip<B: MessageBuilder>(b: &B) -> Option<L1Message> {
    b.build().ok()
}

fn precompile_address(byte: u8) -> Address {
    let mut bytes = [0u8; 20];
    bytes[19] = byte;
    Address::new(bytes)
}
