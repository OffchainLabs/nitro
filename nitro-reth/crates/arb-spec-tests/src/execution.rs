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

use crate::case::SpecError;

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
        Ok(())
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
