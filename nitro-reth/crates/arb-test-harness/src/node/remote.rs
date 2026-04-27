use std::{collections::BTreeMap, time::Duration};

use alloy_primitives::{Address, Bytes, B256, U256};
use serde_json::{json, Value};

use crate::{
    error::HarnessError,
    messaging::L1Message,
    node::{
        ArbReceiptFields, Block, BlockId, EvmLog, ExecutionNode, NodeKind, TxReceipt, TxRequest,
    },
    rpc::JsonRpcClient,
    Result,
};

#[derive(Debug, Clone)]
pub struct RemoteNode {
    rpc_url: String,
    rpc: JsonRpcClient,
    kind: NodeKind,
}

impl RemoteNode {
    pub fn connect(rpc_url: impl Into<String>, kind: NodeKind) -> Self {
        let url = rpc_url.into();
        let rpc = JsonRpcClient::new(url.clone()).with_timeout(Duration::from_secs(60));
        Self {
            rpc_url: url,
            rpc,
            kind,
        }
    }

    pub fn arbreth(rpc_url: impl Into<String>) -> Self {
        Self::connect(rpc_url, NodeKind::Arbreth)
    }

    pub fn nitro(rpc_url: impl Into<String>) -> Self {
        Self::connect(rpc_url, NodeKind::NitroLocal)
    }
}

impl ExecutionNode for RemoteNode {
    fn kind(&self) -> NodeKind {
        self.kind
    }

    fn rpc_url(&self) -> &str {
        &self.rpc_url
    }

    fn submit_message(
        &mut self,
        idx: u64,
        msg: &L1Message,
        delayed_messages_read: u64,
    ) -> Result<()> {
        let params = json!([
            idx,
            {
                "message": {
                    "header": &msg.header,
                    "l2Msg": &msg.l2_msg,
                },
                "delayedMessagesRead": delayed_messages_read,
            },
            Value::Null,
        ]);
        self.rpc.call("nitroexecution_digestMessage", params)?;
        Ok(())
    }

    fn block(&self, id: BlockId) -> Result<Block> {
        let v = self
            .rpc
            .call("eth_getBlockByNumber", json!([id.to_rpc(), false]))?;
        block_from_json(&v)
    }

    fn receipt(&self, tx: B256) -> Result<TxReceipt> {
        let v = self
            .rpc
            .call("eth_getTransactionReceipt", json!([format!("{tx:#x}")]))?;
        receipt_from_json(&v)
    }

    fn arb_receipt(&self, tx: B256) -> Result<ArbReceiptFields> {
        let v = self
            .rpc
            .call("eth_getTransactionReceipt", json!([format!("{tx:#x}")]))?;
        Ok(arb_receipt_fields(&v))
    }

    fn storage(&self, addr: Address, slot: B256, at: BlockId) -> Result<B256> {
        let v = self.rpc.call(
            "eth_getStorageAt",
            json!([format!("{addr:#x}"), format!("{slot:#x}"), at.to_rpc()]),
        )?;
        json_to_b256(&v)
    }

    fn balance(&self, addr: Address, at: BlockId) -> Result<U256> {
        let v = self.rpc.call(
            "eth_getBalance",
            json!([format!("{addr:#x}"), at.to_rpc()]),
        )?;
        json_to_u256(&v)
    }

    fn nonce(&self, addr: Address, at: BlockId) -> Result<u64> {
        let v = self.rpc.call(
            "eth_getTransactionCount",
            json!([format!("{addr:#x}"), at.to_rpc()]),
        )?;
        json_to_u64(&v)
    }

    fn code(&self, addr: Address, at: BlockId) -> Result<Bytes> {
        let v = self.rpc.call(
            "eth_getCode",
            json!([format!("{addr:#x}"), at.to_rpc()]),
        )?;
        json_to_bytes(&v)
    }

    fn eth_call(&self, tx: TxRequest, at: BlockId) -> Result<Bytes> {
        let v = self.rpc.call(
            "eth_call",
            json!([tx_request_to_json(&tx), at.to_rpc()]),
        )?;
        json_to_bytes(&v)
    }

    fn debug_storage_range(
        &self,
        addr: Address,
        at: BlockId,
    ) -> Result<BTreeMap<B256, B256>> {
        let block = self.block(at.clone())?;
        let v = self.rpc.call(
            "debug_storageRangeAt",
            json!([
                format!("{:#x}", block.hash),
                0,
                format!("{addr:#x}"),
                format!("{:#x}", B256::ZERO),
                u32::MAX,
            ]),
        )?;
        let mut out = BTreeMap::new();
        if let Some(map) = v.get("storage").and_then(|s| s.as_object()) {
            for entry in map.values() {
                let key = entry.get("key").and_then(|x| x.as_str());
                let val = entry.get("value").and_then(|x| x.as_str());
                if let (Some(k), Some(v)) = (key, val) {
                    let k = parse_b256(k)?;
                    let v = parse_b256(v)?;
                    out.insert(k, v);
                }
            }
        }
        Ok(out)
    }

    fn shutdown(self: Box<Self>) -> Result<()> {
        Ok(())
    }
}

fn tx_request_to_json(tx: &TxRequest) -> Value {
    let mut map = serde_json::Map::new();
    if let Some(to) = tx.to {
        map.insert("to".into(), Value::String(format!("{to:#x}")));
    }
    if let Some(from) = tx.from {
        map.insert("from".into(), Value::String(format!("{from:#x}")));
    }
    if let Some(data) = &tx.data {
        map.insert("data".into(), Value::String(format!("0x{}", hex::encode(data))));
    }
    if let Some(value) = tx.value {
        map.insert("value".into(), Value::String(format!("0x{value:x}")));
    }
    if let Some(gas) = tx.gas {
        map.insert("gas".into(), Value::String(format!("0x{gas:x}")));
    }
    Value::Object(map)
}

fn block_from_json(v: &Value) -> Result<Block> {
    if v.is_null() {
        return Err(HarnessError::Rpc("block not found".into()));
    }
    Ok(Block {
        number: v
            .get("number")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        hash: v
            .get("hash")
            .map(json_to_b256)
            .transpose()?
            .unwrap_or(B256::ZERO),
        parent_hash: v
            .get("parentHash")
            .map(json_to_b256)
            .transpose()?
            .unwrap_or(B256::ZERO),
        state_root: v
            .get("stateRoot")
            .map(json_to_b256)
            .transpose()?
            .unwrap_or(B256::ZERO),
        receipts_root: v
            .get("receiptsRoot")
            .map(json_to_b256)
            .transpose()?
            .unwrap_or(B256::ZERO),
        transactions_root: v
            .get("transactionsRoot")
            .map(json_to_b256)
            .transpose()?
            .unwrap_or(B256::ZERO),
        gas_used: v
            .get("gasUsed")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        gas_limit: v
            .get("gasLimit")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        timestamp: v
            .get("timestamp")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        tx_hashes: extract_tx_hashes(v),
    })
}

fn extract_tx_hashes(v: &Value) -> Vec<B256> {
    let Some(arr) = v.get("transactions").and_then(|t| t.as_array()) else {
        return Vec::new();
    };
    let mut out = Vec::with_capacity(arr.len());
    for entry in arr {
        let hash_str = match entry {
            Value::String(s) => Some(s.as_str()),
            Value::Object(map) => map.get("hash").and_then(|h| h.as_str()),
            _ => None,
        };
        if let Some(s) = hash_str {
            if let Ok(h) = s.parse::<B256>() {
                out.push(h);
            }
        }
    }
    out
}

fn receipt_from_json(v: &Value) -> Result<TxReceipt> {
    if v.is_null() {
        return Err(HarnessError::Rpc("receipt not found".into()));
    }
    Ok(TxReceipt {
        tx_hash: v
            .get("transactionHash")
            .map(json_to_b256)
            .transpose()?
            .unwrap_or(B256::ZERO),
        block_number: v
            .get("blockNumber")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        status: v
            .get("status")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0) as u8,
        gas_used: v
            .get("gasUsed")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        cumulative_gas_used: v
            .get("cumulativeGasUsed")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        effective_gas_price: v
            .get("effectiveGasPrice")
            .and_then(|x| x.as_str())
            .and_then(|s| u128::from_str_radix(s.trim_start_matches("0x"), 16).ok())
            .unwrap_or(0),
        from: v
            .get("from")
            .and_then(|x| x.as_str())
            .and_then(|s| s.parse::<Address>().ok())
            .unwrap_or_default(),
        to: v
            .get("to")
            .and_then(|x| x.as_str())
            .and_then(|s| s.parse::<Address>().ok()),
        contract_address: v
            .get("contractAddress")
            .and_then(|x| x.as_str())
            .and_then(|s| s.parse::<Address>().ok()),
        logs: extract_logs(v),
    })
}

fn extract_logs(v: &Value) -> Vec<EvmLog> {
    let Some(arr) = v.get("logs").and_then(|l| l.as_array()) else {
        return Vec::new();
    };
    let mut out = Vec::with_capacity(arr.len());
    for entry in arr {
        let address = entry
            .get("address")
            .and_then(|x| x.as_str())
            .and_then(|s| s.parse::<Address>().ok())
            .unwrap_or_default();
        let topics = entry
            .get("topics")
            .and_then(|x| x.as_array())
            .map(|a| {
                a.iter()
                    .filter_map(|t| t.as_str().and_then(|s| s.parse::<B256>().ok()))
                    .collect()
            })
            .unwrap_or_default();
        let data = entry
            .get("data")
            .and_then(|x| x.as_str())
            .and_then(|s| {
                hex::decode(s.trim_start_matches("0x")).ok().map(Bytes::from)
            })
            .unwrap_or_default();
        let log_index = entry
            .get("logIndex")
            .and_then(|x| x.as_str())
            .and_then(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).ok())
            .unwrap_or(0);
        let block_number = entry
            .get("blockNumber")
            .and_then(|x| x.as_str())
            .and_then(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).ok())
            .unwrap_or(0);
        let tx_hash = entry
            .get("transactionHash")
            .and_then(|x| x.as_str())
            .and_then(|s| s.parse::<B256>().ok())
            .unwrap_or_default();
        out.push(EvmLog {
            address,
            topics,
            data,
            log_index,
            block_number,
            tx_hash,
        });
    }
    out
}

fn arb_receipt_fields(v: &Value) -> ArbReceiptFields {
    ArbReceiptFields {
        gas_used_for_l1: v
            .get("gasUsedForL1")
            .and_then(|x| x.as_str())
            .and_then(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).ok()),
        l1_block_number: v
            .get("l1BlockNumber")
            .and_then(|x| x.as_str())
            .and_then(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).ok()),
        multi_gas: None,
    }
}

fn parse_b256(s: &str) -> Result<B256> {
    s.parse::<B256>()
        .map_err(|e| HarnessError::Rpc(format!("invalid B256 {s}: {e}")))
}

fn json_to_u64(v: &Value) -> Result<u64> {
    let s = v
        .as_str()
        .ok_or_else(|| HarnessError::Rpc(format!("expected hex string, got {v}")))?;
    u64::from_str_radix(s.trim_start_matches("0x"), 16)
        .map_err(|e| HarnessError::Rpc(format!("invalid u64 hex {s}: {e}")))
}

fn json_to_u256(v: &Value) -> Result<U256> {
    let s = v
        .as_str()
        .ok_or_else(|| HarnessError::Rpc(format!("expected hex string, got {v}")))?;
    U256::from_str_radix(s.trim_start_matches("0x"), 16)
        .map_err(|e| HarnessError::Rpc(format!("invalid u256 hex {s}: {e}")))
}

fn json_to_b256(v: &Value) -> Result<B256> {
    let s = v
        .as_str()
        .ok_or_else(|| HarnessError::Rpc(format!("expected hex string, got {v}")))?;
    parse_b256(s)
}

fn json_to_bytes(v: &Value) -> Result<Bytes> {
    let s = v
        .as_str()
        .ok_or_else(|| HarnessError::Rpc(format!("expected hex string, got {v}")))?;
    let stripped = s.trim_start_matches("0x");
    let bytes = if stripped.is_empty() {
        Vec::new()
    } else {
        hex::decode(stripped).map_err(|e| HarnessError::Rpc(format!("invalid hex: {e}")))?
    };
    Ok(Bytes::from(bytes))
}
