use alloy_primitives::U256;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Fixture {
    pub name: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub setup: Setup,
    #[serde(default)]
    pub assertions: Assertions,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Setup {
    #[serde(default = "default_arbos_version")]
    pub arbos_version: u64,
    #[serde(default = "default_chain_id")]
    pub chain_id: u64,
    #[serde(default)]
    pub l1_initial_base_fee: Option<U256>,
}

fn default_arbos_version() -> u64 {
    30
}

fn default_chain_id() -> u64 {
    412346
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Assertions {
    #[serde(default)]
    pub arbos_state: Option<ArbosStateAssertions>,
    #[serde(default)]
    pub l1_pricing: Option<L1PricingAssertions>,
    #[serde(default)]
    pub l2_pricing: Option<L2PricingAssertions>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ArbosStateAssertions {
    pub arbos_version: Option<u64>,
    pub chain_id: Option<U256>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct L1PricingAssertions {
    pub last_update_time: Option<u64>,
    pub price_per_unit: Option<U256>,
    pub units_since_update: Option<u64>,
    pub l1_fees_available: Option<U256>,
    pub inertia: Option<u64>,
    pub per_unit_reward: Option<u64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct L2PricingAssertions {
    pub base_fee_wei: Option<U256>,
    pub min_base_fee_wei: Option<U256>,
    pub speed_limit_per_second: Option<u64>,
    pub gas_backlog: Option<u64>,
    pub pricing_inertia: Option<u64>,
    pub backlog_tolerance: Option<u64>,
}
