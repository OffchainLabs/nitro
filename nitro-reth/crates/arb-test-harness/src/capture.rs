//! Capture state from a [`crate::node::ExecutionNode`] into the
//! `ExecutionFixture` JSON shape consumed by `arb-spec-tests`.

use crate::{error::HarnessError, node::ExecutionNode, scenario::Scenario, Result};

pub struct CapturedScenario {
    pub scenario: Scenario,
    pub expected_json: serde_json::Value,
}

/// Run `scenario` against `node`, collecting per-block + per-tx +
/// per-storage state into a JSON object that matches
/// `ExecutionExpectations`.
pub fn capture_from_node(_node: &mut dyn ExecutionNode, _scenario: &Scenario) -> Result<CapturedScenario> {
    Err(HarnessError::NotImplemented {
        what: "capture_from_node (Stage 2 / Agent A)",
    })
}
