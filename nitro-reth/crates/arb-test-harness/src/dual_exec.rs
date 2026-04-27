//! Run a [`Scenario`] against one or two [`ExecutionNode`] backends and
//! produce a structural [`DiffReport`].

use alloy_primitives::{Address, B256, U256};
use serde::{Deserialize, Serialize};

use crate::{
    node::{Block, ExecutionNode, MultiGasDims, TxReceipt},
    scenario::Scenario,
    Result,
};

/// Pair of nodes. By convention the LEFT node is the "truth" (Nitro)
/// and the RIGHT node is the "subject" (arbreth). Spec tests in
/// `Verify` mode only run against RIGHT.
pub struct DualExec<L: ExecutionNode, R: ExecutionNode> {
    pub left: L,
    pub right: R,
}

impl<L: ExecutionNode, R: ExecutionNode> DualExec<L, R> {
    pub fn new(left: L, right: R) -> Self {
        Self { left, right }
    }

    /// Execute the scenario on both nodes, collecting per-block,
    /// per-tx, per-storage, per-log diffs. Implementation deferred to
    /// Agent A — this skeleton just defines the shape.
    pub fn run(&mut self, _scenario: &Scenario) -> Result<DiffReport> {
        Err(crate::HarnessError::NotImplemented {
            what: "DualExec::run (Stage 2 / Agent A)",
        })
    }
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct DiffReport {
    pub block_diffs: Vec<BlockDiff>,
    pub tx_diffs: Vec<TxDiff>,
    pub state_diffs: Vec<StateDiff>,
    pub log_diffs: Vec<LogDiff>,
}

impl DiffReport {
    pub fn is_clean(&self) -> bool {
        self.block_diffs.is_empty()
            && self.tx_diffs.is_empty()
            && self.state_diffs.is_empty()
            && self.log_diffs.is_empty()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlockDiff {
    pub number: u64,
    pub field: String,
    pub left: serde_json::Value,
    pub right: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TxDiff {
    pub tx_hash: B256,
    pub field: String,
    pub left: serde_json::Value,
    pub right: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateDiff {
    pub address: Address,
    pub at_block: u64,
    pub field: StateField,
    pub left: serde_json::Value,
    pub right: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum StateField {
    Balance,
    Nonce,
    Code,
    Storage(B256),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogDiff {
    pub block_number: u64,
    pub log_index: u64,
    pub field: String,
    pub left: serde_json::Value,
    pub right: serde_json::Value,
}

/// Convenience ctor used in tests / fuzz when only a single side is
/// available.
pub fn check_block_field<T: PartialEq + serde::Serialize>(
    number: u64,
    field: &str,
    left: &T,
    right: &T,
) -> Option<BlockDiff> {
    if left == right {
        None
    } else {
        Some(BlockDiff {
            number,
            field: field.to_string(),
            left: serde_json::to_value(left).unwrap_or(serde_json::Value::Null),
            right: serde_json::to_value(right).unwrap_or(serde_json::Value::Null),
        })
    }
}

/// Convenience ctor for tx-level diffs.
pub fn check_tx_field<T: PartialEq + serde::Serialize>(
    tx_hash: B256,
    field: &str,
    left: &T,
    right: &T,
) -> Option<TxDiff> {
    if left == right {
        None
    } else {
        Some(TxDiff {
            tx_hash,
            field: field.to_string(),
            left: serde_json::to_value(left).unwrap_or(serde_json::Value::Null),
            right: serde_json::to_value(right).unwrap_or(serde_json::Value::Null),
        })
    }
}

// Suppress unused-warning placeholders until impls land.
#[doc(hidden)]
pub fn _placeholder_uses(
    _b: Block,
    _r: TxReceipt,
    _m: MultiGasDims,
    _u: U256,
) {
}
