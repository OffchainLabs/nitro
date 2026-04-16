use std::path::Path;

use alloy_primitives::B256;
use arb_test_utils::ArbosHarness;
use arbos::{address_table::open_address_table, blockhash::open_blockhashes, merkle_accumulator::open_merkle_accumulator};

use crate::fixture::{Action, Assertions, Fixture, Setup};

pub struct SpecCase {
    pub fixture: Fixture,
}

#[derive(Debug, thiserror::Error)]
pub enum SpecError {
    #[error("io: {0}")]
    Io(#[from] std::io::Error),
    #[error("json: {0}")]
    Json(#[from] serde_json::Error),
    #[error("hex: {0}")]
    Hex(#[from] hex::FromHexError),
    #[error("action failed: {0}")]
    Action(String),
    #[error("assertion failed: {0}")]
    Assertion(String),
}

impl SpecCase {
    pub fn load(path: &Path) -> Result<Self, SpecError> {
        let bytes = std::fs::read(path)?;
        let fixture: Fixture = serde_json::from_slice(&bytes)?;
        Ok(Self { fixture })
    }

    pub fn run(&self) -> Result<(), SpecError> {
        let mut harness = build_harness(&self.fixture.setup);
        for action in &self.fixture.actions {
            apply_action(&mut harness, action)?;
        }
        check_assertions(&mut harness, &self.fixture.assertions)
    }
}

fn build_harness(setup: &Setup) -> ArbosHarness {
    let mut h = ArbosHarness::new()
        .with_arbos_version(setup.arbos_version)
        .with_chain_id(setup.chain_id);
    if let Some(fee) = setup.l1_initial_base_fee {
        h = h.with_l1_initial_base_fee(fee);
    }
    h.initialize()
}

fn apply_action(harness: &mut ArbosHarness, action: &Action) -> Result<(), SpecError> {
    match action {
        Action::L1PricingSetPricePerUnit { value } => {
            harness
                .l1_pricing_state()
                .set_price_per_unit(*value)
                .map_err(|_| SpecError::Action("set_price_per_unit failed".into()))?;
        }
        Action::L1PricingSetUnitsSinceUpdate { value } => {
            harness
                .l1_pricing_state()
                .set_units_since_update(*value)
                .map_err(|_| SpecError::Action("set_units_since_update failed".into()))?;
        }
        Action::L2PricingSetGasBacklog { value } => {
            harness
                .l2_pricing_state()
                .set_gas_backlog(*value)
                .map_err(|_| SpecError::Action("set_gas_backlog failed".into()))?;
        }
        Action::L2PricingUpdateModel { time_passed } => {
            let arbos_version = harness.arbos_version();
            harness
                .l2_pricing_state()
                .update_pricing_model(*time_passed, arbos_version)
                .map_err(|_| SpecError::Action("update_pricing_model failed".into()))?;
        }
        Action::BlockhashRecord { number, hash } => {
            let arbos_version = harness.arbos_version();
            let root = harness.root_storage();
            let bh = open_blockhashes(root.open_sub_storage(&[6]));
            bh.record_new_l1_block(*number, *hash, arbos_version)
                .map_err(|_| SpecError::Action("record_new_l1_block failed".into()))?;
        }
        Action::AddressTableRegister { address } => {
            let root = harness.root_storage();
            let t = open_address_table(root.open_sub_storage(&[3]));
            t.register(*address)
                .map_err(|_| SpecError::Action("address_table register failed".into()))?;
        }
        Action::MerkleAppend { item } => {
            let root = harness.root_storage();
            let m = open_merkle_accumulator(root.open_sub_storage(&[5]));
            m.append(*item)
                .map_err(|_| SpecError::Action("merkle append failed".into()))?;
        }
        Action::RetryableCreate {
            id,
            timeout,
            from,
            to,
            callvalue,
            beneficiary,
            calldata_hex,
        } => {
            let calldata = if calldata_hex.is_empty() {
                Vec::new()
            } else {
                hex::decode(calldata_hex.trim_start_matches("0x"))?
            };
            harness
                .retryable_state()
                .create_retryable(*id, *timeout, *from, *to, *callvalue, *beneficiary, &calldata)
                .map_err(|_| SpecError::Action("create_retryable failed".into()))?;
        }
    }
    Ok(())
}

fn check_assertions(harness: &mut ArbosHarness, a: &Assertions) -> Result<(), SpecError> {
    if let Some(s) = &a.arbos_state {
        let st = harness.arbos_state();
        if let Some(v) = s.arbos_version {
            ensure_eq("arbos_state.arbos_version", st.arbos_version(), v)?;
        }
        if let Some(v) = s.chain_id {
            ensure_eq("arbos_state.chain_id", st.chain_id().map_err(map_err)?, v)?;
        }
    }
    if let Some(s) = &a.l1_pricing {
        let l1 = harness.l1_pricing_state();
        if let Some(v) = s.last_update_time {
            ensure_eq("l1.last_update_time", l1.last_update_time().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.price_per_unit {
            ensure_eq("l1.price_per_unit", l1.price_per_unit().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.units_since_update {
            ensure_eq("l1.units_since_update", l1.units_since_update().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.l1_fees_available {
            ensure_eq("l1.l1_fees_available", l1.l1_fees_available().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.inertia {
            ensure_eq("l1.inertia", l1.inertia().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.per_unit_reward {
            ensure_eq("l1.per_unit_reward", l1.per_unit_reward().map_err(map_err)?, v)?;
        }
    }
    if let Some(s) = &a.l2_pricing {
        let l2 = harness.l2_pricing_state();
        if let Some(v) = s.base_fee_wei {
            ensure_eq("l2.base_fee_wei", l2.base_fee_wei().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.min_base_fee_wei {
            ensure_eq("l2.min_base_fee_wei", l2.min_base_fee_wei().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.speed_limit_per_second {
            ensure_eq("l2.speed_limit_per_second", l2.speed_limit_per_second().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.gas_backlog {
            ensure_eq("l2.gas_backlog", l2.gas_backlog().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.pricing_inertia {
            ensure_eq("l2.pricing_inertia", l2.pricing_inertia().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.backlog_tolerance {
            ensure_eq("l2.backlog_tolerance", l2.backlog_tolerance().map_err(map_err)?, v)?;
        }
        if let Some(threshold) = s.base_fee_at_least {
            let actual = l2.base_fee_wei().map_err(map_err)?;
            if actual < threshold {
                return Err(SpecError::Assertion(format!(
                    "l2.base_fee_at_least: expected >= {threshold:?}, got {actual:?}"
                )));
            }
        }
    }
    if let Some(s) = &a.blockhash {
        let root = harness.root_storage();
        let bh = open_blockhashes(root.open_sub_storage(&[6]));
        if let Some(v) = s.l1_block_number {
            ensure_eq("blockhash.l1_block_number", bh.l1_block_number().map_err(map_err)?, v)?;
        }
        if let Some(num) = s.has_hash_for {
            let h = bh.block_hash(num).map_err(map_err)?;
            if h.is_none() {
                return Err(SpecError::Assertion(format!(
                    "blockhash.has_hash_for: expected hash at {num}, got none"
                )));
            }
        }
    }
    if let Some(s) = &a.retryable {
        if let Some(check) = &s.exists {
            let rs = harness.retryable_state();
            let opened = rs.open_retryable(check.id, check.at_time).map_err(map_err)?;
            ensure_eq("retryable.exists", opened.is_some(), check.expected)?;
        }
    }
    if let Some(s) = &a.merkle {
        let root = harness.root_storage();
        let m = open_merkle_accumulator(root.open_sub_storage(&[5]));
        if let Some(v) = s.size {
            ensure_eq("merkle.size", m.size().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.root {
            let actual = m.root().map_err(map_err)?;
            ensure_eq::<B256>("merkle.root", actual, v)?;
        }
    }
    Ok(())
}

fn ensure_eq<T: std::fmt::Debug + PartialEq>(field: &str, actual: T, expected: T) -> Result<(), SpecError> {
    if actual != expected {
        return Err(SpecError::Assertion(format!(
            "{field}: expected {expected:?}, got {actual:?}"
        )));
    }
    Ok(())
}

fn map_err(_: ()) -> SpecError {
    SpecError::Assertion("storage read failed".into())
}
