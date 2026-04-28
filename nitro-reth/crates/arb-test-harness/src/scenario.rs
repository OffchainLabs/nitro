use serde::{Deserialize, Serialize};

use crate::messaging::L1Message;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Scenario {
    pub name: String,
    #[serde(default)]
    pub description: String,
    pub setup: ScenarioSetup,
    pub steps: Vec<ScenarioStep>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ScenarioSetup {
    pub l2_chain_id: u64,
    pub arbos_version: u64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub genesis: Option<serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ScenarioStep {
    Message {
        idx: u64,
        message: L1Message,
        delayed_messages_read: u64,
    },
    AdvanceTime {
        seconds: u64,
    },
    AdvanceL1Block {
        blocks: u64,
    },
}
