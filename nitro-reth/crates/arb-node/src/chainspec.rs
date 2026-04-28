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
/// JSON pointer for the flag that suppresses ArbOS alloc injection (used when the genesis already
/// carries a complete pre-seeded alloc).
const SKIP_GENESIS_INJECTION_POINTER: &str = "/config/arbitrum/SkipGenesisInjection";
/// Default initial L1 base fee Nitro uses when no override is provided.
/// Mirrors `arbostypes.DefaultInitialL1BaseFee = 50 * params.GWei`.
const DEFAULT_INITIAL_L1_BASE_FEE_WEI: u64 = 50_000_000_000;

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
            // SkipGenesisInjection used to gate the call entirely, but the
            // captured cache files miss accounts (FilteredTransactionsState)
            // and never carry storage for the ArbOS state account. The
            // injection helper merges per-account: user-supplied fields and
            // explicit slots win on conflict, so it is safe to run
            // unconditionally and always produces the trie Nitro would.
            let _ = skip_injection;
            inject_arbos_alloc(
                &mut value,
                chain_id,
                initial_arbos,
                initial_owner,
                arbos_init,
            )?;
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
    // Pre-compute the Go-canonical chain config bytes so the resulting
    // `chain_config` subspace slots match what Nitro writes when it parses
    // the same chain spec via `json.Unmarshal` + `json.Marshal`.
    let serialized_chain_config = value
        .get("config")
        .map(serialize_chain_config_go_style)
        .unwrap_or_default();

    let alloc_obj = value
        .as_object_mut()
        .ok_or_else(|| eyre!("chain spec is not a JSON object"))?
        .entry("alloc")
        .or_insert_with(|| Value::Object(serde_json::Map::new()))
        .as_object_mut()
        .ok_or_else(|| eyre!("alloc is not a JSON object"))?;

    let entries = compute_arbos_alloc_with_config(
        chain_id,
        arbos_version,
        chain_owner,
        arbos_init,
        serialized_chain_config,
        U256::from(DEFAULT_INITIAL_L1_BASE_FEE_WEI),
    )?;
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
///
/// Equivalent to [`compute_arbos_alloc_with_config`] called with an empty
/// chain config and a zero initial L1 base fee.
pub fn compute_arbos_alloc(
    chain_id: u64,
    arbos_version: u64,
    chain_owner: Address,
    arbos_init: genesis::ArbOSInit,
) -> eyre::Result<Vec<(Address, GenesisAccount)>> {
    compute_arbos_alloc_with_config(
        chain_id,
        arbos_version,
        chain_owner,
        arbos_init,
        Vec::new(),
        U256::ZERO,
    )
}

/// Run [`genesis::initialize_arbos_state`] in a scratch in-memory state
/// using a caller-supplied serialized chain config and initial L1 base fee.
///
/// Use this overload when reproducing the slot set Nitro writes for a
/// given chain spec; pass `serialize_chain_config_go_style(value["config"])`
/// for the bytes and `DEFAULT_INITIAL_L1_BASE_FEE_WEI` for the base fee.
pub fn compute_arbos_alloc_with_config(
    chain_id: u64,
    arbos_version: u64,
    chain_owner: Address,
    arbos_init: genesis::ArbOSInit,
    serialized_chain_config: Vec<u8>,
    initial_l1_base_fee: U256,
) -> eyre::Result<Vec<(Address, GenesisAccount)>> {
    let mut state: State<EmptyDB> = StateBuilder::new()
        .with_database(EmptyDB::default())
        .with_bundle_update()
        .build();

    let init_msg = ParsedInitMessage {
        chain_id: U256::from(chain_id),
        initial_l1_base_fee,
        serialized_chain_config,
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

/// Serialize a chain-config JSON object the way Go's `json.Marshal` of
/// `params.ChainConfig` would. Field order, address casing, and which
/// fields are emitted (vs. dropped via `omitempty`) all match the Go
/// encoder so the resulting bytes are byte-identical to what Nitro stores
/// in the `chain_config` subspace at genesis.
pub fn serialize_chain_config_go_style(config: &Value) -> Vec<u8> {
    let mut out = Vec::with_capacity(512);
    out.push(b'{');
    let mut writer = JsonWriter::new(&mut out);

    let cfg = config.as_object();

    // Mirror the field order of `params.ChainConfig`. Number-typed fork
    // toggles map to `*big.Int` in Go, so any non-null JSON number (incl.
    // `0`) decodes to a non-nil pointer and is emitted.
    writer.write_required_chain_id(cfg);

    for (name, json_key) in BIG_INT_BLOCK_FIELDS {
        writer.write_optional_big_int(cfg, json_key, name);
    }

    if let Some(map) = cfg {
        if map.get("daoForkSupport").and_then(Value::as_bool) == Some(true) {
            writer.write_bool_field("daoForkSupport", true);
        }
    }

    for (name, json_key) in BIG_INT_BLOCK_FIELDS_AFTER_DAO {
        writer.write_optional_big_int(cfg, json_key, name);
    }

    for (name, json_key) in TIME_FIELDS {
        writer.write_optional_uint64(cfg, json_key, name);
    }

    writer.write_optional_big_int(cfg, "terminalTotalDifficulty", "terminalTotalDifficulty");

    // Fixed-size arrays don't trigger Go's omitempty, so this slot always
    // appears even when the address is all zeros.
    writer.write_address_field(
        cfg,
        "depositContractAddress",
        "depositContractAddress",
        true,
    );

    if let Some(map) = cfg {
        if map.get("enableVerkleAtGenesis").and_then(Value::as_bool) == Some(true) {
            writer.write_bool_field("enableVerkleAtGenesis", true);
        }
    }

    // Ethash and Clique are pointer types in Go, so their slots only appear
    // when present in the input. Clique always has period+epoch (no
    // omitempty); BlobScheduleConfig is omitted from arbreth output for
    // ArbOS chains since Nitro never sets it.
    if let Some(eth) = cfg.and_then(|m| m.get("ethash")).filter(|v| v.is_object()) {
        writer.write_raw_object("ethash", eth);
    }
    if let Some(clique) = cfg.and_then(|m| m.get("clique")).filter(|v| v.is_object()) {
        writer.write_clique("clique", clique);
    }

    // ArbitrumChainParams is a value type in Go, so the field is always
    // emitted (even if every sub-field is zero).
    let arbitrum = cfg
        .and_then(|m| m.get("arbitrum"))
        .filter(|v| v.is_object());
    writer.write_arbitrum(arbitrum);

    out.push(b'}');
    out
}

/// Big-int (block number) fields emitted before `daoForkSupport`.
const BIG_INT_BLOCK_FIELDS: &[(&str, &str)] = &[
    ("homesteadBlock", "homesteadBlock"),
    ("daoForkBlock", "daoForkBlock"),
];

/// Big-int (block number) fields emitted after `daoForkSupport`.
const BIG_INT_BLOCK_FIELDS_AFTER_DAO: &[(&str, &str)] = &[
    ("eip150Block", "eip150Block"),
    ("eip155Block", "eip155Block"),
    ("eip158Block", "eip158Block"),
    ("byzantiumBlock", "byzantiumBlock"),
    ("constantinopleBlock", "constantinopleBlock"),
    ("petersburgBlock", "petersburgBlock"),
    ("istanbulBlock", "istanbulBlock"),
    ("muirGlacierBlock", "muirGlacierBlock"),
    ("berlinBlock", "berlinBlock"),
    ("londonBlock", "londonBlock"),
    ("arrowGlacierBlock", "arrowGlacierBlock"),
    ("grayGlacierBlock", "grayGlacierBlock"),
    ("mergeNetsplitBlock", "mergeNetsplitBlock"),
];

/// Timestamp-typed fork fields, emitted only when present in the input.
const TIME_FIELDS: &[(&str, &str)] = &[
    ("shanghaiTime", "shanghaiTime"),
    ("cancunTime", "cancunTime"),
    ("pragueTime", "pragueTime"),
    ("osakaTime", "osakaTime"),
    ("bpo1Time", "bpo1Time"),
    ("bpo2Time", "bpo2Time"),
    ("bpo3Time", "bpo3Time"),
    ("bpo4Time", "bpo4Time"),
    ("bpo5Time", "bpo5Time"),
    ("amsterdamTime", "amsterdamTime"),
    ("verkleTime", "verkleTime"),
];

/// Order-preserving JSON-fragment writer scoped to a single object body.
struct JsonWriter<'a> {
    buf: &'a mut Vec<u8>,
    first: bool,
}

impl<'a> JsonWriter<'a> {
    fn new(buf: &'a mut Vec<u8>) -> Self {
        Self { buf, first: true }
    }

    fn comma(&mut self) {
        if self.first {
            self.first = false;
        } else {
            self.buf.push(b',');
        }
    }

    fn write_key(&mut self, name: &str) {
        self.comma();
        self.buf.push(b'"');
        self.buf.extend_from_slice(name.as_bytes());
        self.buf.extend_from_slice(b"\":");
    }

    fn write_required_chain_id(&mut self, cfg: Option<&serde_json::Map<String, Value>>) {
        let chain_id = cfg
            .and_then(|m| m.get("chainId"))
            .map(value_to_decimal_int)
            .unwrap_or_else(|| "0".to_string());
        self.write_key("chainId");
        self.buf.extend_from_slice(chain_id.as_bytes());
    }

    fn write_optional_big_int(
        &mut self,
        cfg: Option<&serde_json::Map<String, Value>>,
        json_key: &str,
        name: &str,
    ) {
        let map = match cfg {
            Some(m) => m,
            None => return,
        };
        let v = match map.get(json_key) {
            Some(v) if !v.is_null() => v,
            _ => return,
        };
        let s = value_to_decimal_int(v);
        self.write_key(name);
        self.buf.extend_from_slice(s.as_bytes());
    }

    fn write_optional_uint64(
        &mut self,
        cfg: Option<&serde_json::Map<String, Value>>,
        json_key: &str,
        name: &str,
    ) {
        let map = match cfg {
            Some(m) => m,
            None => return,
        };
        let v = match map.get(json_key) {
            Some(v) if !v.is_null() => v,
            _ => return,
        };
        // Pointer-typed timestamps emit their value when non-nil; even a
        // value of 0 should appear so we keep parity with Go's encoder.
        let s = value_to_decimal_int(v);
        self.write_key(name);
        self.buf.extend_from_slice(s.as_bytes());
    }

    fn write_bool_field(&mut self, name: &str, val: bool) {
        self.write_key(name);
        self.buf
            .extend_from_slice(if val { b"true" } else { b"false" });
    }

    fn write_address_field(
        &mut self,
        cfg: Option<&serde_json::Map<String, Value>>,
        json_key: &str,
        name: &str,
        emit_zero: bool,
    ) {
        let addr = cfg
            .and_then(|m| m.get(json_key))
            .and_then(Value::as_str)
            .map(|s| s.trim_start_matches("0x").to_lowercase())
            .unwrap_or_default();
        let normalized = pad_address_lower(&addr);
        if !emit_zero && normalized == "0".repeat(40) {
            return;
        }
        self.write_key(name);
        self.buf.push(b'"');
        self.buf.extend_from_slice(b"0x");
        self.buf.extend_from_slice(normalized.as_bytes());
        self.buf.push(b'"');
    }

    fn write_raw_object(&mut self, name: &str, val: &Value) {
        // Used for value-stable JSON sub-objects (e.g. ethash struct
        // which has no fields). Writes `"name":{}` to match Go's encoder.
        self.write_key(name);
        if let Some(obj) = val.as_object() {
            if obj.is_empty() {
                self.buf.extend_from_slice(b"{}");
                return;
            }
            // Fall back to serde_json for unknown shapes (best-effort).
            let bytes = serde_json::to_vec(val).unwrap_or_else(|_| b"{}".to_vec());
            self.buf.extend_from_slice(&bytes);
        } else {
            self.buf.extend_from_slice(b"null");
        }
    }

    fn write_clique(&mut self, name: &str, val: &Value) {
        self.write_key(name);
        let map = val.as_object();
        let period = map
            .and_then(|m| m.get("period"))
            .map(value_to_decimal_int)
            .unwrap_or_else(|| "0".to_string());
        let epoch = map
            .and_then(|m| m.get("epoch"))
            .map(value_to_decimal_int)
            .unwrap_or_else(|| "0".to_string());
        self.buf.extend_from_slice(b"{\"period\":");
        self.buf.extend_from_slice(period.as_bytes());
        self.buf.extend_from_slice(b",\"epoch\":");
        self.buf.extend_from_slice(epoch.as_bytes());
        self.buf.push(b'}');
    }

    fn write_arbitrum(&mut self, arbitrum: Option<&Value>) {
        self.write_key("arbitrum");
        self.buf.push(b'{');
        let mut inner = JsonWriter::new(self.buf);
        let map = arbitrum.and_then(Value::as_object);

        // The first six fields have no `omitempty` tag, so they always
        // appear even when set to false / 0 / zero address.
        let always_bool = [
            "EnableArbOS",
            "AllowDebugPrecompiles",
            "DataAvailabilityCommittee",
        ];
        for key in always_bool {
            let v = map
                .and_then(|m| m.get(key))
                .and_then(Value::as_bool)
                .unwrap_or(false);
            inner.write_bool_field(key, v);
        }

        let initial_arbos = map
            .and_then(|m| m.get("InitialArbOSVersion"))
            .map(value_to_decimal_int)
            .unwrap_or_else(|| "0".to_string());
        inner.write_key("InitialArbOSVersion");
        inner.buf.extend_from_slice(initial_arbos.as_bytes());

        inner.write_address_field(map, "InitialChainOwner", "InitialChainOwner", true);

        let genesis_block = map
            .and_then(|m| m.get("GenesisBlockNum"))
            .map(value_to_decimal_int)
            .unwrap_or_else(|| "0".to_string());
        inner.write_key("GenesisBlockNum");
        inner.buf.extend_from_slice(genesis_block.as_bytes());

        // The remaining numeric fields use `omitempty`, so a zero value
        // is dropped to match Go's encoder.
        for json_key in ["MaxCodeSize", "MaxInitCodeSize", "MaxUncompressedBatchSize"] {
            if let Some(v) = map.and_then(|m| m.get(json_key)) {
                let n = value_to_decimal_int(v);
                if n != "0" {
                    inner.write_key(json_key);
                    inner.buf.extend_from_slice(n.as_bytes());
                }
            }
        }

        self.buf.push(b'}');
    }
}

fn value_to_decimal_int(v: &Value) -> String {
    match v {
        Value::Number(n) => {
            if let Some(u) = n.as_u64() {
                return u.to_string();
            }
            if let Some(i) = n.as_i64() {
                return i.to_string();
            }
            n.to_string()
        }
        Value::String(s) => {
            if let Some(rest) = s.strip_prefix("0x").or_else(|| s.strip_prefix("0X")) {
                return U256::from_str_radix(rest, 16)
                    .map(|x| x.to_string())
                    .unwrap_or_else(|_| "0".to_string());
            }
            s.clone()
        }
        Value::Bool(b) => (*b as u64).to_string(),
        _ => "0".to_string(),
    }
}

fn pad_address_lower(s: &str) -> String {
    let trimmed = s.trim_start_matches("0x").to_lowercase();
    if trimmed.len() >= 40 {
        return trimmed[trimmed.len() - 40..].to_string();
    }
    let mut out = String::with_capacity(40);
    for _ in 0..(40 - trimmed.len()) {
        out.push('0');
    }
    out.push_str(&trimmed);
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn serialize_chain_config_matches_nitro_v10_default_layout() {
        // Mirrors what `build_chain_config` in the Docker harness emits;
        // the byte sequence here is exactly what the Nitro container
        // writes to slots 0x...7700+ at genesis.
        let cfg = json!({
            "chainId": 421614,
            "homesteadBlock": 0,
            "daoForkSupport": true,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "muirGlacierBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "depositContractAddress": "0x0000000000000000000000000000000000000000",
            "clique": {"period": 0, "epoch": 0},
            "arbitrum": {
                "EnableArbOS": true,
                "AllowDebugPrecompiles": false,
                "DataAvailabilityCommittee": false,
                "InitialArbOSVersion": 10,
                "InitialChainOwner": "0x71B61c2E250AFa05dFc36304D6c91501bE0965D8",
                "GenesisBlockNum": 0u64,
            }
        });
        let bytes = serialize_chain_config_go_style(&cfg);
        let s = std::str::from_utf8(&bytes).unwrap();

        let expected = "{\"chainId\":421614,\"homesteadBlock\":0,\"daoForkSupport\":true,\"eip150Block\":0,\"eip155Block\":0,\"eip158Block\":0,\"byzantiumBlock\":0,\"constantinopleBlock\":0,\"petersburgBlock\":0,\"istanbulBlock\":0,\"muirGlacierBlock\":0,\"berlinBlock\":0,\"londonBlock\":0,\"depositContractAddress\":\"0x0000000000000000000000000000000000000000\",\"clique\":{\"period\":0,\"epoch\":0},\"arbitrum\":{\"EnableArbOS\":true,\"AllowDebugPrecompiles\":false,\"DataAvailabilityCommittee\":false,\"InitialArbOSVersion\":10,\"InitialChainOwner\":\"0x71b61c2e250afa05dfc36304d6c91501be0965d8\",\"GenesisBlockNum\":0}}";
        assert_eq!(s, expected, "Nitro-canonical chain config bytes mismatch");
        assert_eq!(bytes.len(), 549, "expected 549-byte serialization");
    }

    #[test]
    fn serialize_skips_null_and_default_fields() {
        // Pointer-typed fields with `null` should be skipped; bool fields
        // default to `false` and are skipped via Go's omitempty for bool.
        let cfg = json!({
            "chainId": 421614,
            "homesteadBlock": 0,
            "daoForkBlock": null,
            "daoForkSupport": false,
            "eip150Block": 0,
            "arbitrum": {
                "InitialArbOSVersion": 10,
            }
        });
        let bytes = serialize_chain_config_go_style(&cfg);
        let s = std::str::from_utf8(&bytes).unwrap();
        assert!(!s.contains("daoForkBlock"));
        assert!(!s.contains("daoForkSupport"));
        // Defaults still emit the 6 required Arbitrum fields.
        assert!(s.contains("\"EnableArbOS\":false"));
        assert!(s.contains("\"GenesisBlockNum\":0"));
        assert!(s.contains("\"InitialChainOwner\":\"0x0000000000000000000000000000000000000000\""));
    }

    #[test]
    fn serialize_includes_terminal_total_difficulty_when_set() {
        let cfg = json!({
            "chainId": 421614,
            "terminalTotalDifficulty": 0,
            "arbitrum": { "InitialArbOSVersion": 10 }
        });
        let bytes = serialize_chain_config_go_style(&cfg);
        let s = std::str::from_utf8(&bytes).unwrap();
        assert!(s.contains("\"terminalTotalDifficulty\":0"));
    }

    #[test]
    fn serialize_lowercases_addresses_and_strips_prefix() {
        let cfg = json!({
            "chainId": 1,
            "depositContractAddress": "0xABCDEF0000000000000000000000000000000123",
            "arbitrum": {
                "InitialChainOwner": "0xABCDEF0000000000000000000000000000000123",
            }
        });
        let bytes = serialize_chain_config_go_style(&cfg);
        let s = std::str::from_utf8(&bytes).unwrap();
        assert!(
            s.contains("\"depositContractAddress\":\"0xabcdef0000000000000000000000000000000123\"")
        );
        assert!(s.contains("\"InitialChainOwner\":\"0xabcdef0000000000000000000000000000000123\""));
    }
}
