//! RPC-driven execution fixtures. Each JSON file contains a list of
//! L1 incoming messages and per-block / eth_call / storage / balance
//! assertions; the runner replays them against the endpoint in
//! `ARB_SPEC_RPC_URL`.

use std::{
    io::Read,
    path::Path,
    time::{Duration, Instant},
};

use alloy_primitives::{Address, Bytes, B256, U256};
use serde::{Deserialize, Serialize};

use arb_test_harness::{
    capture::capture_from_node,
    dual_exec::DualExec,
    mock_l1::MockL1,
    node::{nitro_docker::NitroDocker, remote::RemoteNode, NodeStartCtx},
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

use crate::{case::SpecError, mode::FixtureMode};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionFixture {
    pub name: String,
    #[serde(default)]
    pub description: String,
    /// Optional inline chain spec JSON. When present the runner spawns
    /// a fresh arbreth process against this genesis so the fixture is
    /// self-contained from L2 block 0.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub genesis: Option<serde_json::Value>,
    pub messages: Vec<ExecutionMessage>,
    #[serde(default)]
    pub expected: ExecutionExpectations,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionMessage {
    #[serde(default, rename = "msgIdx")]
    pub msg_idx: Option<u64>,
    pub message: serde_json::Value,
    #[serde(default, rename = "delayedMessagesRead")]
    pub delayed_messages_read: u64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ExecutionExpectations {
    #[serde(default)]
    pub blocks: Vec<ExpectedBlock>,
    #[serde(default)]
    pub eth_calls: Vec<ExpectedEthCall>,
    #[serde(default)]
    pub storage: Vec<ExpectedStorage>,
    #[serde(default)]
    pub balances: Vec<ExpectedBalance>,
    #[serde(default, rename = "txReceipts", skip_serializing_if = "Vec::is_empty")]
    pub tx_receipts: Vec<ExpectedTxReceipt>,
    #[serde(default, rename = "stateDiffs", skip_serializing_if = "Vec::is_empty")]
    pub state_diffs: Vec<ExpectedStateDiff>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub logs: Vec<ExpectedLog>,
    #[serde(
        default,
        rename = "acceptedDiffs",
        skip_serializing_if = "Vec::is_empty"
    )]
    pub accepted_diffs: Vec<AcceptedDiff>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedTxReceipt {
    #[serde(rename = "txHash")]
    pub tx_hash: B256,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "blockNumber"
    )]
    pub block_number: Option<u64>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub status: Option<u8>,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "gasUsed")]
    pub gas_used: Option<u64>,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "cumulativeGasUsed"
    )]
    pub cumulative_gas_used: Option<u64>,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "effectiveGasPrice"
    )]
    pub effective_gas_price: Option<u128>,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "gasUsedForL1"
    )]
    pub gas_used_for_l1: Option<u64>,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "l1BlockNumber"
    )]
    pub l1_block_number: Option<u64>,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "contractAddress"
    )]
    pub contract_address: Option<Address>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub from: Option<Address>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub to: Option<Address>,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "multiGas")]
    pub multi_gas: Option<MultiGasDims>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub logs: Option<Vec<ExpectedLog>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MultiGasDims {
    pub computation: u64,
    pub history: u64,
    pub storage: u64,
    #[serde(rename = "stateGrowth")]
    pub state_growth: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedStateDiff {
    pub address: Address,
    #[serde(default = "default_block_tag", rename = "atBlock")]
    pub at_block: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub balance: Option<U256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub nonce: Option<u64>,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "codeHash")]
    pub code_hash: Option<B256>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub storage: Vec<StorageSlotExpectation>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StorageSlotExpectation {
    pub slot: B256,
    pub value: U256,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedLog {
    pub address: Address,
    pub topics: Vec<B256>,
    pub data: Bytes,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "blockNumber"
    )]
    pub block_number: Option<u64>,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "txHash")]
    pub tx_hash: Option<B256>,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "logIndex")]
    pub log_index: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AcceptedDiff {
    pub category: String,
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedBlock {
    pub number: u64,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub block_hash: Option<B256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub state_root: Option<B256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub receipts_root: Option<B256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub transactions_root: Option<B256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub gas_used: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedEthCall {
    pub to: Address,
    pub data: Bytes,
    #[serde(default = "default_block_tag")]
    pub at_block: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub result: Option<Bytes>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub result_block_hash_of: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedStorage {
    pub address: Address,
    pub slot: B256,
    #[serde(default = "default_block_tag")]
    pub at_block: String,
    pub value: U256,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedBalance {
    pub address: Address,
    #[serde(default = "default_block_tag")]
    pub at_block: String,
    pub value: U256,
}

fn default_block_tag() -> String {
    "latest".to_string()
}

impl ExecutionFixture {
    pub fn load(path: &Path) -> Result<Self, SpecError> {
        let mut s = String::new();
        std::fs::File::open(path)?.read_to_string(&mut s)?;
        Ok(serde_json::from_str(&s)?)
    }

    pub fn run(&self, rpc_url: &str) -> Result<(), SpecError> {
        let client = RpcClient::new(rpc_url);

        let mut next_idx = 1u64;
        for (i, msg) in self.messages.iter().enumerate() {
            let idx = msg.msg_idx.unwrap_or(next_idx);
            let params = serde_json::json!([
                idx,
                { "message": msg.message, "delayedMessagesRead": msg.delayed_messages_read },
                serde_json::Value::Null,
            ]);
            client
                .call::<serde_json::Value>("nitroexecution_digestMessage", params)
                .map_err(|e| {
                    SpecError::Action(format!("message {i} (idx {idx}) digest failed: {e}"))
                })?;
            next_idx = idx + 1;
        }

        for exp in &self.expected.blocks {
            verify_block(&client, exp)?;
        }
        for exp in &self.expected.eth_calls {
            verify_eth_call(&client, exp)?;
        }
        for exp in &self.expected.storage {
            verify_storage(&client, exp)?;
        }
        for exp in &self.expected.balances {
            verify_balance(&client, exp)?;
        }
        for exp in &self.expected.tx_receipts {
            verify_tx_receipt(&client, exp)?;
        }
        Ok(())
    }

    pub fn run_with_mode(&mut self, mode: FixtureMode, rpc_url: &str) -> Result<(), SpecError> {
        match mode {
            FixtureMode::Verify => self.run(rpc_url),
            FixtureMode::Record => self.record_against_nitro(),
            FixtureMode::Compare => self.compare(rpc_url),
        }
    }

    fn nitro_node_ctx(&self) -> Result<(MockL1, NodeStartCtx), SpecError> {
        let l1_chain_id: u64 = 11_155_111;
        let mock = MockL1::start(l1_chain_id)
            .map_err(|e| SpecError::Action(format!("mock-l1 start: {e}")))?;
        let l2_chain_id = self
            .genesis
            .as_ref()
            .and_then(|g| g.get("config"))
            .and_then(|c| c.get("chainId"))
            .and_then(|v| v.as_u64())
            .unwrap_or(421_614);
        let mock_l1_rpc = mock.rpc_url();
        let ctx = NodeStartCtx {
            binary: None,
            l2_chain_id,
            l1_chain_id,
            mock_l1_rpc,
            genesis: self.genesis.clone().unwrap_or_default(),
            jwt_hex: String::new(),
            workdir: std::path::PathBuf::new(),
            http_port: 0,
            authrpc_port: 0,
        };
        Ok((mock, ctx))
    }

    pub fn record_against_nitro(&mut self) -> Result<(), SpecError> {
        let (_mock, ctx) = self.nitro_node_ctx()?;
        let mut nitro =
            NitroDocker::start(&ctx).map_err(|e| SpecError::Action(format!("nitro start: {e}")))?;
        let scenario = self.to_scenario()?;
        let captured = capture_from_node(&mut nitro, &scenario)
            .map_err(|e| SpecError::Action(format!("capture: {e}")))?;

        let parsed: ExecutionExpectations = serde_json::from_value(captured.expected_json.clone())
            .map_err(|e| SpecError::Action(format!("decode captured expectations: {e}")))?;
        self.expected = parsed;

        if let Ok(out_path) = std::env::var("ARB_SPEC_RECORD_OUT") {
            let body = serde_json::to_vec_pretty(self)
                .map_err(|e| SpecError::Action(format!("encode fixture: {e}")))?;
            std::fs::write(&out_path, body)
                .map_err(|e| SpecError::Action(format!("write {out_path}: {e}")))?;
        }
        Ok(())
    }

    pub fn compare(&mut self, rpc_url: &str) -> Result<(), SpecError> {
        let (_mock, ctx) = self.nitro_node_ctx()?;
        let left =
            NitroDocker::start(&ctx).map_err(|e| SpecError::Action(format!("nitro start: {e}")))?;
        let right = RemoteNode::arbreth(rpc_url);
        let mut dual = DualExec::new(left, right);
        let scenario = self.to_scenario()?;
        let report = dual
            .run(&scenario)
            .map_err(|e| SpecError::Action(format!("dual_exec: {e}")))?;

        let filtered = filter_accepted(&report, &self.expected.accepted_diffs);
        if !filtered.is_clean() {
            return Err(SpecError::Assertion(format!(
                "compare mode: diffs after filtering accepted: {} block, {} tx, {} state, {} log",
                filtered.block_diffs.len(),
                filtered.tx_diffs.len(),
                filtered.state_diffs.len(),
                filtered.log_diffs.len(),
            )));
        }
        Ok(())
    }

    fn to_scenario(&self) -> Result<Scenario, SpecError> {
        const KIND_INITIALIZE: u8 = 11;
        let mut steps: Vec<ScenarioStep> = Vec::with_capacity(self.messages.len());
        let mut next_idx = 1u64;
        for (i, msg) in self.messages.iter().enumerate() {
            let parsed: arb_test_harness::L1Message = serde_json::from_value(msg.message.clone())
                .map_err(|e| {
                SpecError::Action(format!("message {i}: decode L1Message: {e}"))
            })?;
            if parsed.header.kind == KIND_INITIALIZE {
                continue;
            }
            steps.push(ScenarioStep::Message {
                idx: next_idx,
                message: parsed,
                delayed_messages_read: msg.delayed_messages_read,
            });
            next_idx += 1;
        }
        Ok(Scenario {
            name: self.name.clone(),
            description: self.description.clone(),
            setup: ScenarioSetup::default(),
            steps,
        })
    }
}

fn filter_accepted(
    report: &arb_test_harness::DiffReport,
    accepted: &[AcceptedDiff],
) -> arb_test_harness::DiffReport {
    let cats: std::collections::HashSet<&str> =
        accepted.iter().map(|a| a.category.as_str()).collect();
    arb_test_harness::DiffReport {
        block_diffs: report
            .block_diffs
            .iter()
            .filter(|d| !cats.contains(d.field.as_str()))
            .cloned()
            .collect(),
        tx_diffs: report
            .tx_diffs
            .iter()
            .filter(|d| !cats.contains(d.field.as_str()))
            .cloned()
            .collect(),
        state_diffs: report.state_diffs.clone(),
        log_diffs: report
            .log_diffs
            .iter()
            .filter(|d| !cats.contains(d.field.as_str()))
            .cloned()
            .collect(),
    }
}

// ─── RPC client ──────────────────────────────────────────────────────

struct RpcClient {
    url: String,
}

impl RpcClient {
    fn new(url: &str) -> Self {
        Self {
            url: url.to_string(),
        }
    }

    fn call<T: for<'de> Deserialize<'de>>(
        &self,
        method: &str,
        params: serde_json::Value,
    ) -> Result<T, String> {
        let body = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params,
        });
        let deadline = Instant::now() + Duration::from_secs(30);
        let resp = loop {
            let attempt = ureq::post(&self.url)
                .set("Content-Type", "application/json")
                .send_string(&body.to_string());
            match attempt {
                Ok(r) => break r,
                Err(e) if Instant::now() < deadline => {
                    std::thread::sleep(Duration::from_millis(200));
                    let _ = e;
                }
                Err(e) => return Err(format!("http: {e}")),
            }
        };
        let text = resp.into_string().map_err(|e| format!("read body: {e}"))?;
        let v: serde_json::Value =
            serde_json::from_str(&text).map_err(|e| format!("parse: {e} — body: {text}"))?;
        if let Some(err) = v.get("error") {
            return Err(format!("rpc error: {err}"));
        }
        let result = v
            .get("result")
            .cloned()
            .ok_or_else(|| format!("no result: {v}"))?;
        serde_json::from_value(result).map_err(|e| format!("decode: {e}"))
    }
}

fn verify_block(client: &RpcClient, exp: &ExpectedBlock) -> Result<(), SpecError> {
    let num = format!("0x{:x}", exp.number);
    let block: serde_json::Value = client
        .call("eth_getBlockByNumber", serde_json::json!([num, false]))
        .map_err(|e| SpecError::Assertion(format!("block {}: {e}", exp.number)))?;
    if block.is_null() {
        return Err(SpecError::Assertion(format!(
            "block {} not found",
            exp.number
        )));
    }
    let mut diffs: Vec<String> = Vec::new();
    if let Some(expected) = exp.block_hash {
        check_field(&block, "hash", &expected.to_string(), &mut diffs);
    }
    if let Some(expected) = exp.state_root {
        check_field(&block, "stateRoot", &expected.to_string(), &mut diffs);
    }
    if let Some(expected) = exp.receipts_root {
        check_field(&block, "receiptsRoot", &expected.to_string(), &mut diffs);
    }
    if let Some(expected) = exp.transactions_root {
        check_field(
            &block,
            "transactionsRoot",
            &expected.to_string(),
            &mut diffs,
        );
    }
    if let Some(expected) = exp.gas_used {
        let want = format!("0x{expected:x}");
        check_field(&block, "gasUsed", &want, &mut diffs);
    }
    if !diffs.is_empty() {
        return Err(SpecError::Assertion(format!(
            "block {} field mismatch:\n  {}",
            exp.number,
            diffs.join("\n  ")
        )));
    }
    Ok(())
}

fn check_field(block: &serde_json::Value, field: &str, want: &str, diffs: &mut Vec<String>) {
    let got = block
        .get(field)
        .and_then(|v| v.as_str())
        .map(|s| s.to_lowercase())
        .unwrap_or_default();
    let want_lower = want.to_lowercase();
    if got != want_lower {
        diffs.push(format!("{field}: got {got}, want {want_lower}"));
    }
}

fn verify_eth_call(client: &RpcClient, exp: &ExpectedEthCall) -> Result<(), SpecError> {
    let params = serde_json::json!([
        { "to": exp.to, "data": exp.data },
        exp.at_block,
    ]);
    let got: Bytes = client
        .call("eth_call", params)
        .map_err(|e| SpecError::Assertion(format!("eth_call {}: {e}", exp.to)))?;

    let expected = match (&exp.result, &exp.result_block_hash_of) {
        (Some(r), None) => r.clone(),
        (None, Some(block_tag)) => {
            let b: serde_json::Value = client
                .call(
                    "eth_getBlockByNumber",
                    serde_json::json!([block_tag, false]),
                )
                .map_err(|e| {
                    SpecError::Assertion(format!("eth_getBlockByNumber {block_tag}: {e}"))
                })?;
            let hash_str = b
                .get("hash")
                .and_then(|v| v.as_str())
                .ok_or_else(|| SpecError::Assertion(format!("block {block_tag} missing hash")))?;
            let bytes = alloy_primitives::hex::decode(hash_str.trim_start_matches("0x"))
                .map_err(|e| SpecError::Assertion(format!("decode hash {hash_str}: {e}")))?;
            Bytes::from(bytes)
        }
        _ => {
            return Err(SpecError::Assertion(format!(
                "eth_call {}: exactly one of `result` / `result_block_hash_of` must be set",
                exp.to
            )))
        }
    };

    if got != expected {
        return Err(SpecError::Assertion(format!(
            "eth_call {} at {} — got {}, want {}",
            exp.to, exp.at_block, got, expected,
        )));
    }
    Ok(())
}

fn verify_storage(client: &RpcClient, exp: &ExpectedStorage) -> Result<(), SpecError> {
    let params = serde_json::json!([exp.address, exp.slot, exp.at_block]);
    let got: B256 = client
        .call("eth_getStorageAt", params)
        .map_err(|e| SpecError::Assertion(format!("storage {}: {e}", exp.address)))?;
    let want = B256::from_slice(&exp.value.to_be_bytes::<32>());
    if got != want {
        return Err(SpecError::Assertion(format!(
            "storage {}[{}] at {} — got {}, want {}",
            exp.address, exp.slot, exp.at_block, got, want,
        )));
    }
    Ok(())
}

fn verify_tx_receipt(client: &RpcClient, exp: &ExpectedTxReceipt) -> Result<(), SpecError> {
    // Skip the RPC call entirely if the fixture didn't pin anything we
    // actually compare. Logs, gas_used, and status are checked when set.
    if exp.logs.is_none() && exp.gas_used.is_none() && exp.status.is_none() {
        return Ok(());
    }
    let receipt: serde_json::Value = client
        .call(
            "eth_getTransactionReceipt",
            serde_json::json!([exp.tx_hash]),
        )
        .map_err(|e| SpecError::Assertion(format!("receipt {}: {e}", exp.tx_hash)))?;
    if receipt.is_null() {
        return Err(SpecError::Assertion(format!(
            "receipt for {} not found",
            exp.tx_hash
        )));
    }

    if let Some(want_gas) = exp.gas_used {
        let got_str = receipt
            .get("gasUsed")
            .and_then(|v| v.as_str())
            .ok_or_else(|| {
                SpecError::Assertion(format!("receipt {} missing gasUsed", exp.tx_hash))
            })?;
        let got = u64::from_str_radix(got_str.trim_start_matches("0x"), 16).map_err(|e| {
            SpecError::Assertion(format!(
                "receipt {} parse gasUsed {got_str}: {e}",
                exp.tx_hash
            ))
        })?;
        if got != want_gas {
            return Err(SpecError::Assertion(format!(
                "receipt {} gasUsed: got {got}, want {want_gas} (Δ = {})",
                exp.tx_hash,
                got as i128 - want_gas as i128
            )));
        }
    }

    if let Some(want_status) = exp.status {
        let got_str = receipt
            .get("status")
            .and_then(|v| v.as_str())
            .ok_or_else(|| {
                SpecError::Assertion(format!("receipt {} missing status", exp.tx_hash))
            })?;
        let got = u8::from_str_radix(got_str.trim_start_matches("0x"), 16).map_err(|e| {
            SpecError::Assertion(format!(
                "receipt {} parse status {got_str}: {e}",
                exp.tx_hash
            ))
        })?;
        if got != want_status {
            return Err(SpecError::Assertion(format!(
                "receipt {} status: got {got}, want {want_status}",
                exp.tx_hash
            )));
        }
    }

    if let Some(want_logs) = exp.logs.as_ref() {
        let got_logs = receipt
            .get("logs")
            .and_then(|v| v.as_array())
            .ok_or_else(|| {
                SpecError::Assertion(format!("receipt {} missing logs array", exp.tx_hash))
            })?;
        if got_logs.len() != want_logs.len() {
            return Err(SpecError::Assertion(format!(
                "receipt {} log count: got {}, want {}",
                exp.tx_hash,
                got_logs.len(),
                want_logs.len()
            )));
        }
        for (i, (want, got)) in want_logs.iter().zip(got_logs.iter()).enumerate() {
            verify_log(&exp.tx_hash, i, want, got)?;
        }
    }
    Ok(())
}

fn verify_log(
    tx: &B256,
    index: usize,
    want: &ExpectedLog,
    got: &serde_json::Value,
) -> Result<(), SpecError> {
    let got_addr = got
        .get("address")
        .and_then(|v| v.as_str())
        .map(|s| s.to_lowercase())
        .unwrap_or_default();
    let want_addr = format!("{:#x}", want.address).to_lowercase();
    if got_addr != want_addr {
        return Err(SpecError::Assertion(format!(
            "tx {tx} log[{index}] address: got {got_addr}, want {want_addr}"
        )));
    }
    let got_topics: Vec<String> = got
        .get("topics")
        .and_then(|v| v.as_array())
        .map(|arr| {
            arr.iter()
                .filter_map(|t| t.as_str().map(|s| s.to_lowercase()))
                .collect()
        })
        .unwrap_or_default();
    let want_topics: Vec<String> = want
        .topics
        .iter()
        .map(|t| format!("{t:#x}").to_lowercase())
        .collect();
    if got_topics != want_topics {
        return Err(SpecError::Assertion(format!(
            "tx {tx} log[{index}] topics: got {got_topics:?}, want {want_topics:?}"
        )));
    }
    let got_data = got
        .get("data")
        .and_then(|v| v.as_str())
        .map(|s| s.to_lowercase())
        .unwrap_or_default();
    let want_data = format!("{want_data}", want_data = want.data).to_lowercase();
    if got_data != want_data {
        return Err(SpecError::Assertion(format!(
            "tx {tx} log[{index}] data mismatch:\n  got  {got_data}\n  want {want_data}"
        )));
    }
    Ok(())
}

fn verify_balance(client: &RpcClient, exp: &ExpectedBalance) -> Result<(), SpecError> {
    let params = serde_json::json!([exp.address, exp.at_block]);
    let got: U256 = client
        .call("eth_getBalance", params)
        .map_err(|e| SpecError::Assertion(format!("balance {}: {e}", exp.address)))?;
    if got != exp.value {
        return Err(SpecError::Assertion(format!(
            "balance {} at {} — got {}, want {}",
            exp.address, exp.at_block, got, exp.value,
        )));
    }
    Ok(())
}
