//! Build chain-spec JSON from typed inputs.
//!
//! Replaces the prior `generate_genesis.py`. Composes a Nitro/arbreth
//! genesis JSON from an [`arb_chainspec::ArbosVersion`] plus an
//! account allocation map plus optional ArbOS-storage seeds.

use std::collections::BTreeMap;

use alloy_primitives::{Address, B256, U256};
use serde::{Deserialize, Serialize};

use crate::{error::HarnessError, Result};

#[derive(Debug, Clone)]
pub struct GenesisBuilder {
    pub l2_chain_id: u64,
    pub initial_arbos_version: u64,
    pub initial_block_num: u64,
    pub allow_debug_precompiles: bool,
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
    pub code: Option<alloy_primitives::Bytes>,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub storage: BTreeMap<B256, B256>,
}

impl GenesisBuilder {
    pub fn new(l2_chain_id: u64, initial_arbos_version: u64) -> Self {
        Self {
            l2_chain_id,
            initial_arbos_version,
            initial_block_num: 0,
            allow_debug_precompiles: true,
            allocations: BTreeMap::new(),
            arbos_storage: BTreeMap::new(),
        }
    }

    pub fn with_account(mut self, addr: Address, alloc: AccountAlloc) -> Self {
        self.allocations.insert(addr, alloc);
        self
    }

    pub fn with_arbos_slot(mut self, slot: B256, value: B256) -> Self {
        self.arbos_storage.insert(slot, value);
        self
    }

    pub fn build(&self) -> Result<serde_json::Value> {
        Err(HarnessError::NotImplemented {
            what: "GenesisBuilder::build (Stage 2 / Agent A)",
        })
    }
}
