use std::path::Path;

use arb_test_utils::ArbosHarness;

use crate::fixture::{Assertions, Fixture, Setup};

pub struct SpecCase {
    pub fixture: Fixture,
}

#[derive(Debug, thiserror::Error)]
pub enum SpecError {
    #[error("io: {0}")]
    Io(#[from] std::io::Error),
    #[error("json: {0}")]
    Json(#[from] serde_json::Error),
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
        let harness = build_harness(&self.fixture.setup);
        check_assertions(harness, &self.fixture.assertions)
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

fn check_assertions(mut harness: ArbosHarness, a: &Assertions) -> Result<(), SpecError> {
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
