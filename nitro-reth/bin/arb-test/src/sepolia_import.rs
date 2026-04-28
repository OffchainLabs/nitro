use std::{collections::BTreeMap, path::PathBuf};

use anyhow::{anyhow, Context, Result};
use clap::Subcommand;
use serde_json::{json, Map, Value};

#[derive(Debug, Subcommand)]
pub enum SepoliaImportCommand {
    /// Refresh `alloc[*].storage` in an existing fixture against live RPC.
    RefreshStorage(RefreshArgs),
}

#[derive(Debug, clap::Args)]
pub struct RefreshArgs {
    /// Path to the fixture JSON to update in place.
    #[arg(long)]
    pub fixture: PathBuf,
    /// Transaction hash whose state we are reproducing.
    #[arg(long)]
    pub tx: String,
    /// Archive RPC URL.
    #[arg(long, env = "ARB_SEPOLIA_RPC")]
    pub rpc: String,
}

pub fn run(cmd: SepoliaImportCommand) -> Result<()> {
    match cmd {
        SepoliaImportCommand::RefreshStorage(args) => refresh_storage(args),
    }
}

fn refresh_storage(args: RefreshArgs) -> Result<()> {
    let mut fixture: Value = serde_json::from_slice(&std::fs::read(&args.fixture)?)
        .with_context(|| format!("read fixture {}", args.fixture.display()))?;

    let rpc = Rpc::new(args.rpc);

    let tx = rpc.call("eth_getTransactionByHash", json!([args.tx]))?;
    let block_hex = tx
        .get("blockNumber")
        .and_then(Value::as_str)
        .ok_or_else(|| anyhow!("tx has no blockNumber"))?;
    let block_num = u64::from_str_radix(block_hex.trim_start_matches("0x"), 16)?;
    let parent_num = block_num
        .checked_sub(1)
        .ok_or_else(|| anyhow!("tx is in genesis block"))?;
    let parent_hex = format!("0x{:x}", parent_num);
    eprintln!(
        "tx {} mined in block {} ({}); refreshing storage at parent {}",
        args.tx, block_hex, block_num, parent_hex
    );

    let call_tree = rpc.call(
        "debug_traceTransaction",
        json!([args.tx, {"tracer": "callTracer", "tracerConfig": {"onlyTopCall": false}}]),
    )?;
    let mut storage_addrs: Vec<String> = Vec::new();
    collect_storage_addrs(&call_tree, &lower(tx_to(&tx)?), &mut storage_addrs);
    storage_addrs.sort();
    storage_addrs.dedup();
    eprintln!(
        "storage contexts in call tree ({}):\n  {}",
        storage_addrs.len(),
        storage_addrs.join("\n  ")
    );

    let prestate = rpc.call(
        "debug_traceTransaction",
        json!([args.tx, {"tracer": "prestateTracer", "tracerConfig": {"diffMode": false}}]),
    )?;

    let alloc_obj = fixture
        .pointer_mut("/genesis/alloc")
        .and_then(Value::as_object_mut)
        .ok_or_else(|| anyhow!("fixture has no /genesis/alloc"))?;

    let mut summary: Vec<String> = Vec::new();
    for addr in &storage_addrs {
        let entry = alloc_obj
            .entry(addr.clone())
            .or_insert_with(|| json!({"balance": "0x0", "nonce": "0x1"}));
        let entry_obj = entry
            .as_object_mut()
            .ok_or_else(|| anyhow!("alloc[{addr}] is not an object"))?;
        let existing_storage = entry_obj
            .get("storage")
            .and_then(Value::as_object)
            .cloned()
            .unwrap_or_default();
        let mut candidate_slots: Vec<String> = existing_storage.keys().cloned().collect();
        for src in candidates_for(addr, &prestate) {
            candidate_slots.push(src);
        }
        candidate_slots.sort();
        candidate_slots.dedup();

        let mut new_storage: BTreeMap<String, String> = BTreeMap::new();
        let mut nonzero = 0usize;
        for slot in &candidate_slots {
            let v = rpc.call("eth_getStorageAt", json!([addr, slot, parent_hex.clone()]))?;
            let val = v
                .as_str()
                .ok_or_else(|| anyhow!("eth_getStorageAt returned non-string"))?;
            if val != "0x0000000000000000000000000000000000000000000000000000000000000000"
                && val != "0x0"
            {
                new_storage.insert(slot.clone(), val.to_string());
                nonzero += 1;
            }
        }
        let new_obj: Map<String, Value> = new_storage
            .into_iter()
            .map(|(k, v)| (k, Value::String(v)))
            .collect();
        if new_obj.is_empty() {
            entry_obj.remove("storage");
        } else {
            entry_obj.insert("storage".into(), Value::Object(new_obj));
        }
        summary.push(format!(
            "  {addr}: {nonzero} non-zero / {} candidates",
            candidate_slots.len()
        ));
    }

    eprintln!("storage refresh summary:");
    for line in &summary {
        eprintln!("{line}");
    }

    let pretty = serde_json::to_string_pretty(&fixture)?;
    std::fs::write(&args.fixture, pretty.as_bytes())?;
    eprintln!("wrote {}", args.fixture.display());
    Ok(())
}

fn tx_to(tx: &Value) -> Result<String> {
    tx.get("to")
        .and_then(Value::as_str)
        .map(String::from)
        .ok_or_else(|| anyhow!("tx has no `to` (contract creation not supported)"))
}

/// DFS the call tree from a callTracer trace and append every frame's
/// storage-context address to `out`. DELEGATECALL/CALLCODE inherit the
/// caller's storage; CALL/STATICCALL switch to the callee.
fn collect_storage_addrs(node: &Value, current_storage: &str, out: &mut Vec<String>) {
    let kind = node.get("type").and_then(Value::as_str).unwrap_or("CALL");
    let to = node
        .get("to")
        .and_then(Value::as_str)
        .map(lower)
        .unwrap_or_default();
    let storage_addr = match kind {
        "DELEGATECALL" | "CALLCODE" => current_storage.to_string(),
        _ => {
            if to.is_empty() {
                current_storage.to_string()
            } else {
                to.clone()
            }
        }
    };
    if !storage_addr.is_empty() {
        out.push(storage_addr.clone());
    }
    if let Some(calls) = node.get("calls").and_then(Value::as_array) {
        for child in calls {
            collect_storage_addrs(child, &storage_addr, out);
        }
    }
}

fn candidates_for(addr: &str, prestate: &Value) -> Vec<String> {
    let mut out: Vec<String> = Vec::new();
    if let Some(map) = prestate.as_object() {
        for (key, val) in map {
            if !addr_eq(key, addr) {
                if let Some(code) = val.get("code").and_then(Value::as_str) {
                    if code != "0x" {
                        continue;
                    }
                }
            }
            if let Some(storage) = val.get("storage").and_then(Value::as_object) {
                for slot in storage.keys() {
                    out.push(slot.clone());
                }
            }
        }
    }
    out
}

fn lower(s: impl AsRef<str>) -> String {
    s.as_ref().to_ascii_lowercase()
}

fn addr_eq(a: &str, b: &str) -> bool {
    a.eq_ignore_ascii_case(b)
}

struct Rpc {
    url: String,
    agent: ureq::Agent,
}

impl Rpc {
    fn new(url: String) -> Self {
        Self {
            url,
            agent: ureq::AgentBuilder::new()
                .timeout(std::time::Duration::from_secs(60))
                .build(),
        }
    }

    fn call(&self, method: &str, params: Value) -> Result<Value> {
        let body = json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params,
        });
        let resp: Value = self
            .agent
            .post(&self.url)
            .set("Content-Type", "application/json")
            .send_json(body)
            .with_context(|| format!("rpc {method}"))?
            .into_json()
            .with_context(|| format!("rpc {method} body"))?;
        if let Some(err) = resp.get("error") {
            return Err(anyhow!("rpc {method} error: {err}"));
        }
        resp.get("result")
            .cloned()
            .ok_or_else(|| anyhow!("rpc {method} missing result"))
    }
}
