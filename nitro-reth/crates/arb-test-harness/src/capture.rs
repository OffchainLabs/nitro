use serde_json::{json, Map, Value};

use crate::{
    node::{BlockId, EvmLog, ExecutionNode, TxReceipt},
    scenario::{Scenario, ScenarioStep},
    Result,
};

#[derive(Debug, Clone)]
pub struct CapturedScenario {
    pub scenario: Scenario,
    pub expected_json: Value,
}

pub fn capture_from_node(
    node: &mut dyn ExecutionNode,
    scenario: &Scenario,
) -> Result<CapturedScenario> {
    for step in &scenario.steps {
        match step {
            ScenarioStep::Message {
                idx,
                message,
                delayed_messages_read,
            } => {
                node.submit_message(*idx, message, *delayed_messages_read)?;
            }
            ScenarioStep::AdvanceTime { .. } | ScenarioStep::AdvanceL1Block { .. } => {}
        }
    }

    let latest = node.block(BlockId::Latest)?;

    let mut blocks: Vec<Value> = Vec::new();
    let mut tx_receipts: Vec<Value> = Vec::new();

    let start = if latest.number == 0 { 0 } else { 1 };
    for n in start..=latest.number {
        let block = node.block(BlockId::Number(n))?;
        blocks.push(block_to_expected_json(&block));

        for tx_hash in &block.tx_hashes {
            match node.receipt(*tx_hash) {
                Ok(receipt) => {
                    let arb = node.arb_receipt(*tx_hash).ok();
                    tx_receipts.push(receipt_to_expected_json(&receipt, arb.as_ref()));
                }
                Err(_) => continue,
            }
        }
    }

    let expected = json!({
        "blocks": blocks,
        "txReceipts": tx_receipts,
    });

    Ok(CapturedScenario {
        scenario: scenario.clone(),
        expected_json: expected,
    })
}

fn block_to_expected_json(block: &crate::node::Block) -> Value {
    json!({
        "number": block.number,
        "block_hash": format!("{:#x}", block.hash),
        "state_root": format!("{:#x}", block.state_root),
        "receipts_root": format!("{:#x}", block.receipts_root),
        "transactions_root": format!("{:#x}", block.transactions_root),
        "gas_used": block.gas_used,
    })
}

fn receipt_to_expected_json(
    receipt: &TxReceipt,
    arb: Option<&crate::node::ArbReceiptFields>,
) -> Value {
    let mut map = Map::new();
    map.insert(
        "txHash".into(),
        Value::String(format!("{:#x}", receipt.tx_hash)),
    );
    map.insert("blockNumber".into(), json!(receipt.block_number));
    map.insert("status".into(), json!(receipt.status));
    map.insert("gasUsed".into(), json!(receipt.gas_used));
    map.insert(
        "cumulativeGasUsed".into(),
        json!(receipt.cumulative_gas_used),
    );
    map.insert(
        "effectiveGasPrice".into(),
        json!(receipt.effective_gas_price),
    );
    if let Some(addr) = receipt.contract_address {
        map.insert(
            "contractAddress".into(),
            Value::String(format!("{addr:#x}")),
        );
    }
    map.insert("from".into(), Value::String(format!("{:#x}", receipt.from)));
    if let Some(to) = receipt.to {
        map.insert("to".into(), Value::String(format!("{to:#x}")));
    }
    if let Some(arb) = arb {
        if let Some(g) = arb.gas_used_for_l1 {
            map.insert("gasUsedForL1".into(), json!(g));
        }
        if let Some(b) = arb.l1_block_number {
            map.insert("l1BlockNumber".into(), json!(b));
        }
        if let Some(mg) = &arb.multi_gas {
            map.insert(
                "multiGas".into(),
                json!({
                    "computation": mg.computation,
                    "history": mg.history,
                    "storage": mg.storage,
                    "stateGrowth": mg.state_growth,
                }),
            );
        }
    }
    if !receipt.logs.is_empty() {
        let logs: Vec<Value> = receipt.logs.iter().map(log_to_expected_json).collect();
        map.insert("logs".into(), Value::Array(logs));
    }
    Value::Object(map)
}

fn log_to_expected_json(log: &EvmLog) -> Value {
    let topics: Vec<String> = log.topics.iter().map(|t| format!("{t:#x}")).collect();
    json!({
        "address": format!("{:#x}", log.address),
        "topics": topics,
        "data": format!("0x{}", hex::encode(&log.data)),
        "blockNumber": log.block_number,
        "txHash": format!("{:#x}", log.tx_hash),
        "logIndex": log.log_index,
    })
}
