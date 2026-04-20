//! Chain spec parser that pre-populates the genesis `alloc` with
//! ArbOS state when the spec declares `config.arbitrum.InitialArbOSVersion`
//! but does not include the ArbOS state account in the alloc.

use std::{path::Path, str::FromStr, sync::Arc};

use alloy_genesis::GenesisAccount;
use alloy_primitives::{Address, B256, U256};
use eyre::eyre;
use reth_chainspec::ChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_ethereum_cli::chainspec::EthereumChainSpecParser;
use revm::database::{EmptyDB, State, StateBuilder};
use revm_database::states::bundle_state::BundleRetention;
use serde_json::Value;

use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::arbos_types::ParsedInitMessage;

use crate::genesis;

#[derive(Debug, Clone, Default)]
#[non_exhaustive]
pub struct ArbChainSpecParser;

impl ChainSpecParser for ArbChainSpecParser {
    type ChainSpec = ChainSpec;

    const SUPPORTED_CHAINS: &'static [&'static str] = EthereumChainSpecParser::SUPPORTED_CHAINS;

    fn parse(s: &str) -> eyre::Result<Arc<ChainSpec>> {
        if EthereumChainSpecParser::SUPPORTED_CHAINS.contains(&s) {
            return EthereumChainSpecParser::parse(s);
        }

        let raw = if Path::new(s).exists() {
            std::fs::read_to_string(s).map_err(|e| eyre!("read chain spec {s}: {e}"))?
        } else {
            s.to_string()
        };

        let mut value: Value =
            serde_json::from_str(&raw).map_err(|e| eyre!("parse chain spec JSON: {e}"))?;

        let initial_arbos = value
            .pointer("/config/arbitrum/InitialArbOSVersion")
            .and_then(Value::as_u64)
            .unwrap_or(0);
        let chain_id = value
            .pointer("/config/chainId")
            .and_then(Value::as_u64)
            .unwrap_or(0);
        let initial_owner = value
            .pointer("/config/arbitrum/InitialChainOwner")
            .and_then(Value::as_str)
            .and_then(|s| Address::from_str(s.trim_start_matches("0x")).ok())
            .unwrap_or(Address::ZERO);
        let arbos_init = parse_arbos_init(&value);

        if initial_arbos > 0 && chain_id > 0 {
            inject_arbos_alloc(&mut value, chain_id, initial_arbos, initial_owner, arbos_init)?;
        }

        let augmented = serde_json::to_string(&value)?;
        EthereumChainSpecParser::parse(&augmented)
    }
}

fn parse_arbos_init(value: &Value) -> genesis::ArbOSInit {
    let native = value
        .pointer("/config/arbitrum/ArbOSInit/nativeTokenSupplyManagementEnabled")
        .or_else(|| value.pointer("/config/arbitrum/nativeTokenSupplyManagementEnabled"))
        .and_then(Value::as_bool)
        .unwrap_or(false);
    let filtering = value
        .pointer("/config/arbitrum/ArbOSInit/transactionFilteringEnabled")
        .or_else(|| value.pointer("/config/arbitrum/transactionFilteringEnabled"))
        .and_then(Value::as_bool)
        .unwrap_or(false);
    genesis::ArbOSInit {
        native_token_supply_management_enabled: native,
        transaction_filtering_enabled: filtering,
    }
}

fn inject_arbos_alloc(
    value: &mut Value,
    chain_id: u64,
    arbos_version: u64,
    chain_owner: Address,
    arbos_init: genesis::ArbOSInit,
) -> eyre::Result<()> {
    let alloc_obj = value
        .as_object_mut()
        .ok_or_else(|| eyre!("chain spec is not a JSON object"))?
        .entry("alloc")
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .ok_or_else(|| eyre!("alloc is not a JSON object"))?;

    let arbos_addr_key = address_lower_no_prefix(ARBOS_STATE_ADDRESS);
    if alloc_obj.contains_key(&arbos_addr_key)
        || alloc_obj.contains_key(&format!("0x{arbos_addr_key}"))
    {
        return Ok(());
    }

    let entries = compute_arbos_alloc(chain_id, arbos_version, chain_owner, arbos_init)?;
    for (addr, account) in entries {
        let key = address_lower_no_prefix(addr);
        if alloc_obj.contains_key(&key) || alloc_obj.contains_key(&format!("0x{key}")) {
            continue;
        }
        let json = serde_json::to_value(account)?;
        alloc_obj.insert(format!("0x{key}"), json);
    }
    Ok(())
}

fn address_lower_no_prefix(addr: Address) -> String {
    let s = format!("{addr:x}");
    let mut padded = String::with_capacity(40);
    for _ in 0..(40 - s.len()) {
        padded.push('0');
    }
    padded.push_str(&s);
    padded
}

/// Run [`genesis::initialize_arbos_state`] in a scratch in-memory state
/// and dump the resulting account/storage map. Returns one entry per
/// account touched (the ArbOS state address plus all genesis precompile
/// markers).
pub fn compute_arbos_alloc(
    chain_id: u64,
    arbos_version: u64,
    chain_owner: Address,
    arbos_init: genesis::ArbOSInit,
) -> eyre::Result<Vec<(Address, GenesisAccount)>> {
    let mut state: State<EmptyDB> = StateBuilder::new()
        .with_database(EmptyDB::default())
        .with_bundle_update()
        .build();

    let init_msg = ParsedInitMessage {
        chain_id: U256::from(chain_id),
        initial_l1_base_fee: U256::ZERO,
        serialized_chain_config: Vec::new(),
    };

    genesis::initialize_arbos_state(
        &mut state,
        &init_msg,
        chain_id,
        arbos_version,
        chain_owner,
        arbos_init,
    )
    .map_err(|e| eyre!("initialize_arbos_state: {e}"))?;

    state.merge_transitions(BundleRetention::PlainState);
    let bundle = state.take_bundle();

    let mut out = Vec::new();
    for (addr, account) in bundle.state.iter() {
        let info = match &account.info {
            Some(info) => info,
            None => continue,
        };

        let mut storage = std::collections::BTreeMap::new();
        for (slot, slot_value) in account.storage.iter() {
            if slot_value.present_value.is_zero() {
                continue;
            }
            storage.insert(
                B256::from(slot.to_be_bytes::<32>()),
                B256::from(slot_value.present_value.to_be_bytes::<32>()),
            );
        }

        let code = match &info.code {
            Some(c) if !c.original_bytes().is_empty() => Some(c.original_bytes()),
            _ => None,
        };

        let entry = GenesisAccount {
            balance: info.balance,
            nonce: Some(info.nonce),
            code,
            storage: if storage.is_empty() {
                None
            } else {
                Some(storage)
            },
            private_key: None,
        };
        out.push((*addr, entry));
    }
    out.sort_by_key(|(a, _)| *a);
    Ok(out)
}
