use alloy_primitives::{Address, B256};
use serde::{Deserialize, Serialize};

use crate::{
    node::{BlockId, EvmLog, ExecutionNode, TxReceipt},
    scenario::{Scenario, ScenarioStep},
    Result,
};

pub struct DualExec<L: ExecutionNode, R: ExecutionNode> {
    pub left: L,
    pub right: R,
}

impl<L: ExecutionNode, R: ExecutionNode> DualExec<L, R> {
    pub fn new(left: L, right: R) -> Self {
        Self { left, right }
    }

    pub fn run(&mut self, scenario: &Scenario) -> Result<DiffReport> {
        let mut report = DiffReport::default();

        for step in &scenario.steps {
            match step {
                ScenarioStep::Message {
                    idx,
                    message,
                    delayed_messages_read,
                } => {
                    self.left.submit_message(*idx, message, *delayed_messages_read)?;
                    self.right.submit_message(*idx, message, *delayed_messages_read)?;
                }
                ScenarioStep::AdvanceTime { .. } | ScenarioStep::AdvanceL1Block { .. } => {}
            }
        }

        let left_latest = self.left.block(BlockId::Latest)?;
        let right_latest = self.right.block(BlockId::Latest)?;
        let max_n = left_latest.number.min(right_latest.number);

        for n in 0..=max_n {
            self.diff_block(n, &mut report)?;
        }

        if left_latest.number != right_latest.number {
            report.block_diffs.push(BlockDiff {
                number: max_n,
                field: "presence".to_string(),
                left: serde_json::json!(left_latest.number),
                right: serde_json::json!(right_latest.number),
            });
        }

        Ok(report)
    }

    fn diff_block(&self, number: u64, report: &mut DiffReport) -> Result<()> {
        let left = self.left.block(BlockId::Number(number)).ok();
        let right = self.right.block(BlockId::Number(number)).ok();

        match (left, right) {
            (Some(l), Some(r)) => {
                push_block_field(number, "gas_used", &l.gas_used, &r.gas_used, report);
                push_block_field(number, "state_root", &l.state_root, &r.state_root, report);
                push_block_field(
                    number,
                    "receipts_root",
                    &l.receipts_root,
                    &r.receipts_root,
                    report,
                );
                push_block_field(
                    number,
                    "transactions_root",
                    &l.transactions_root,
                    &r.transactions_root,
                    report,
                );
                push_block_field(number, "parent_hash", &l.parent_hash, &r.parent_hash, report);
                push_block_field(number, "timestamp", &l.timestamp, &r.timestamp, report);

                let tx_pairs = pair_tx_hashes(&l.tx_hashes, &r.tx_hashes);
                if l.tx_hashes.len() != r.tx_hashes.len() {
                    report.block_diffs.push(BlockDiff {
                        number,
                        field: "tx_count".to_string(),
                        left: serde_json::json!(l.tx_hashes.len()),
                        right: serde_json::json!(r.tx_hashes.len()),
                    });
                }
                for pair in tx_pairs {
                    self.diff_tx(pair, report);
                }
            }
            (Some(_), None) => {
                report.block_diffs.push(BlockDiff {
                    number,
                    field: "presence".to_string(),
                    left: serde_json::Value::Bool(true),
                    right: serde_json::Value::Bool(false),
                });
            }
            (None, Some(_)) => {
                report.block_diffs.push(BlockDiff {
                    number,
                    field: "presence".to_string(),
                    left: serde_json::Value::Bool(false),
                    right: serde_json::Value::Bool(true),
                });
            }
            (None, None) => {}
        }
        Ok(())
    }

    fn diff_tx(&self, pair: TxPair, report: &mut DiffReport) {
        match pair {
            TxPair::Both(hash) => {
                let left = self.left.receipt(hash);
                let right = self.right.receipt(hash);
                match (left, right) {
                    (Ok(l), Ok(r)) => diff_receipt(hash, &l, &r, report),
                    (Err(le), Err(re)) => report.tx_diffs.push(TxDiff {
                        tx_hash: hash,
                        field: "fetch".into(),
                        left: serde_json::json!(le.to_string()),
                        right: serde_json::json!(re.to_string()),
                    }),
                    (Err(e), Ok(_)) => report.tx_diffs.push(TxDiff {
                        tx_hash: hash,
                        field: "fetch".into(),
                        left: serde_json::json!(e.to_string()),
                        right: serde_json::Value::Null,
                    }),
                    (Ok(_), Err(e)) => report.tx_diffs.push(TxDiff {
                        tx_hash: hash,
                        field: "fetch".into(),
                        left: serde_json::Value::Null,
                        right: serde_json::json!(e.to_string()),
                    }),
                }
            }
            TxPair::LeftOnly(hash) => {
                report.tx_diffs.push(TxDiff {
                    tx_hash: hash,
                    field: "presence".into(),
                    left: serde_json::Value::Bool(true),
                    right: serde_json::Value::Bool(false),
                });
            }
            TxPair::RightOnly(hash) => {
                report.tx_diffs.push(TxDiff {
                    tx_hash: hash,
                    field: "presence".into(),
                    left: serde_json::Value::Bool(false),
                    right: serde_json::Value::Bool(true),
                });
            }
        }
    }
}

#[derive(Debug, Clone)]
enum TxPair {
    Both(B256),
    LeftOnly(B256),
    RightOnly(B256),
}

fn pair_tx_hashes(left: &[B256], right: &[B256]) -> Vec<TxPair> {
    let mut out: Vec<TxPair> = Vec::new();
    let n = left.len().min(right.len());
    for i in 0..n {
        if left[i] == right[i] {
            out.push(TxPair::Both(left[i]));
        } else {
            out.push(TxPair::LeftOnly(left[i]));
            out.push(TxPair::RightOnly(right[i]));
        }
    }
    for h in left.iter().skip(n) {
        out.push(TxPair::LeftOnly(*h));
    }
    for h in right.iter().skip(n) {
        out.push(TxPair::RightOnly(*h));
    }
    out
}

fn diff_receipt(hash: B256, l: &TxReceipt, r: &TxReceipt, report: &mut DiffReport) {
    push_tx_field(hash, "status", &l.status, &r.status, report);
    push_tx_field(hash, "gas_used", &l.gas_used, &r.gas_used, report);
    push_tx_field(
        hash,
        "cumulative_gas_used",
        &l.cumulative_gas_used,
        &r.cumulative_gas_used,
        report,
    );
    push_tx_field(
        hash,
        "effective_gas_price",
        &l.effective_gas_price,
        &r.effective_gas_price,
        report,
    );
    push_tx_field(
        hash,
        "contract_address",
        &l.contract_address,
        &r.contract_address,
        report,
    );

    if l.logs.len() != r.logs.len() {
        report.tx_diffs.push(TxDiff {
            tx_hash: hash,
            field: "log_count".into(),
            left: serde_json::json!(l.logs.len()),
            right: serde_json::json!(r.logs.len()),
        });
    }
    let n = l.logs.len().min(r.logs.len());
    for i in 0..n {
        diff_log(l.block_number.max(r.block_number), &l.logs[i], &r.logs[i], report);
    }
}

fn diff_log(block_number: u64, l: &EvmLog, r: &EvmLog, report: &mut DiffReport) {
    let log_index = l.log_index.max(r.log_index);
    push_log_field(block_number, log_index, "address", &l.address, &r.address, report);
    push_log_field(block_number, log_index, "topics", &l.topics, &r.topics, report);
    push_log_field(block_number, log_index, "data", &l.data, &r.data, report);
}

fn push_block_field<T: PartialEq + serde::Serialize>(
    number: u64,
    field: &str,
    left: &T,
    right: &T,
    report: &mut DiffReport,
) {
    if let Some(d) = check_block_field(number, field, left, right) {
        report.block_diffs.push(d);
    }
}

fn push_tx_field<T: PartialEq + serde::Serialize>(
    tx_hash: B256,
    field: &str,
    left: &T,
    right: &T,
    report: &mut DiffReport,
) {
    if let Some(d) = check_tx_field(tx_hash, field, left, right) {
        report.tx_diffs.push(d);
    }
}

fn push_log_field<T: PartialEq + serde::Serialize>(
    block_number: u64,
    log_index: u64,
    field: &str,
    left: &T,
    right: &T,
    report: &mut DiffReport,
) {
    if left != right {
        report.log_diffs.push(LogDiff {
            block_number,
            log_index,
            field: field.to_string(),
            left: serde_json::to_value(left).unwrap_or(serde_json::Value::Null),
            right: serde_json::to_value(right).unwrap_or(serde_json::Value::Null),
        });
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
