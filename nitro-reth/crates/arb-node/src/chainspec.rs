//! Chain spec parser that pre-populates the genesis `alloc` with
//! ArbOS state when the spec declares `config.arbitrum.InitialArbOSVersion`
//! but does not include the ArbOS state account in the alloc.

use std::{path::Path, str::FromStr, sync::Arc};

use alloy_genesis::GenesisAccount;
use alloy_primitives::{hex, Address, B256, U256};
use eyre::eyre;
use reth_chainspec::ChainSpec;
use reth_cli::chainspec::ChainSpecParser;
use reth_ethereum_cli::chainspec::EthereumChainSpecParser;
use revm::database::{EmptyDB, State, StateBuilder};
use revm_database::states::bundle_state::BundleRetention;
use serde_json::Value;

use arbos::arbos_types::ParsedInitMessage;

use crate::genesis;

/// Block gas limit used by Nitro at genesis (`l2pricing.GethBlockGasLimit = 1 << 50`).
const NITRO_GENESIS_GAS_LIMIT: u64 = 1 << 50;
/// Initial L2 base fee in wei used by Nitro at genesis (`l2pricing.InitialBaseFeeWei = 0.1 gwei`).
const NITRO_GENESIS_BASE_FEE: u64 = 100_000_000;
/// JSON pointer for the flag that suppresses ArbOS alloc injection (used when the genesis already carries a complete pre-seeded alloc).
const SKIP_GENESIS_INJECTION_POINTER: &str = "/config/arbitrum/SkipGenesisInjection";

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

        let allow_debug = value
            .pointer("/config/arbitrum/AllowDebugPrecompiles")
            .and_then(Value::as_bool)
            .unwrap_or(false);
        arb_precompiles::set_allow_debug_precompiles(allow_debug);

        let skip_injection = value
            .pointer(SKIP_GENESIS_INJECTION_POINTER)
            .and_then(Value::as_bool)
            .unwrap_or(false);

        if initial_arbos > 0 && chain_id > 0 {
            if !skip_injection {
                inject_arbos_alloc(
                    &mut value,
                    chain_id,
                    initial_arbos,
                    initial_owner,
                    arbos_init,
                )?;
            }
            override_arbos_genesis_header(&mut value, initial_arbos)?;
        }

        let augmented = serde_json::to_string(&value)?;
        EthereumChainSpecParser::parse(&augmented)
    }
}

/// Force the genesis header fields that Nitro hardcodes in `MakeGenesisBlock`.
/// reth's `make_genesis_header` reads these directly from the JSON, so the
/// only way to keep arbreth and Nitro in sync without forking reth is to
/// rewrite them here before parsing.
fn override_arbos_genesis_header(value: &mut Value, arbos_version: u64) -> eyre::Result<()> {
    let obj = value
        .as_object_mut()
        .ok_or_else(|| eyre!("chain spec is not a JSON object"))?;

    // Genesis header reads the L1 init message → nonce field is set to 1.
    obj.insert("nonce".into(), Value::String("0x1".into()));

    // `extraData` carries `SendRoot` (32 zero bytes at genesis).
    obj.insert(
        "extraData".into(),
        Value::String(format!("0x{}", hex::encode([0u8; 32]))),
    );

    // `mixHash` packs `[SendCount(8) | L1BlockNumber(8) | ArbOSFormatVersion(8) | flags(8)]`
    // big-endian. At genesis only `ArbOSFormatVersion` is non-zero.
    let mut mix_hash = [0u8; 32];
    mix_hash[16..24].copy_from_slice(&arbos_version.to_be_bytes());
    obj.insert(
        "mixHash".into(),
        Value::String(format!("0x{}", hex::encode(mix_hash))),
    );

    obj.insert("difficulty".into(), Value::String("0x1".into()));
    obj.insert(
        "gasLimit".into(),
        Value::String(format!("{NITRO_GENESIS_GAS_LIMIT:#x}")),
    );
    obj.insert(
        "baseFeePerGas".into(),
        Value::String(format!("{NITRO_GENESIS_BASE_FEE:#x}")),
    );
    obj.insert(
        "coinbase".into(),
        Value::String(format!("0x{}", hex::encode([0u8; 20]))),
    );

    Ok(())
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

    let entries = compute_arbos_alloc(chain_id, arbos_version, chain_owner, arbos_init)?;
    for (addr, account) in entries {
        let key = address_lower_no_prefix(addr);
        let prefixed = format!("0x{key}");
        let existing_key = if alloc_obj.contains_key(&key) {
            Some(key.clone())
        } else if alloc_obj.contains_key(&prefixed) {
            Some(prefixed.clone())
        } else {
            None
        };
        let injected = serde_json::to_value(&account)?;
        match existing_key {
            None => {
                alloc_obj.insert(prefixed, injected);
            }
            Some(k) => {
                // Merge injected entry into the user-supplied one. User-set
                // fields (balance, nonce, code, individual storage slots)
                // win on conflict so fixture overrides replace bootstrap
                // values; injected fields fill in anything the user didn't
                // specify.
                let user = alloc_obj.get_mut(&k).unwrap();
                if !user.is_object() || !injected.is_object() {
                    continue;
                }
                let user_obj = user.as_object_mut().unwrap();
                let injected_obj = injected.as_object().unwrap();
                for (field, val) in injected_obj {
                    if field == "storage" {
                        continue;
                    }
                    user_obj.entry(field.clone()).or_insert(val.clone());
                }
                let injected_storage = injected
                    .get("storage")
                    .and_then(|s| s.as_object())
                    .cloned()
                    .unwrap_or_default();
                let storage = user_obj
                    .entry("storage")
                    .or_insert_with(|| Value::Object(serde_json::Map::new()))
                    .as_object_mut()
                    .ok_or_else(|| eyre!("alloc[{k}].storage is not an object"))?;
                for (slot, val) in injected_storage {
                    storage.entry(slot).or_insert(val);
                }
            }
        }
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
