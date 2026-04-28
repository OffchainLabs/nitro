use std::sync::atomic::AtomicU16;

use alloy_primitives::{Address, Bytes, B256, U256};
use serde_json::Value;

use crate::{
    error::HarnessError,
    node::{ArbReceiptFields, Block, EvmLog, TxReceipt, TxRequest},
    Result,
};

pub(crate) fn tx_request_to_json(tx: &TxRequest) -> Value {
    let mut map = serde_json::Map::new();
    if let Some(to) = tx.to {
        map.insert("to".into(), Value::String(format!("{to:#x}")));
    }
    if let Some(from) = tx.from {
        map.insert("from".into(), Value::String(format!("{from:#x}")));
    }
    if let Some(data) = &tx.data {
        map.insert(
            "data".into(),
            Value::String(format!("0x{}", hex::encode(data))),
        );
    }
    if let Some(value) = tx.value {
        map.insert("value".into(), Value::String(format!("0x{value:x}")));
    }
    if let Some(gas) = tx.gas {
        map.insert("gas".into(), Value::String(format!("0x{gas:x}")));
    }
    Value::Object(map)
}

pub(crate) fn block_from_json(v: &Value) -> Result<Block> {
    if v.is_null() {
        return Err(HarnessError::Rpc("block not found".into()));
    }
    Ok(Block {
        number: v.get("number").map(json_to_u64).transpose()?.unwrap_or(0),
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
        gas_used: v.get("gasUsed").map(json_to_u64).transpose()?.unwrap_or(0),
        gas_limit: v.get("gasLimit").map(json_to_u64).transpose()?.unwrap_or(0),
        timestamp: v
            .get("timestamp")
            .map(json_to_u64)
            .transpose()?
            .unwrap_or(0),
        tx_hashes: extract_tx_hashes(v),
    })
}

pub(crate) fn extract_tx_hashes(v: &Value) -> Vec<B256> {
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

pub(crate) fn receipt_from_json(v: &Value) -> Result<TxReceipt> {
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
        status: v.get("status").map(json_to_u64).transpose()?.unwrap_or(0) as u8,
        gas_used: v.get("gasUsed").map(json_to_u64).transpose()?.unwrap_or(0),
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

pub(crate) fn extract_logs(v: &Value) -> Vec<EvmLog> {
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
                hex::decode(s.trim_start_matches("0x"))
                    .ok()
                    .map(Bytes::from)
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

pub(crate) fn arb_receipt_fields(v: &Value) -> ArbReceiptFields {
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

pub(crate) fn parse_b256(s: &str) -> Result<B256> {
    s.parse::<B256>()
        .map_err(|e| HarnessError::Rpc(format!("invalid B256 {s}: {e}")))
}

pub(crate) fn tail(s: &str, max: usize) -> &str {
    if s.len() <= max {
        s
    } else {
        &s[s.len() - max..]
    }
}

pub(crate) fn free_tcp_port(_counter: &AtomicU16) -> Result<u16> {
    let listener = std::net::TcpListener::bind(("127.0.0.1", 0)).map_err(HarnessError::Io)?;
    let port = listener.local_addr().map_err(HarnessError::Io)?.port();
    drop(listener);
    Ok(port)
}

pub(crate) fn json_to_u64(v: &Value) -> Result<u64> {
    let s = v
        .as_str()
        .ok_or_else(|| HarnessError::Rpc(format!("expected hex string, got {v}")))?;
    u64::from_str_radix(s.trim_start_matches("0x"), 16)
        .map_err(|e| HarnessError::Rpc(format!("invalid u64 hex {s}: {e}")))
}

pub(crate) fn json_to_u256(v: &Value) -> Result<U256> {
    let s = v
        .as_str()
        .ok_or_else(|| HarnessError::Rpc(format!("expected hex string, got {v}")))?;
    U256::from_str_radix(s.trim_start_matches("0x"), 16)
        .map_err(|e| HarnessError::Rpc(format!("invalid u256 hex {s}: {e}")))
}

pub(crate) fn json_to_b256(v: &Value) -> Result<B256> {
    let s = v
        .as_str()
        .ok_or_else(|| HarnessError::Rpc(format!("expected hex string, got {v}")))?;
    parse_b256(s)
}

pub(crate) fn json_to_bytes(v: &Value) -> Result<Bytes> {
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
