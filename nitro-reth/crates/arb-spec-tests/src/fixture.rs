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
    L1PricingSetInertia { value: u64 },
    L1PricingAddToFeesAvailable { amount: U256 },
    L1PricingAddPoster { poster: Address, pay_to: Address },
    L1PricingSetPosterFundsDue { poster: Address, amount: U256 },
    L2PricingSetGasBacklog { value: u64 },
    L2PricingSetMinBaseFee { value: U256 },
    L2PricingUpdateModel { time_passed: u64 },
    L2PricingAddGasConstraint { target: u64, adjustment_window: u64, backlog: u64 },
    L2PricingClearGasConstraints,
    BlockhashRecord { number: u64, hash: B256 },
    AddressTableRegister { address: Address },
    MerkleAppend { item: B256 },
    ChainOwnerAdd { owner: Address },
    ChainOwnerRemove { owner: Address },
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
    RetryableIncrementNumTries { id: B256, at_time: u64 },
    RetryableSetTimeout { id: B256, at_time: u64, new_timeout: u64 },
    /// Delete a retryable. Escrow balance for the closure is `escrow_balance`.
    RetryableDelete { id: B256, escrow_balance: U256 },
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
    #[serde(default)]
    pub address_table: Option<AddressTableAssertions>,
    #[serde(default)]
    pub chain_owners: Option<ChainOwnersAssertions>,
    #[serde(default)]
    pub transfers: Option<TransferAssertions>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ArbosStateAssertions {
    pub arbos_version: Option<u64>,
    pub chain_id: Option<U256>,
    pub brotli_compression_level: Option<u64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct L1PricingAssertions {
    pub last_update_time: Option<u64>,
    pub price_per_unit: Option<U256>,
    pub units_since_update: Option<u64>,
    pub l1_fees_available: Option<U256>,
    pub inertia: Option<u64>,
    pub per_unit_reward: Option<u64>,
    pub per_batch_gas_cost: Option<i64>,
    pub equilibration_units: Option<U256>,
    /// (magnitude, negative) tuple for the surplus.
    pub surplus_is_zero: Option<bool>,
    pub surplus_at_least: Option<U256>,
    pub total_funds_due: Option<U256>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct L2PricingAssertions {
    pub base_fee_wei: Option<U256>,
    pub min_base_fee_wei: Option<U256>,
    pub speed_limit_per_second: Option<u64>,
    pub gas_backlog: Option<u64>,
    pub pricing_inertia: Option<u64>,
    pub backlog_tolerance: Option<u64>,
    pub per_block_gas_limit: Option<u64>,
    pub per_tx_gas_limit: Option<u64>,
    pub base_fee_at_least: Option<U256>,
    pub base_fee_at_most: Option<U256>,
    pub gas_constraints_length: Option<u64>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct BlockhashAssertions {
    pub l1_block_number: Option<u64>,
    pub has_hash_for: Option<u64>,
    pub no_hash_for: Option<u64>,
    pub hash_for_block_equals: Option<HashAtBlockCheck>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HashAtBlockCheck {
    pub block_number: u64,
    pub expected: B256,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct RetryableAssertions {
    pub exists: Option<RetryableExistsCheck>,
    pub num_tries: Option<RetryableNumTriesCheck>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RetryableExistsCheck {
    pub id: B256,
    pub at_time: u64,
    pub expected: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RetryableNumTriesCheck {
    pub id: B256,
    pub at_time: u64,
    pub expected: u64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct MerkleAssertions {
    pub size: Option<u64>,
    pub root: Option<B256>,
    pub root_not: Option<B256>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AddressTableAssertions {
    pub size: Option<u64>,
    pub address_at_index: Option<AddressAtIndexCheck>,
    pub index_for_address: Option<IndexForAddressCheck>,
    pub contains: Option<AddressContainsCheck>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddressAtIndexCheck {
    pub index: u64,
    pub expected: Address,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IndexForAddressCheck {
    pub address: Address,
    pub expected_index: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AddressContainsCheck {
    pub address: Address,
    pub expected: bool,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ChainOwnersAssertions {
    pub size: Option<u64>,
    pub contains: Option<AddressContainsCheck>,
}

/// Assertions about the side-effect log captured during retryable_delete actions.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TransferAssertions {
    pub log_length: Option<usize>,
    pub log_contains: Option<TransferEntry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransferEntry {
    pub from: Address,
    pub to: Address,
    pub amount: U256,
}
