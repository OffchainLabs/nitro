pub mod arbos;
pub mod stylus;
pub mod tx;

pub use arbos::ArbosVersion;
pub use stylus::StylusFuzzInput;
pub use tx::{BoundedBytes, TxScenario};

use alloy_primitives::Address;
use arb_test_harness::messaging::L1Message;
use arb_test_harness::scenario::{Scenario, ScenarioSetup, ScenarioStep};
use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;

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
        Scenario {
            name: format!("fuzz_precompile_{:#04x}", self.precompile.0),
            description: "fuzz-generated precompile invocation".into(),
            setup: ScenarioSetup {
                l2_chain_id: 412_346,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps: Vec::<ScenarioStep>::new(),
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
        Scenario {
            name: "fuzz_tx".into(),
            description: "fuzz-generated single-tx scenario".into(),
            setup: ScenarioSetup {
                l2_chain_id: 412_346,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps: Vec::<ScenarioStep>::new(),
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
        Scenario {
            name: "fuzz_property".into(),
            description: "fuzz-generated mixed-tx scenario".into(),
            setup: ScenarioSetup {
                l2_chain_id: 412_346,
                arbos_version: self.arbos_version.0,
                genesis: None,
            },
            steps: Vec::<ScenarioStep>::new(),
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
