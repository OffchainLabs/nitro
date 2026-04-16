use alloy_primitives::{Address, B256, U256};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Fixture {
    pub name: String,
    #[serde(default)]
    pub description: String,
    #[serde(default)]
    pub setup: Setup,
    #[serde(default)]
    pub actions: Vec<Action>,
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

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum Action {
    L1PricingSetPricePerUnit { value: U256 },
    L1PricingSetUnitsSinceUpdate { value: u64 },
    L2PricingSetGasBacklog { value: u64 },
    L2PricingUpdateModel { time_passed: u64 },
    BlockhashRecord { number: u64, hash: B256 },
    AddressTableRegister { address: Address },
    MerkleAppend { item: B256 },
    RetryableCreate {
        id: B256,
        timeout: u64,
        from: Address,
        #[serde(default)]
        to: Option<Address>,
        callvalue: U256,
        beneficiary: Address,
        #[serde(default)]
        calldata_hex: String,
    },
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Assertions {
    #[serde(default)]
    pub arbos_state: Option<ArbosStateAssertions>,
    #[serde(default)]
    pub l1_pricing: Option<L1PricingAssertions>,
    #[serde(default)]
    pub l2_pricing: Option<L2PricingAssertions>,
    #[serde(default)]
    pub blockhash: Option<BlockhashAssertions>,
    #[serde(default)]
    pub retryable: Option<RetryableAssertions>,
    #[serde(default)]
    pub merkle: Option<MerkleAssertions>,
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
    /// `>=` comparison: actual base fee must be at least this value.
    pub base_fee_at_least: Option<U256>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct BlockhashAssertions {
    pub l1_block_number: Option<u64>,
    pub has_hash_for: Option<u64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct RetryableAssertions {
    pub exists: Option<RetryableExistsCheck>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RetryableExistsCheck {
    pub id: B256,
    pub at_time: u64,
    pub expected: bool,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct MerkleAssertions {
    pub size: Option<u64>,
    pub root: Option<B256>,
}
