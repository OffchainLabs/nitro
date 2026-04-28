use std::collections::BTreeMap;

use alloy_primitives::{Address, Bytes, B256, U256};
use serde::{Deserialize, Serialize};
use serde_json::{json, Map, Value};

use arbos::header::ARBOS_STATE_ADDRESS;

use crate::Result;

#[derive(Debug, Clone)]
pub struct GenesisBuilder {
    pub l2_chain_id: u64,
    pub initial_arbos_version: u64,
    pub initial_block_num: u64,
    pub allow_debug_precompiles: bool,
    pub initial_chain_owner: Address,
    pub gas_limit: u64,
    pub base_fee_per_gas: u64,
    pub timestamp: u64,
    pub allocations: BTreeMap<Address, AccountAlloc>,
    pub arbos_storage: BTreeMap<B256, B256>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AccountAlloc {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub balance: Option<U256>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nonce: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub code: Option<Bytes>,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub storage: BTreeMap<B256, B256>,
}

const DEFAULT_OWNER: Address = {
    let mut bytes = [0u8; 20];
    bytes[0] = 0x71;
    bytes[1] = 0xb6;
    bytes[2] = 0x1c;
    bytes[3] = 0x2e;
    bytes[4] = 0x25;
    bytes[5] = 0x0a;
    bytes[6] = 0xfa;
    bytes[7] = 0x05;
    bytes[8] = 0xdf;
    bytes[9] = 0xc3;
    bytes[10] = 0x63;
    bytes[11] = 0x04;
    bytes[12] = 0xd6;
    bytes[13] = 0xc9;
    bytes[14] = 0x15;
    bytes[15] = 0x01;
    bytes[16] = 0xbe;
    bytes[17] = 0x09;
    bytes[18] = 0x65;
    bytes[19] = 0xd8;
    Address::new(bytes)
};

const DEFAULT_GAS_LIMIT: u64 = 0x4000000000000;
const DEFAULT_BASE_FEE: u64 = 0x5f5e100;

impl GenesisBuilder {
    pub fn new(l2_chain_id: u64, initial_arbos_version: u64) -> Self {
        Self {
            l2_chain_id,
            initial_arbos_version,
            initial_block_num: 0,
            allow_debug_precompiles: true,
            initial_chain_owner: DEFAULT_OWNER,
            gas_limit: DEFAULT_GAS_LIMIT,
            base_fee_per_gas: DEFAULT_BASE_FEE,
            timestamp: 0,
            allocations: BTreeMap::new(),
            arbos_storage: BTreeMap::new(),
        }
    }

    pub fn with_arbos_slot(mut self, slot: B256, value: B256) -> Self {
        self.arbos_storage.insert(slot, value);
        self
    }

    pub fn build(&self) -> Result<Value> {
        let alloc = self.render_alloc();

        let config = json!({
            "chainId": self.l2_chain_id,
            "homesteadBlock": 0,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "arbitrum": {
                "EnableArbOS": true,
                "AllowDebugPrecompiles": self.allow_debug_precompiles,
                "DataAvailabilityCommittee": false,
                "InitialArbOSVersion": self.initial_arbos_version,
                "InitialChainOwner": format!("{:#x}", self.initial_chain_owner),
                "GenesisBlockNum": self.initial_block_num,
            },
        });

        Ok(json!({
            "config": config,
            "nonce": "0x1",
            "timestamp": format!("0x{:x}", self.timestamp),
            "gasLimit": format!("0x{:x}", self.gas_limit),
            "difficulty": "0x1",
            "mixHash": format!(
                "0x000000000000000000000000000000000000000000000000{:016x}",
                self.initial_arbos_version << 48,
            ),
            "coinbase": "0x0000000000000000000000000000000000000000",
            "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
            "baseFeePerGas": format!("0x{:x}", self.base_fee_per_gas),
            "alloc": alloc,
        }))
    }

    fn render_alloc(&self) -> Map<String, Value> {
        let mut alloc = Map::new();

        let mut user_entries: BTreeMap<Address, Value> = BTreeMap::new();
        for (addr, account) in &self.allocations {
            user_entries.insert(*addr, alloc_value(account));
        }

        for byte in 0x64u8..=0x70 {
            let mut a = [0u8; 20];
            a[19] = byte;
            user_entries
                .entry(Address::new(a))
                .or_insert_with(|| sentinel_alloc());
        }
        let mut ff = [0u8; 20];
        ff[19] = 0xff;
        user_entries
            .entry(Address::new(ff))
            .or_insert_with(|| sentinel_alloc());

        let mut a4b05_short = [0u8; 20];
        a4b05_short[17] = 0x0a;
        a4b05_short[18] = 0x4b;
        a4b05_short[19] = 0x05;
        user_entries
            .entry(Address::new(a4b05_short))
            .or_insert_with(|| sentinel_alloc());

        let arbos_state_addr = ARBOS_STATE_ADDRESS;
        let mut arbos_entry = Map::new();
        arbos_entry.insert("balance".into(), Value::String("0x0".into()));
        arbos_entry.insert("nonce".into(), Value::Number(1u64.into()));
        if !self.arbos_storage.is_empty() {
            let mut storage = Map::new();
            for (slot, value) in &self.arbos_storage {
                storage.insert(format!("{slot:#x}"), Value::String(format!("{value:#x}")));
            }
            arbos_entry.insert("storage".into(), Value::Object(storage));
        }
        user_entries
            .entry(arbos_state_addr)
            .and_modify(|v| {
                if let Value::Object(map) = v {
                    if !map.contains_key("nonce") {
                        map.insert("nonce".into(), Value::Number(1u64.into()));
                    }
                }
            })
            .or_insert(Value::Object(arbos_entry));

        for (addr, value) in user_entries {
            alloc.insert(addr_alloc_key(&addr), value);
        }

        alloc
    }
}

fn addr_alloc_key(addr: &Address) -> String {
    hex::encode(addr.as_slice())
}

fn sentinel_alloc() -> Value {
    json!({
        "code": "0xfe",
        "balance": "0x0",
    })
}

fn alloc_value(alloc: &AccountAlloc) -> Value {
    let mut map = Map::new();
    map.insert(
        "balance".into(),
        Value::String(format!("0x{:x}", alloc.balance.unwrap_or(U256::ZERO))),
    );
    if let Some(nonce) = alloc.nonce {
        if nonce > 0 {
            map.insert("nonce".into(), Value::Number(nonce.into()));
        }
    }
    if let Some(code) = &alloc.code {
        if !code.is_empty() {
            map.insert(
                "code".into(),
                Value::String(format!("0x{}", hex::encode(code))),
            );
        }
    }
    if !alloc.storage.is_empty() {
        let mut storage = Map::new();
        for (slot, value) in &alloc.storage {
            storage.insert(format!("{slot:#x}"), Value::String(format!("{value:#x}")));
        }
        map.insert("storage".into(), Value::Object(storage));
    }
    Value::Object(map)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn build_emits_expected_top_level_fields() {
        let g = GenesisBuilder::new(421614, 10).build().unwrap();
        let cfg = g.get("config").unwrap();
        assert_eq!(cfg.get("chainId").unwrap().as_u64().unwrap(), 421614);
        let arb = cfg.get("arbitrum").unwrap();
        assert_eq!(arb.get("InitialArbOSVersion").unwrap().as_u64().unwrap(), 10);
        assert!(arb.get("AllowDebugPrecompiles").unwrap().as_bool().unwrap());
        assert_eq!(g.get("difficulty").unwrap().as_str().unwrap(), "0x1");
        assert!(g.get("alloc").unwrap().as_object().unwrap().len() >= 14);
    }

    #[test]
    fn arbos_storage_seeds_appear_under_state_address() {
        let slot = B256::with_last_byte(1);
        let value = B256::with_last_byte(2);
        let g = GenesisBuilder::new(1, 10)
            .with_arbos_slot(slot, value)
            .build()
            .unwrap();
        let alloc = g.get("alloc").unwrap().as_object().unwrap();
        let key = hex::encode(ARBOS_STATE_ADDRESS.as_slice());
        let entry = alloc.get(&key).unwrap();
        let storage = entry.get("storage").unwrap().as_object().unwrap();
        assert_eq!(storage.len(), 1);
    }
}
