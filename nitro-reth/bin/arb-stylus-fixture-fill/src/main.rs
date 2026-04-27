//! Replace `_TODO_*` placeholders in stylus fixture JSON files with real,
//! base64-encoded L2 messages that arbreth's parser can decode.
//!
//! For every JSON file under `crates/arb-spec-tests/fixtures/stylus/**`, each
//! message whose `l2Msg` starts with `_TODO_` is rewritten to a `ContractTx`
//! (kind=3 sub-kind=1) whose body deploys a Stylus program (WAT compiled via
//! `wat::parse_str` and prefixed with the Stylus discriminant) or invokes a
//! previously deployed program.

use std::collections::BTreeMap;
use std::path::{Path, PathBuf};

use alloy_primitives::{address, keccak256, Address, Bytes, U256};
use alloy_sol_types::{sol, SolCall};
use anyhow::{anyhow, bail, Context, Result};
use arb_test_harness::messaging::{ContractTxBuilder, MessageBuilder};
use base64::Engine;
use walkdir::WalkDir;

mod placeholders;
mod wat_sources;

use placeholders::ResolvedPlaceholder;

const FUNDED_FROM: Address = address!("cd5fe7820f6d69ad4fde67de05b4791afd1bd27c");
const ARB_WASM_ADDR: Address = address!("0000000000000000000000000000000000000071");
const ARB_OWNER_ADDR: Address = address!("0000000000000000000000000000000000000070");

const STYLUS_DISCRIMINANT: [u8; 3] = [0xEF, 0xF0, 0x00];

const DEFAULT_GAS_LIMIT_DEPLOY: u64 = 50_000_000;
const DEFAULT_GAS_LIMIT_INVOKE: u64 = 30_000_000;
const DEFAULT_MAX_FEE_PER_GAS: u64 = 1_000_000_000;
const BASE_FEE_L1: u64 = 0;

sol! {
    interface IArbWasm {
        function activateProgram(address program) external payable returns (uint16, uint256);
    }

    interface IArbOwner {
        function setMaxStylusContractFragments(uint8 maxFragments) external;
    }
}

fn main() -> Result<()> {
    let workspace_root = locate_workspace_root()?;
    let fixtures_root = workspace_root.join("crates/arb-spec-tests/fixtures/stylus");
    if !fixtures_root.is_dir() {
        bail!("fixtures dir not found at {}", fixtures_root.display());
    }
    let wat_root = workspace_root.join("crates/arb-spec-tests/fixtures/_wat");

    let mut touched = Vec::new();
    let mut unresolved: Vec<(PathBuf, String)> = Vec::new();

    for entry in WalkDir::new(&fixtures_root).into_iter().filter_map(Result::ok) {
        if !entry.file_type().is_file() {
            continue;
        }
        let path = entry.path();
        if path.extension().and_then(|e| e.to_str()) != Some("json") {
            continue;
        }
        let body = std::fs::read_to_string(path)
            .with_context(|| format!("read {}", path.display()))?;
        if !body.contains("_TODO_") {
            continue;
        }
        let mut value: serde_json::Value = serde_json::from_str(&body)
            .with_context(|| format!("parse {}", path.display()))?;

        let fixture_name = value
            .get("name")
            .and_then(|v| v.as_str())
            .unwrap_or("<unnamed>")
            .to_string();

        match rewrite_fixture(&mut value, &wat_root, &fixture_name) {
            Ok(()) => {
                let pretty = serde_json::to_string_pretty(&value)?;
                std::fs::write(path, pretty + "\n")
                    .with_context(|| format!("write {}", path.display()))?;
                touched.push(path.to_path_buf());
            }
            Err(e) => {
                unresolved.push((path.to_path_buf(), e.to_string()));
            }
        }
    }

    println!("touched fixtures ({}):", touched.len());
    for p in &touched {
        println!("  {}", p.display());
    }
    if !unresolved.is_empty() {
        println!("unresolved ({}):", unresolved.len());
        for (p, e) in &unresolved {
            println!("  {} — {e}", p.display());
        }
    }

    // Round-trip verification: every l2Msg in every fixture must parse via
    // arbos::parse_incoming_l1_message, and the file must deserialize cleanly
    // as an `ExecutionFixture`.
    let mut bad: Vec<String> = Vec::new();
    for entry in WalkDir::new(&fixtures_root).into_iter().filter_map(Result::ok) {
        if entry.file_type().is_file()
            && entry.path().extension().and_then(|e| e.to_str()) == Some("json")
        {
            let path = entry.path();
            if let Err(e) = arb_spec_tests::ExecutionFixture::load(path) {
                bad.push(format!("{}: ExecutionFixture::load: {e}", path.display()));
                continue;
            }
            if let Err(e) = verify_fixture(path) {
                bad.push(format!("{}: {e}", path.display()));
            }
        }
    }
    if !bad.is_empty() {
        eprintln!("verify failures ({}):", bad.len());
        for b in &bad {
            eprintln!("  {b}");
        }
        bail!("fixture round-trip verification failed");
    }
    println!("all fixture l2Msg values round-trip cleanly");
    Ok(())
}

fn verify_fixture(path: &Path) -> Result<()> {
    let body = std::fs::read_to_string(path)?;
    let value: serde_json::Value = serde_json::from_str(&body)?;
    let messages = value
        .get("messages")
        .and_then(|v| v.as_array())
        .ok_or_else(|| anyhow!("missing messages"))?;
    for (i, msg) in messages.iter().enumerate() {
        let inner = msg
            .get("message")
            .ok_or_else(|| anyhow!("msg {i}: missing message"))?;
        let header = inner
            .get("header")
            .ok_or_else(|| anyhow!("msg {i}: missing header"))?;
        let kind = header
            .get("kind")
            .and_then(|v| v.as_u64())
            .ok_or_else(|| anyhow!("msg {i}: missing kind"))? as u8;
        let sender_str = header
            .get("sender")
            .and_then(|v| v.as_str())
            .ok_or_else(|| anyhow!("msg {i}: missing sender"))?;
        let sender = parse_address(sender_str)
            .ok_or_else(|| anyhow!("msg {i}: bad sender {sender_str}"))?;
        let block_number = header
            .get("blockNumber")
            .and_then(|v| v.as_u64())
            .unwrap_or(0);
        let timestamp = header
            .get("timestamp")
            .and_then(|v| v.as_u64())
            .unwrap_or(0);
        let request_id = header
            .get("requestId")
            .and_then(|v| v.as_str())
            .map(|s| {
                let raw = s.trim_start_matches("0x");
                let mut buf = [0u8; 32];
                hex_into(raw, &mut buf).unwrap_or(());
                buf
            });
        let base_fee_l1 = header
            .get("baseFeeL1")
            .and_then(|v| v.as_u64())
            .unwrap_or(0);
        let l2_msg_b64 = inner
            .get("l2Msg")
            .and_then(|v| v.as_str())
            .ok_or_else(|| anyhow!("msg {i}: missing l2Msg"))?;
        let l2_msg = base64::engine::general_purpose::STANDARD
            .decode(l2_msg_b64.as_bytes())
            .map_err(|e| anyhow!("msg {i}: bad base64: {e}"))?;

        // Reconstruct the L1 incoming wire format exactly as arbos expects.
        let mut wire = Vec::with_capacity(1 + 32 + 8 + 8 + 32 + 32 + l2_msg.len());
        wire.push(kind);
        let mut padded = [0u8; 32];
        padded[12..].copy_from_slice(sender.as_slice());
        wire.extend_from_slice(&padded);
        wire.extend_from_slice(&block_number.to_be_bytes());
        wire.extend_from_slice(&timestamp.to_be_bytes());
        wire.extend_from_slice(&request_id.unwrap_or([0u8; 32]));
        let mut fee_buf = [0u8; 32];
        fee_buf[24..].copy_from_slice(&base_fee_l1.to_be_bytes());
        wire.extend_from_slice(&fee_buf);
        wire.extend_from_slice(&l2_msg);

        let parsed = arbos::arbos_types::parse_incoming_l1_message(&wire)
            .map_err(|e| anyhow!("msg {i}: parse_incoming_l1_message: {e}"))?;
        // Drive parse_l2_transactions on the inner body for kind=3 messages —
        // that's the path that actually decodes ContractTx / Batch / etc.
        if kind == 3 {
            arbos::parse_l2::parse_l2_transactions(
                parsed.header.kind,
                parsed.header.poster,
                &parsed.l2_msg,
                parsed.header.request_id,
                parsed.header.l1_base_fee,
                421_614,
            )
            .map_err(|e| anyhow!("msg {i}: parse_l2_transactions: {e}"))?;
        }
    }
    Ok(())
}

fn parse_address(s: &str) -> Option<Address> {
    let raw = s.trim_start_matches("0x");
    if raw.len() != 40 {
        return None;
    }
    let mut buf = [0u8; 20];
    hex_into(raw, &mut buf).ok()?;
    Some(Address::from(buf))
}

fn hex_into(s: &str, out: &mut [u8]) -> Result<()> {
    if s.len() != out.len() * 2 {
        bail!("hex length mismatch: {} vs {}", s.len(), out.len() * 2);
    }
    for (i, b) in out.iter_mut().enumerate() {
        let hi =
            u8::from_str_radix(&s[i * 2..i * 2 + 1], 16).map_err(|e| anyhow!("hex hi: {e}"))?;
        let lo = u8::from_str_radix(&s[i * 2 + 1..i * 2 + 2], 16)
            .map_err(|e| anyhow!("hex lo: {e}"))?;
        *b = (hi << 4) | lo;
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

/// State for a single fixture's rewrite: tracks deploy nonce & deployed addresses
/// so subsequent messages can reference them.
struct FixtureState {
    next_nonce: u64,
    /// Map of program role (e.g. `program`, `multicall`, `program_a`) → address.
    deployed: BTreeMap<String, Address>,
    /// Per-message timestamp from the original fixture, for stable request_seq.
    request_seq_counter: u64,
}

impl FixtureState {
    fn new() -> Self {
        Self {
            next_nonce: 0,
            deployed: BTreeMap::new(),
            request_seq_counter: 0x1000,
        }
    }

    /// Compute and remember the address that the next CREATE from FUNDED_FROM
    /// will deploy to, using the current account nonce.
    fn record_deployment(&mut self, role: String) -> Address {
        let addr = create_address(FUNDED_FROM, self.next_nonce);
        self.deployed.insert(role, addr);
        self.next_nonce += 1;
        addr
    }

    fn bump_nonce(&mut self) {
        self.next_nonce += 1;
    }

    fn next_request_seq(&mut self) -> u64 {
        let v = self.request_seq_counter;
        self.request_seq_counter += 1;
        v
    }
}

fn rewrite_fixture(
    value: &mut serde_json::Value,
    wat_root: &Path,
    fixture_name: &str,
) -> Result<()> {
    let messages = value
        .get_mut("messages")
        .and_then(|v| v.as_array_mut())
        .ok_or_else(|| anyhow!("fixture missing messages array"))?;

    let mut state = FixtureState::new();

    // Cache compiled WASM bodies for repeated deploys (e.g. multicall used twice).
    let mut wasm_cache: BTreeMap<String, Vec<u8>> = BTreeMap::new();

    for msg in messages.iter_mut() {
        let l2_msg = msg
            .get("message")
            .and_then(|m| m.get("l2Msg"))
            .and_then(|s| s.as_str())
            .unwrap_or("");
        if !l2_msg.starts_with("_TODO_") {
            continue;
        }
        let placeholder = l2_msg.to_string();
        let header_ts = msg
            .get("message")
            .and_then(|m| m.get("header"))
            .and_then(|h| h.get("timestamp"))
            .and_then(|t| t.as_u64())
            .unwrap_or(0);
        let header_block = msg
            .get("message")
            .and_then(|m| m.get("header"))
            .and_then(|h| h.get("blockNumber"))
            .and_then(|t| t.as_u64())
            .unwrap_or(0);

        let resolved = placeholders::resolve(&placeholder, fixture_name)
            .with_context(|| format!("resolve {placeholder}"))?;

        let new_msg_body = match resolved {
            ResolvedPlaceholder::Deploy { role, source } => {
                let wasm = wasm_for_source(&source, wat_root, &mut wasm_cache, fixture_name)?;
                let stylus_bytecode = stylus_bytecode(&wasm);
                let init_code = build_deploy_init_code(&stylus_bytecode);
                let _addr = state.record_deployment(role.clone());
                let request_seq = state.next_request_seq();
                let builder = ContractTxBuilder {
                    from: FUNDED_FROM,
                    gas_limit: DEFAULT_GAS_LIMIT_DEPLOY,
                    max_fee_per_gas: U256::from(DEFAULT_MAX_FEE_PER_GAS),
                    to: Address::ZERO,
                    value: U256::ZERO,
                    data: Bytes::from(init_code),
                    l1_block_number: header_block,
                    timestamp: header_ts,
                    request_seq,
                    base_fee_l1: BASE_FEE_L1,
                };
                builder
                    .build()
                    .map_err(|e| anyhow!("build deploy: {e}"))?
                    .l2_msg
            }
            ResolvedPlaceholder::DeployEvm { role, runtime_code } => {
                let init_code = build_deploy_init_code(&runtime_code);
                let _addr = state.record_deployment(role.clone());
                let request_seq = state.next_request_seq();
                let builder = ContractTxBuilder {
                    from: FUNDED_FROM,
                    gas_limit: DEFAULT_GAS_LIMIT_DEPLOY,
                    max_fee_per_gas: U256::from(DEFAULT_MAX_FEE_PER_GAS),
                    to: Address::ZERO,
                    value: U256::ZERO,
                    data: Bytes::from(init_code),
                    l1_block_number: header_block,
                    timestamp: header_ts,
                    request_seq,
                    base_fee_l1: BASE_FEE_L1,
                };
                builder
                    .build()
                    .map_err(|e| anyhow!("build deploy_evm: {e}"))?
                    .l2_msg
            }
            ResolvedPlaceholder::ActivateProgram { target_role } => {
                let target = state
                    .deployed
                    .get(&target_role)
                    .copied()
                    .ok_or_else(|| anyhow!("activate before deploy of {target_role}"))?;
                let calldata = IArbWasm::activateProgramCall { program: target }.abi_encode();
                state.bump_nonce();
                let request_seq = state.next_request_seq();
                let builder = ContractTxBuilder {
                    from: FUNDED_FROM,
                    gas_limit: DEFAULT_GAS_LIMIT_INVOKE,
                    max_fee_per_gas: U256::from(DEFAULT_MAX_FEE_PER_GAS),
                    to: ARB_WASM_ADDR,
                    // Activation requires a payable data fee; pass 1 wei placeholder.
                    value: U256::from(1u64),
                    data: Bytes::from(calldata),
                    l1_block_number: header_block,
                    timestamp: header_ts,
                    request_seq,
                    base_fee_l1: BASE_FEE_L1,
                };
                builder
                    .build()
                    .map_err(|e| anyhow!("build activate: {e}"))?
                    .l2_msg
            }
            ResolvedPlaceholder::Invoke {
                target_role,
                calldata,
                value,
            } => {
                let target = state
                    .deployed
                    .get(&target_role)
                    .copied()
                    .ok_or_else(|| anyhow!("invoke before deploy of {target_role}"))?;
                state.bump_nonce();
                let request_seq = state.next_request_seq();
                let builder = ContractTxBuilder {
                    from: FUNDED_FROM,
                    gas_limit: DEFAULT_GAS_LIMIT_INVOKE,
                    max_fee_per_gas: U256::from(DEFAULT_MAX_FEE_PER_GAS),
                    to: target,
                    value,
                    data: Bytes::from(calldata),
                    l1_block_number: header_block,
                    timestamp: header_ts,
                    request_seq,
                    base_fee_l1: BASE_FEE_L1,
                };
                builder
                    .build()
                    .map_err(|e| anyhow!("build invoke: {e}"))?
                    .l2_msg
            }
            ResolvedPlaceholder::ArbOwnerSetMaxFragments { max_fragments } => {
                let calldata = IArbOwner::setMaxStylusContractFragmentsCall {
                    maxFragments: max_fragments,
                }
                .abi_encode();
                state.bump_nonce();
                let request_seq = state.next_request_seq();
                let builder = ContractTxBuilder {
                    from: FUNDED_FROM,
                    gas_limit: DEFAULT_GAS_LIMIT_INVOKE,
                    max_fee_per_gas: U256::from(DEFAULT_MAX_FEE_PER_GAS),
                    to: ARB_OWNER_ADDR,
                    value: U256::ZERO,
                    data: Bytes::from(calldata),
                    l1_block_number: header_block,
                    timestamp: header_ts,
                    request_seq,
                    base_fee_l1: BASE_FEE_L1,
                };
                builder
                    .build()
                    .map_err(|e| anyhow!("build arbowner: {e}"))?
                    .l2_msg
            }
            ResolvedPlaceholder::Batch { invokes } => {
                // L2 batch: outer kind=3 sub-kind=3, then each segment is a
                // bytestring whose body is itself an L2 message (sub-kind=1
                // ContractTx). We encode the batch payload manually here.
                let mut batch_payload: Vec<u8> = Vec::new();
                for inv in invokes {
                    let target = state.deployed.get(&inv.target_role).copied().ok_or_else(
                        || anyhow!("batch invoke before deploy of {}", inv.target_role),
                    )?;
                    state.bump_nonce();
                    // Inner segment: kind byte 1 (CONTRACT_TX) + 4 32-byte
                    // fields + calldata.
                    let mut segment = Vec::with_capacity(1 + 32 * 4 + inv.calldata.len());
                    segment.push(0x01); // L2_MESSAGE_KIND_CONTRACT_TX
                    segment.extend_from_slice(
                        &U256::from(DEFAULT_GAS_LIMIT_INVOKE).to_be_bytes::<32>(),
                    );
                    segment.extend_from_slice(
                        &U256::from(DEFAULT_MAX_FEE_PER_GAS).to_be_bytes::<32>(),
                    );
                    let mut to32 = [0u8; 32];
                    to32[12..].copy_from_slice(target.as_slice());
                    segment.extend_from_slice(&to32);
                    segment.extend_from_slice(&inv.value.to_be_bytes::<32>());
                    segment.extend_from_slice(&inv.calldata);

                    // Length-prefix the segment per the batch wire format.
                    let len = segment.len() as u64;
                    batch_payload.extend_from_slice(&len.to_be_bytes());
                    batch_payload.extend_from_slice(&segment);
                }
                let mut body = Vec::with_capacity(1 + batch_payload.len());
                body.push(0x03); // L2_MESSAGE_KIND_BATCH
                body.extend_from_slice(&batch_payload);
                let l2_msg = base64::engine::general_purpose::STANDARD.encode(&body);
                let request_seq = state.next_request_seq();
                // Override the message header so the new request_id is unique.
                if let Some(hdr) = msg
                    .get_mut("message")
                    .and_then(|m| m.get_mut("header"))
                {
                    let req_id = format!("0x{:064x}", request_seq);
                    hdr["requestId"] = serde_json::Value::String(req_id);
                }
                l2_msg
            }
        };

        if let Some(m) = msg.get_mut("message").and_then(|m| m.as_object_mut()) {
            m.insert(
                "l2Msg".to_string(),
                serde_json::Value::String(new_msg_body),
            );
            // Set sender → funded address so balance check passes for ContractTx.
            if let Some(hdr) = m.get_mut("header").and_then(|h| h.as_object_mut()) {
                hdr.insert(
                    "sender".to_string(),
                    serde_json::Value::String(format!("{FUNDED_FROM:#x}")),
                );
                // Always populate requestId so ContractTx parses cleanly.
                let need_req = hdr
                    .get("requestId")
                    .map(|v| v.is_null())
                    .unwrap_or(true);
                if need_req {
                    let request_seq = state.request_seq_counter; // already bumped
                    hdr.insert(
                        "requestId".to_string(),
                        serde_json::Value::String(format!("0x{:064x}", request_seq.saturating_sub(1))),
                    );
                }
            }
        }
    }

    Ok(())
}

fn wasm_for_source(
    source: &placeholders::WasmSource,
    wat_root: &Path,
    cache: &mut BTreeMap<String, Vec<u8>>,
    fixture_name: &str,
) -> Result<Vec<u8>> {
    let key = source.cache_key();
    if let Some(bytes) = cache.get(&key) {
        return Ok(bytes.clone());
    }
    let wat_text = match source {
        placeholders::WasmSource::WatFile { stem } => {
            let path = wat_root.join(format!("{stem}.wat"));
            std::fs::read_to_string(&path)
                .with_context(|| format!("read {}", path.display()))?
        }
        placeholders::WasmSource::Inline { wat } => wat.clone(),
        placeholders::WasmSource::Padded { stem, target_size } => {
            let path = wat_root.join(format!("{stem}.wat"));
            let mut wat_text = std::fs::read_to_string(&path)
                .with_context(|| format!("read {}", path.display()))?;
            // Append a sizable data section so the compiled WASM exceeds the
            // target size after WAT compilation.
            let extra = wat_sources::oversize_data_segment(*target_size);
            // Inject the extra `(data ...)` declaration just before the
            // module's closing paren.
            let last_paren = wat_text
                .rfind(')')
                .ok_or_else(|| anyhow!("no closing paren in WAT for {stem}"))?;
            wat_text.insert_str(last_paren, &extra);
            wat_text
        }
    };
    let _ = fixture_name;
    let wasm = wat::parse_str(&wat_text)
        .map_err(|e| anyhow!("compile wat: {e}"))?;
    cache.insert(key, wasm.clone());
    Ok(wasm)
}

fn stylus_bytecode(wasm: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(STYLUS_DISCRIMINANT.len() + wasm.len());
    out.extend_from_slice(&STYLUS_DISCRIMINANT);
    out.extend_from_slice(wasm);
    out
}

/// Build EVM constructor bytecode that returns `code` as the deployed
/// contract bytecode. Uses CODECOPY to copy the trailing payload.
fn build_deploy_init_code(code: &[u8]) -> Vec<u8> {
    // Header layout (12 bytes):
    //   PUSH2 size      (3 bytes)
    //   PUSH1 0x0c      (2 bytes) ; offset within init code where payload begins
    //   PUSH1 0x00      (2 bytes) ; mem dest
    //   CODECOPY        (1 byte)
    //   PUSH2 size      (3 bytes)
    //   PUSH1 0x00      (2 bytes)
    //   RETURN          (1 byte)
    //   = 14 bytes total.
    // We pick a 14-byte header.
    let size = code.len();
    if size > u16::MAX as usize {
        panic!("payload too large for PUSH2 size encoding: {size}");
    }
    let size_hi = (size >> 8) as u8;
    let size_lo = (size & 0xff) as u8;
    let mut buf = Vec::with_capacity(14 + code.len());
    // PUSH2 size
    buf.extend_from_slice(&[0x61, size_hi, size_lo]);
    // PUSH1 0x0e (offset = 14 = header length)
    buf.extend_from_slice(&[0x60, 0x0e]);
    // PUSH1 0x00
    buf.extend_from_slice(&[0x60, 0x00]);
    // CODECOPY
    buf.push(0x39);
    // PUSH2 size
    buf.extend_from_slice(&[0x61, size_hi, size_lo]);
    // PUSH1 0x00
    buf.extend_from_slice(&[0x60, 0x00]);
    // RETURN
    buf.push(0xf3);
    debug_assert_eq!(buf.len(), 14);
    buf.extend_from_slice(code);
    buf
}

/// Compute the CREATE address: keccak256(rlp([sender, nonce]))[12:].
fn create_address(sender: Address, nonce: u64) -> Address {
    let mut rlp = Vec::with_capacity(32);
    let list_payload_len =
        sender_rlp_len(sender) + nonce_rlp_len(nonce);
    encode_list_header(&mut rlp, list_payload_len);
    encode_address_rlp(&mut rlp, sender);
    encode_nonce_rlp(&mut rlp, nonce);
    let hash = keccak256(&rlp);
    Address::from_slice(&hash.0[12..])
}

fn sender_rlp_len(_sender: Address) -> usize {
    // 20-byte address: header 0x94 + 20 bytes = 21
    21
}

fn nonce_rlp_len(nonce: u64) -> usize {
    if nonce == 0 {
        1
    } else {
        let mut buf = [0u8; 8];
        buf.copy_from_slice(&nonce.to_be_bytes());
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

