use std::{cell::RefCell, path::Path};

use alloy_primitives::{Address, B256, U256};
use arb_test_utils::ArbosHarness;
use arbos::{
    address_set::open_address_set, address_table::open_address_table, blockhash::open_blockhashes,
    merkle_accumulator::open_merkle_accumulator,
};

use crate::fixture::{Action, Assertions, Fixture, Setup, TransferEntry};

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

const CHAIN_OWNER_SUBSPACE: u8 = 4;
const BLOCKHASH_SUBSPACE: u8 = 6;
const ADDRESS_TABLE_SUBSPACE: u8 = 3;
const SEND_MERKLE_SUBSPACE: u8 = 5;

type TransferLog = RefCell<Vec<TransferEntry>>;

impl SpecCase {
    pub fn load(path: &Path) -> Result<Self, SpecError> {
        let bytes = std::fs::read(path)?;
        let fixture: Fixture = serde_json::from_slice(&bytes)?;
        Ok(Self { fixture })
    }

    pub fn run(&self) -> Result<(), SpecError> {
        let mut harness = build_harness(&self.fixture.setup);
        let transfers: TransferLog = Default::default();
        for action in &self.fixture.actions {
            apply_action(&mut harness, action, &transfers)?;
        }
        check_assertions(&mut harness, &self.fixture.assertions, &transfers)
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

fn apply_action(
    harness: &mut ArbosHarness,
    action: &Action,
    transfers: &TransferLog,
) -> Result<(), SpecError> {
    match action {
        Action::L1PricingSetPricePerUnit { value } => harness
            .l1_pricing_state()
            .set_price_per_unit(*value)
            .map_err(|_| SpecError::Action("set_price_per_unit".into()))?,
        Action::L1PricingSetUnitsSinceUpdate { value } => harness
            .l1_pricing_state()
            .set_units_since_update(*value)
            .map_err(|_| SpecError::Action("set_units_since_update".into()))?,
        Action::L1PricingSetInertia { value } => harness
            .l1_pricing_state()
            .set_inertia(*value)
            .map_err(|_| SpecError::Action("set_inertia".into()))?,
        Action::L1PricingAddToFeesAvailable { amount } => harness
            .l1_pricing_state()
            .add_to_l1_fees_available(*amount)
            .map_err(|_| SpecError::Action("add_to_l1_fees_available".into()))?,
        Action::L1PricingAddPoster { poster, pay_to } => {
            let l1 = harness.l1_pricing_state();
            l1.batch_poster_table()
                .add_poster(*poster, *pay_to)
                .map_err(|_| SpecError::Action("add_poster".into()))?;
        }
        Action::L1PricingSetPosterFundsDue { poster, amount } => {
            let l1 = harness.l1_pricing_state();
            let bpt = l1.batch_poster_table();
            let bp = bpt
                .open_poster(*poster, true)
                .map_err(|_| SpecError::Action("open_poster".into()))?;
            bp.set_funds_due(*amount, &bpt.total_funds_due)
                .map_err(|_| SpecError::Action("set_funds_due".into()))?;
        }
        Action::L2PricingSetGasBacklog { value } => harness
            .l2_pricing_state()
            .set_gas_backlog(*value)
            .map_err(|_| SpecError::Action("set_gas_backlog".into()))?,
        Action::L2PricingSetMinBaseFee { value } => harness
            .l2_pricing_state()
            .set_min_base_fee_wei(*value)
            .map_err(|_| SpecError::Action("set_min_base_fee".into()))?,
        Action::L2PricingUpdateModel { time_passed } => {
            let v = harness.arbos_version();
            harness
                .l2_pricing_state()
                .update_pricing_model(*time_passed, v)
                .map_err(|_| SpecError::Action("update_pricing_model".into()))?;
        }
        Action::L2PricingAddGasConstraint {
            target,
            adjustment_window,
            backlog,
        } => harness
            .l2_pricing_state()
            .add_gas_constraint(*target, *adjustment_window, *backlog)
            .map_err(|_| SpecError::Action("add_gas_constraint".into()))?,
        Action::L2PricingClearGasConstraints => harness
            .l2_pricing_state()
            .clear_gas_constraints()
            .map_err(|_| SpecError::Action("clear_gas_constraints".into()))?,
        Action::BlockhashRecord { number, hash } => {
            let v = harness.arbos_version();
            let root = harness.root_storage();
            let bh = open_blockhashes(root.open_sub_storage(&[BLOCKHASH_SUBSPACE]));
            bh.record_new_l1_block(*number, *hash, v)
                .map_err(|_| SpecError::Action("record_new_l1_block".into()))?;
        }
        Action::AddressTableRegister { address } => {
            let root = harness.root_storage();
            let t = open_address_table(root.open_sub_storage(&[ADDRESS_TABLE_SUBSPACE]));
            t.register(*address)
                .map_err(|_| SpecError::Action("address_table register".into()))?;
        }
        Action::MerkleAppend { item } => {
            let root = harness.root_storage();
            let m = open_merkle_accumulator(root.open_sub_storage(&[SEND_MERKLE_SUBSPACE]));
            m.append(*item)
                .map_err(|_| SpecError::Action("merkle append".into()))?;
        }
        Action::ChainOwnerAdd { owner } => {
            let root = harness.root_storage();
            let s = open_address_set(root.open_sub_storage(&[CHAIN_OWNER_SUBSPACE]));
            s.add(*owner)
                .map_err(|_| SpecError::Action("chain_owners.add".into()))?;
        }
        Action::ChainOwnerRemove { owner } => {
            let v = harness.arbos_version();
            let root = harness.root_storage();
            let s = open_address_set(root.open_sub_storage(&[CHAIN_OWNER_SUBSPACE]));
            s.remove(*owner, v)
                .map_err(|_| SpecError::Action("chain_owners.remove".into()))?;
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
                .create_retryable(
                    *id,
                    *timeout,
                    *from,
                    *to,
                    *callvalue,
                    *beneficiary,
                    &calldata,
                )
                .map_err(|_| SpecError::Action("create_retryable".into()))?;
        }
        Action::RetryableIncrementNumTries { id, at_time } => {
            let rs = harness.retryable_state();
            let r = rs
                .open_retryable(*id, *at_time)
                .map_err(|_| SpecError::Action("open_retryable".into()))?
                .ok_or_else(|| SpecError::Action("retryable not found".into()))?;
            r.increment_num_tries()
                .map_err(|_| SpecError::Action("increment_num_tries".into()))?;
        }
        Action::RetryableSetTimeout {
            id,
            at_time,
            new_timeout,
        } => {
            let rs = harness.retryable_state();
            let r = rs
                .open_retryable(*id, *at_time)
                .map_err(|_| SpecError::Action("open_retryable".into()))?
                .ok_or_else(|| SpecError::Action("retryable not found".into()))?;
            r.set_timeout(*new_timeout)
                .map_err(|_| SpecError::Action("set_timeout".into()))?;
        }
        Action::RetryableDelete { id, escrow_balance } => {
            let escrow = arbos::retryables::retryable_escrow_address(*id);
            let bal = *escrow_balance;
            let rs = harness.retryable_state();
            rs.delete_retryable(
                *id,
                |from, to, amount| {
                    transfers
                        .borrow_mut()
                        .push(TransferEntry { from, to, amount });
                    Ok(())
                },
                |addr| if addr == escrow { bal } else { U256::ZERO },
            )
            .map_err(|_| SpecError::Action("delete_retryable".into()))?;
        }
    }
    Ok(())
}

fn check_assertions(
    harness: &mut ArbosHarness,
    a: &Assertions,
    transfers: &TransferLog,
) -> Result<(), SpecError> {
    if let Some(s) = &a.arbos_state {
        let st = harness.arbos_state();
        if let Some(v) = s.arbos_version {
            ensure_eq("arbos_state.arbos_version", st.arbos_version(), v)?;
        }
        if let Some(v) = s.chain_id {
            ensure_eq("arbos_state.chain_id", st.chain_id().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.brotli_compression_level {
            ensure_eq(
                "arbos_state.brotli_compression_level",
                st.brotli_compression_level().map_err(map_err)?,
                v,
            )?;
        }
    }
    if let Some(s) = &a.l1_pricing {
        let l1 = harness.l1_pricing_state();
        if let Some(v) = s.last_update_time {
            ensure_eq(
                "l1.last_update_time",
                l1.last_update_time().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.price_per_unit {
            ensure_eq(
                "l1.price_per_unit",
                l1.price_per_unit().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.units_since_update {
            ensure_eq(
                "l1.units_since_update",
                l1.units_since_update().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.l1_fees_available {
            ensure_eq(
                "l1.l1_fees_available",
                l1.l1_fees_available().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.inertia {
            ensure_eq("l1.inertia", l1.inertia().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.per_unit_reward {
            ensure_eq(
                "l1.per_unit_reward",
                l1.per_unit_reward().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.per_batch_gas_cost {
            ensure_eq(
                "l1.per_batch_gas_cost",
                l1.per_batch_gas_cost().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.equilibration_units {
            ensure_eq(
                "l1.equilibration_units",
                l1.equilibration_units().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(true) = s.surplus_is_zero {
            let (mag, _) = l1.get_l1_pricing_surplus().map_err(map_err)?;
            ensure_eq("l1.surplus_is_zero", mag, U256::ZERO)?;
        }
        if let Some(min) = s.surplus_at_least {
            let (mag, neg) = l1.get_l1_pricing_surplus().map_err(map_err)?;
            if neg || mag < min {
                return Err(SpecError::Assertion(format!(
                    "l1.surplus_at_least: expected positive >= {min}, got {mag} (neg={neg})"
                )));
            }
        }
        if let Some(v) = s.total_funds_due {
            let bpt = l1.batch_poster_table();
            ensure_eq(
                "l1.total_funds_due",
                bpt.total_funds_due().map_err(map_err)?,
                v,
            )?;
        }
    }
    if let Some(s) = &a.l2_pricing {
        let l2 = harness.l2_pricing_state();
        if let Some(v) = s.base_fee_wei {
            ensure_eq("l2.base_fee_wei", l2.base_fee_wei().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.min_base_fee_wei {
            ensure_eq(
                "l2.min_base_fee_wei",
                l2.min_base_fee_wei().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.speed_limit_per_second {
            ensure_eq(
                "l2.speed_limit_per_second",
                l2.speed_limit_per_second().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.gas_backlog {
            ensure_eq("l2.gas_backlog", l2.gas_backlog().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.pricing_inertia {
            ensure_eq(
                "l2.pricing_inertia",
                l2.pricing_inertia().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.backlog_tolerance {
            ensure_eq(
                "l2.backlog_tolerance",
                l2.backlog_tolerance().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.per_block_gas_limit {
            ensure_eq(
                "l2.per_block_gas_limit",
                l2.per_block_gas_limit().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.per_tx_gas_limit {
            ensure_eq(
                "l2.per_tx_gas_limit",
                l2.per_tx_gas_limit().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(v) = s.gas_constraints_length {
            ensure_eq(
                "l2.gas_constraints_length",
                l2.gas_constraints_length().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(threshold) = s.base_fee_at_least {
            let actual = l2.base_fee_wei().map_err(map_err)?;
            if actual < threshold {
                return Err(SpecError::Assertion(format!(
                    "l2.base_fee_at_least: expected >= {threshold}, got {actual}"
                )));
            }
        }
        if let Some(threshold) = s.base_fee_at_most {
            let actual = l2.base_fee_wei().map_err(map_err)?;
            if actual > threshold {
                return Err(SpecError::Assertion(format!(
                    "l2.base_fee_at_most: expected <= {threshold}, got {actual}"
                )));
            }
        }
    }
    if let Some(s) = &a.blockhash {
        let root = harness.root_storage();
        let bh = open_blockhashes(root.open_sub_storage(&[BLOCKHASH_SUBSPACE]));
        if let Some(v) = s.l1_block_number {
            ensure_eq(
                "blockhash.l1_block_number",
                bh.l1_block_number().map_err(map_err)?,
                v,
            )?;
        }
        if let Some(num) = s.has_hash_for {
            if bh.block_hash(num).map_err(map_err)?.is_none() {
                return Err(SpecError::Assertion(format!(
                    "blockhash.has_hash_for {num}: missing"
                )));
            }
        }
        if let Some(num) = s.no_hash_for {
            if bh.block_hash(num).map_err(map_err)?.is_some() {
                return Err(SpecError::Assertion(format!(
                    "blockhash.no_hash_for {num}: present"
                )));
            }
        }
        if let Some(check) = &s.hash_for_block_equals {
            let actual = bh.block_hash(check.block_number).map_err(map_err)?;
            ensure_eq(
                "blockhash.hash_for_block_equals",
                actual,
                Some(check.expected),
            )?;
        }
    }
    if let Some(s) = &a.retryable {
        if let Some(check) = &s.exists {
            let rs = harness.retryable_state();
            let opened = rs
                .open_retryable(check.id, check.at_time)
                .map_err(map_err)?;
            ensure_eq("retryable.exists", opened.is_some(), check.expected)?;
        }
        if let Some(check) = &s.num_tries {
            let rs = harness.retryable_state();
            let opened = rs
                .open_retryable(check.id, check.at_time)
                .map_err(map_err)?
                .ok_or_else(|| SpecError::Assertion("retryable.num_tries: missing".into()))?;
            ensure_eq(
                "retryable.num_tries",
                opened.num_tries().map_err(map_err)?,
                check.expected,
            )?;
        }
    }
    if let Some(s) = &a.merkle {
        let root = harness.root_storage();
        let m = open_merkle_accumulator(root.open_sub_storage(&[SEND_MERKLE_SUBSPACE]));
        if let Some(v) = s.size {
            ensure_eq("merkle.size", m.size().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.root {
            ensure_eq::<B256>("merkle.root", m.root().map_err(map_err)?, v)?;
        }
        if let Some(v) = s.root_not {
            let actual = m.root().map_err(map_err)?;
            if actual == v {
                return Err(SpecError::Assertion(format!(
                    "merkle.root_not: matched forbidden value {v:?}"
                )));
            }
        }
    }
    if let Some(s) = &a.address_table {
        let root = harness.root_storage();
        let t = open_address_table(root.open_sub_storage(&[ADDRESS_TABLE_SUBSPACE]));
        if let Some(v) = s.size {
            ensure_eq("address_table.size", t.size().map_err(map_err)?, v)?;
        }
        if let Some(c) = &s.address_at_index {
            let actual = t.lookup_index(c.index).map_err(map_err)?;
            ensure_eq("address_table.address_at_index", actual, Some(c.expected))?;
        }
        if let Some(c) = &s.index_for_address {
            let (idx, exists) = t.lookup(c.address).map_err(map_err)?;
            if !exists {
                return Err(SpecError::Assertion(format!(
                    "address_table.index_for_address: address {} not registered",
                    c.address
                )));
            }
            ensure_eq("address_table.index_for_address", idx, c.expected_index)?;
        }
        if let Some(c) = &s.contains {
            ensure_eq(
                "address_table.contains",
                t.address_exists(c.address).map_err(map_err)?,
                c.expected,
            )?;
        }
    }
    if let Some(s) = &a.chain_owners {
        let root = harness.root_storage();
        let owners = open_address_set(root.open_sub_storage(&[CHAIN_OWNER_SUBSPACE]));
        if let Some(v) = s.size {
            ensure_eq("chain_owners.size", owners.size().map_err(map_err)?, v)?;
        }
        if let Some(c) = &s.contains {
            ensure_eq(
                "chain_owners.contains",
                owners.is_member(c.address).map_err(map_err)?,
                c.expected,
            )?;
        }
    }
    if let Some(s) = &a.transfers {
        let log = transfers.borrow();
        if let Some(v) = s.log_length {
            ensure_eq("transfers.log_length", log.len(), v)?;
        }
        if let Some(entry) = &s.log_contains {
            let found = log
                .iter()
                .any(|t| t.from == entry.from && t.to == entry.to && t.amount == entry.amount);
            if !found {
                return Err(SpecError::Assertion(format!(
                    "transfers.log_contains: ({:?}, {:?}, {}) not in log",
                    entry.from, entry.to, entry.amount
                )));
            }
        }
    }
    Ok(())
}

fn ensure_eq<T: std::fmt::Debug + PartialEq>(
    field: &str,
    actual: T,
    expected: T,
) -> Result<(), SpecError> {
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

#[allow(dead_code)]
fn _unused_marker(_: Address) {}
