//! Rewrite Stylus fixture messages so each one consists of:
//!
//! 1. A `kind=12` ETH deposit funding the deterministic dev address.
//! 2. A sequence of `kind=3 sub=4` SignedL2Tx envelopes performing
//!    deploy / activate / invoke against Stylus or EVM helper programs.
//!
//! Stylus deploys carry brotli-compressed WASM bodies wrapped with the
//! 4-byte discriminant `[0xEF, 0xF0, 0x00, 0x00]` and an EVM init-code
//! prologue (mstore + return) so a CREATE tx delivers the deployable
//! contract bytecode to chain state.

mod common;
mod scenarios;
mod wat_sources;

use std::path::{Path, PathBuf};

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use anyhow::{anyhow, bail, Context, Result};
use arb_test_harness::messaging::{
    DepositBuilder, L2TxKind, MessageBuilder, SignedL2TxBuilder,
};
use std::collections::BTreeMap;
use walkdir::WalkDir;

use common::{bridge_aliased_sender, dev_address, dev_signing_key};
use scenarios::{Operation, Scenario};

const ARB_WASM_ADDR: Address = address!("0000000000000000000000000000000000000071");
const ARB_OWNER_ADDR: Address = address!("0000000000000000000000000000000000000070");

const SEQUENCER_HEADER_SENDER: Address =
    address!("a4b000000000000000000073657175656e636572");

const STYLUS_DISCRIMINANT: [u8; 3] = [0xEF, 0xF0, 0x00];
const STYLUS_DICT_BYTE: u8 = 0;

const CHAIN_ID: u64 = 421614;
const DEFAULT_GAS_LIMIT_DEPLOY: u64 = 50_000_000;
const DEFAULT_GAS_LIMIT_INVOKE: u64 = 30_000_000;
const DEFAULT_MAX_FEE: u128 = 1_000_000_000;
const DEFAULT_MAX_PRIORITY: u128 = 0;
const DEFAULT_DEPOSIT_AMOUNT: u128 = 1_000_000_000_000_000_000_000u128;
const BASE_FEE_L1: u64 = 0;

fn main() -> Result<()> {
    let workspace_root = locate_workspace_root()?;
    let fixtures_root = workspace_root.join("crates/arb-spec-tests/fixtures/stylus");
    if !fixtures_root.is_dir() {
        bail!("fixtures dir not found at {}", fixtures_root.display());
    }
    let wat_root = workspace_root.join("crates/arb-spec-tests/fixtures/_wat");

    let mut touched: Vec<PathBuf> = Vec::new();
    let mut errors: Vec<(PathBuf, String)> = Vec::new();

    for entry in WalkDir::new(&fixtures_root).into_iter().filter_map(Result::ok) {
        if !entry.file_type().is_file() {
            continue;
        }
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) != Some("json") {
            continue;
        }
        match rewrite_fixture(path, &wat_root) {
            Ok(()) => touched.push(path.to_path_buf()),
            Err(e) => errors.push((path.to_path_buf(), e.to_string())),
        }
    }

    println!("touched Stylus fixtures ({}):", touched.len());
    for p in &touched {
        println!("  {}", p.display());
    }
    if !errors.is_empty() {
        eprintln!("rewrite errors ({}):", errors.len());
        for (p, e) in &errors {
            eprintln!("  {} — {e}", p.display());
        }
        bail!("fixture rewrite failed for some files");
    }

    let mut bad: Vec<String> = Vec::new();
    for p in &touched {
        if let Err(e) = arb_spec_tests::ExecutionFixture::load(p) {
            bad.push(format!("{}: ExecutionFixture::load: {e}", p.display()));
            continue;
        }
        if let Err(e) = verify_fixture(p) {
            bad.push(format!("{}: {e}", p.display()));
        }
    }
    if !bad.is_empty() {
        eprintln!("verify failures ({}):", bad.len());
        for b in &bad {
            eprintln!("  {b}");
        }
        bail!("fixture round-trip verification failed");
    }
    println!("all Stylus fixtures parse cleanly via parse_incoming_l1_message");
    Ok(())
}

fn rewrite_fixture(path: &Path, wat_root: &Path) -> Result<()> {
    let body = std::fs::read_to_string(path)
        .with_context(|| format!("read {}", path.display()))?;
    let mut value: serde_json::Value = serde_json::from_str(&body)
        .with_context(|| format!("parse {}", path.display()))?;

    let fixture_name = value
        .get("name")
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow!("fixture missing name"))?
        .to_string();

    let scenario = scenarios::for_fixture(&fixture_name)
        .ok_or_else(|| anyhow!("no scenario for fixture: {fixture_name}"))?;

    let (orig_block, orig_ts) = first_message_timing(&value).unwrap_or((0, 1_700_000_000));
    let new_messages = build_messages(&scenario, wat_root, orig_block, orig_ts)?;

    let messages = value
        .get_mut("messages")
        .ok_or_else(|| anyhow!("fixture missing messages array"))?;
    *messages = serde_json::Value::Array(new_messages);

    let pretty = serde_json::to_string_pretty(&value)?;
    std::fs::write(path, pretty + "\n")
        .with_context(|| format!("write {}", path.display()))?;
    Ok(())
}

fn first_message_timing(value: &serde_json::Value) -> Option<(u64, u64)> {
    let m = value
        .get("messages")?
        .as_array()?
        .first()?
        .get("message")?
        .get("header")?;
    let block = m.get("blockNumber")?.as_u64()?;
    let ts = m.get("timestamp")?.as_u64()?;
    Some((block, ts))
}

struct FixtureState {
    next_nonce: u64,
    deployed: BTreeMap<String, Address>,
    request_seq: u64,
    block: u64,
    ts: u64,
}

impl FixtureState {
    fn new(orig_block: u64, orig_ts: u64) -> Self {
        Self {
            next_nonce: 0,
            deployed: BTreeMap::new(),
            request_seq: 1,
            block: orig_block,
            ts: orig_ts,
        }
    }

    fn next_request_seq(&mut self) -> u64 {
        let v = self.request_seq;
        self.request_seq += 1;
        v
    }

    fn record_deployment(&mut self, role: String, sender: Address) -> Address {
        let addr = create_address(sender, self.next_nonce);
        self.deployed.insert(role, addr);
        self.next_nonce += 1;
        addr
    }

    fn bump_nonce(&mut self) {
        self.next_nonce += 1;
    }

    fn step_block(&mut self) {
        self.block += 1;
        self.ts += 1;
    }
}

fn build_messages(
    scenario: &Scenario,
    wat_root: &Path,
    orig_block: u64,
    orig_ts: u64,
) -> Result<Vec<serde_json::Value>> {
    let dev = dev_address();
    let signing_key = dev_signing_key();

    let mut state = FixtureState::new(orig_block, orig_ts);
    let mut wasm_cache: BTreeMap<String, Vec<u8>> = BTreeMap::new();
    let mut out: Vec<serde_json::Value> = Vec::new();

    // First message: deposit funding the dev address.
    let request_seq = state.next_request_seq();
    let deposit = DepositBuilder {
        from: bridge_aliased_sender(),
        to: dev,
        amount: U256::from(DEFAULT_DEPOSIT_AMOUNT),
        l1_block_number: state.block,
        timestamp: state.ts,
        request_seq,
        base_fee_l1: BASE_FEE_L1,
    }
    .build()
    .map_err(|e| anyhow!("build deposit: {e}"))?;
    out.push(wrap_msg(&deposit, 1));

    state.step_block();

    let use_eip1559 = scenario.eip1559;

    for op in &scenario.ops {
        let msg = match op {
            Operation::DeployStylus { role, source } => {
                let wasm = compile_wasm(source, wat_root, &mut wasm_cache)?;
                let stylus_payload = build_stylus_bytecode(&wasm)?;
                let init_code = build_evm_init_code(&stylus_payload);
                state.record_deployment(role.clone(), dev);
                build_signed_create(
                    signing_key,
                    use_eip1559,
                    state.next_nonce - 1,
                    Bytes::from(init_code),
                    DEFAULT_GAS_LIMIT_DEPLOY,
                    state.block,
                    state.ts,
                )?
            }
            Operation::DeployEvm { role, runtime_code } => {
                let init_code = build_evm_init_code(runtime_code);
                state.record_deployment(role.clone(), dev);
                build_signed_create(
                    signing_key,
                    use_eip1559,
                    state.next_nonce - 1,
                    Bytes::from(init_code),
                    DEFAULT_GAS_LIMIT_DEPLOY,
                    state.block,
                    state.ts,
                )?
            }
            Operation::Activate { target_role } => {
                let target = state.deployed.get(target_role).copied().ok_or_else(|| {
                    anyhow!("activate before deploy of {target_role}")
                })?;
                let mut calldata = Vec::with_capacity(36);
                calldata.extend_from_slice(&keccak4("activateProgram(address)"));
                calldata.extend_from_slice(&abi_addr32(target));
                let nonce = state.next_nonce;
                state.bump_nonce();
                build_signed_call(
                    signing_key,
                    use_eip1559,
                    nonce,
                    ARB_WASM_ADDR,
                    U256::from(1u64),
                    Bytes::from(calldata),
                    DEFAULT_GAS_LIMIT_INVOKE,
                    state.block,
                    state.ts,
                )?
            }
            Operation::Invoke {
                target_role,
                calldata,
                value,
            } => {
                let target = state.deployed.get(target_role).copied().ok_or_else(|| {
                    anyhow!("invoke before deploy of {target_role}")
                })?;
                let nonce = state.next_nonce;
                state.bump_nonce();
                build_signed_call(
                    signing_key,
                    use_eip1559,
                    nonce,
                    target,
                    *value,
                    Bytes::from(calldata.clone()),
                    DEFAULT_GAS_LIMIT_INVOKE,
                    state.block,
                    state.ts,
                )?
            }
            Operation::ArbOwnerSetMaxStylusFragments { max_fragments } => {
                let mut calldata = Vec::with_capacity(36);
                calldata
                    .extend_from_slice(&keccak4("setMaxStylusContractFragments(uint8)"));
                let mut word = [0u8; 32];
                word[31] = *max_fragments;
                calldata.extend_from_slice(&word);
                let nonce = state.next_nonce;
                state.bump_nonce();
                build_signed_call(
                    signing_key,
                    use_eip1559,
                    nonce,
                    ARB_OWNER_ADDR,
                    U256::ZERO,
                    Bytes::from(calldata),
                    DEFAULT_GAS_LIMIT_INVOKE,
                    state.block,
                    state.ts,
                )?
            }
        };
        out.push(wrap_msg(&msg, 1));
        state.step_block();
    }

    Ok(out)
}

fn keccak4(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn abi_addr32(a: Address) -> [u8; 32] {
    let mut out = [0u8; 32];
    out[12..].copy_from_slice(a.as_slice());
    out
}

#[allow(clippy::too_many_arguments)]
fn build_signed_call(
    signing_key: B256,
    use_eip1559: bool,
    nonce: u64,
    to: Address,
    value: U256,
    data: Bytes,
    gas_limit: u64,
    block: u64,
    ts: u64,
) -> Result<arb_test_harness::messaging::L1Message> {
    let kind = if use_eip1559 {
        L2TxKind::Eip1559
    } else {
        L2TxKind::Legacy
    };
    SignedL2TxBuilder {
        chain_id: CHAIN_ID,
        nonce,
        to: Some(to),
        value,
        data,
        gas_limit,
        gas_price: DEFAULT_MAX_FEE,
        max_fee_per_gas: DEFAULT_MAX_FEE,
        max_priority_fee_per_gas: DEFAULT_MAX_PRIORITY,
        access_list: Vec::new(),
        kind,
        signing_key,
        l1_block_number: block,
        timestamp: ts,
        request_id: None,
        sender: SEQUENCER_HEADER_SENDER,
        base_fee_l1: BASE_FEE_L1,
    }
    .build()
    .map_err(|e| anyhow!("build signed l2 tx: {e}"))
}

fn build_signed_create(
    signing_key: B256,
    use_eip1559: bool,
    nonce: u64,
    init_code: Bytes,
    gas_limit: u64,
    block: u64,
    ts: u64,
) -> Result<arb_test_harness::messaging::L1Message> {
    let kind = if use_eip1559 {
        L2TxKind::Eip1559
    } else {
        L2TxKind::Legacy
    };
    SignedL2TxBuilder {
        chain_id: CHAIN_ID,
        nonce,
        to: None,
        value: U256::ZERO,
        data: init_code,
        gas_limit,
        gas_price: DEFAULT_MAX_FEE,
        max_fee_per_gas: DEFAULT_MAX_FEE,
        max_priority_fee_per_gas: DEFAULT_MAX_PRIORITY,
        access_list: Vec::new(),
        kind,
        signing_key,
        l1_block_number: block,
        timestamp: ts,
        request_id: None,
        sender: SEQUENCER_HEADER_SENDER,
        base_fee_l1: BASE_FEE_L1,
    }
    .build()
    .map_err(|e| anyhow!("build signed create tx: {e}"))
}

fn compile_wasm(
    source: &scenarios::WasmSource,
    wat_root: &Path,
    cache: &mut BTreeMap<String, Vec<u8>>,
) -> Result<Vec<u8>> {
    let key = source.cache_key();
    if let Some(bytes) = cache.get(&key) {
        return Ok(bytes.clone());
    }
    let wat_text = match source {
        scenarios::WasmSource::WatFile { stem } => {
            let path = wat_root.join(format!("{stem}.wat"));
            std::fs::read_to_string(&path)
                .with_context(|| format!("read {}", path.display()))?
        }
        scenarios::WasmSource::Inline { wat } => wat.clone(),
        scenarios::WasmSource::Padded {
            stem,
            target_size,
        } => {
            let path = wat_root.join(format!("{stem}.wat"));
            let mut wat_text = std::fs::read_to_string(&path)
                .with_context(|| format!("read {}", path.display()))?;
            let extra = wat_sources::oversize_data_segment(*target_size);
            let last_paren = wat_text
                .rfind(')')
                .ok_or_else(|| anyhow!("no closing paren in WAT for {stem}"))?;
            wat_text.insert_str(last_paren, &extra);
            wat_text
        }
    };
    let wasm = wat::parse_str(&wat_text).map_err(|e| anyhow!("compile wat: {e}"))?;
    cache.insert(key, wasm.clone());
    Ok(wasm)
}

/// Compress `wasm` with brotli (no dictionary) and prepend the 4-byte
/// Stylus discriminant `[0xEF, 0xF0, 0x00, 0x00]`.
fn build_stylus_bytecode(wasm: &[u8]) -> Result<Vec<u8>> {
    let compressed = brotli_compress(wasm)?;
    let mut out = Vec::with_capacity(4 + compressed.len());
    out.extend_from_slice(&STYLUS_DISCRIMINANT);
    out.push(STYLUS_DICT_BYTE);
    out.extend_from_slice(&compressed);
    Ok(out)
}

fn brotli_compress(input: &[u8]) -> Result<Vec<u8>> {
    use std::io::Cursor;
    let mut output: Vec<u8> = Vec::new();
    let params = brotli::enc::BrotliEncoderParams::default();
    let mut reader = Cursor::new(input);
    brotli::BrotliCompress(&mut reader, &mut output, &params)
        .map_err(|e| anyhow!("brotli compress: {e}"))?;
    Ok(output)
}

/// Build EVM constructor bytecode that returns `code` as the deployed
/// contract bytecode. Header is 14 bytes (PUSH2 size; PUSH1 0x0e; PUSH1 0;
/// CODECOPY; PUSH2 size; PUSH1 0; RETURN), then the payload appended.
fn build_evm_init_code(code: &[u8]) -> Vec<u8> {
    let size = code.len();
    assert!(size <= u16::MAX as usize, "payload too large for PUSH2");
    let size_hi = (size >> 8) as u8;
    let size_lo = (size & 0xff) as u8;
    let mut buf = Vec::with_capacity(14 + size);
    buf.extend_from_slice(&[0x61, size_hi, size_lo]);
    buf.extend_from_slice(&[0x60, 0x0e]);
    buf.extend_from_slice(&[0x60, 0x00]);
    buf.push(0x39);
    buf.extend_from_slice(&[0x61, size_hi, size_lo]);
    buf.extend_from_slice(&[0x60, 0x00]);
    buf.push(0xf3);
    debug_assert_eq!(buf.len(), 14);
    buf.extend_from_slice(code);
    buf
}

/// keccak256(rlp([sender, nonce]))[12:]
fn create_address(sender: Address, nonce: u64) -> Address {
    let mut rlp = Vec::with_capacity(32);
    let payload_len = sender_rlp_len() + nonce_rlp_len(nonce);
    encode_list_header(&mut rlp, payload_len);
    encode_address_rlp(&mut rlp, sender);
    encode_nonce_rlp(&mut rlp, nonce);
    let hash = keccak256(&rlp);
    Address::from_slice(&hash.0[12..])
}

fn sender_rlp_len() -> usize {
    21
}

fn nonce_rlp_len(nonce: u64) -> usize {
    if nonce == 0 {
        1
    } else {
        let buf = nonce.to_be_bytes();
        let trimmed = trim_leading_zeros(&buf);
        if trimmed.len() == 1 && trimmed[0] < 0x80 {
            1
        } else {
            1 + trimmed.len()
        }
    }
}

fn encode_list_header(out: &mut Vec<u8>, payload_len: usize) {
    if payload_len < 56 {
        out.push(0xc0 + payload_len as u8);
    } else {
        let len_bytes = payload_len.to_be_bytes();
        let trimmed = trim_leading_zeros(&len_bytes);
        out.push(0xf7 + trimmed.len() as u8);
        out.extend_from_slice(trimmed);
    }
}

fn encode_address_rlp(out: &mut Vec<u8>, addr: Address) {
    out.push(0x94);
    out.extend_from_slice(addr.as_slice());
}

fn encode_nonce_rlp(out: &mut Vec<u8>, nonce: u64) {
    if nonce == 0 {
        out.push(0x80);
        return;
    }
    let buf = nonce.to_be_bytes();
    let trimmed = trim_leading_zeros(&buf);
    if trimmed.len() == 1 && trimmed[0] < 0x80 {
        out.push(trimmed[0]);
    } else {
        out.push(0x80 + trimmed.len() as u8);
        out.extend_from_slice(trimmed);
    }
}

fn trim_leading_zeros(buf: &[u8]) -> &[u8] {
    let mut start = 0;
    while start < buf.len() - 1 && buf[start] == 0 {
        start += 1;
    }
    &buf[start..]
}

fn wrap_msg(
    msg: &arb_test_harness::messaging::L1Message,
    delayed_messages_read: u64,
) -> serde_json::Value {
    serde_json::json!({
        "msgIdx": serde_json::Value::Null,
        "message": msg,
        "delayedMessagesRead": delayed_messages_read,
    })
}

fn verify_fixture(path: &Path) -> Result<()> {
    let body = std::fs::read_to_string(path)?;
    let value: serde_json::Value = serde_json::from_str(&body)?;
    let messages = value
        .get("messages")
        .and_then(|v| v.as_array())
        .ok_or_else(|| anyhow!("missing messages"))?;
    for (i, msg) in messages.iter().enumerate() {
        common::verify_l1_message(msg, i)?;
    }
    Ok(())
}

fn locate_workspace_root() -> Result<PathBuf> {
    let mut dir = std::env::current_dir()?;
    loop {
        let candidate = dir.join("Cargo.toml");
        if candidate.is_file() {
            let body = std::fs::read_to_string(&candidate)?;
            if body.contains("[workspace]") {
                return Ok(dir);
            }
        }
        if !dir.pop() {
            bail!("could not find workspace root");
        }
    }
}
