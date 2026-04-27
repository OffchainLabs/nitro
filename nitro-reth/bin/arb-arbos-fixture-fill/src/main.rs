//! Rewrite ArbOS gate fixture messages so each one consists of:
//!
//! 1. A `kind=12` ETH deposit funding the deterministic dev address.
//! 2. (Optional) `kind=3 sub=4` SignedL2Tx envelopes calling the gated
//!    precompile method that the fixture name documents.
//! 3. (For ArbRetryableTx fixtures) a `kind=9` SubmitRetryable followed by
//!    a SignedL2Tx redeem call.
//!
//! After this runs every fixture round-trips cleanly via
//! `arbos::arbos_types::parse_incoming_l1_message` and (for `kind=3` outer)
//! `arbos::parse_l2::parse_l2_transactions`.

mod common;

use std::path::{Path, PathBuf};

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use anyhow::{anyhow, bail, Context, Result};
use arb_test_harness::messaging::{
    DepositBuilder, L2TxKind, MessageBuilder, RetryableSubmitBuilder, SignedL2TxBuilder,
};
use walkdir::WalkDir;

use common::{bridge_aliased_sender, dev_address, dev_signing_key};

const ARB_OWNER_ADDR: Address = address!("0000000000000000000000000000000000000070");
const ARB_RETRYABLE_ADDR: Address = address!("000000000000000000000000000000000000006e");
const ARB_WASM_ADDR: Address = address!("0000000000000000000000000000000000000071");
const ARB_NATIVE_TOKEN_MGR_ADDR: Address = address!("0000000000000000000000000000000000000073");
const P256_VERIFY_ADDR: Address = address!("0000000000000000000000000000000000000100");

const SEQUENCER_HEADER_SENDER: Address =
    address!("a4b000000000000000000073657175656e636572");

const CHAIN_ID: u64 = 421614;
const DEFAULT_GAS_LIMIT: u64 = 500_000;
const DEFAULT_DEPOSIT_AMOUNT: u128 = 1_000_000_000_000_000_000_000u128; // 1000 ETH
const DEFAULT_GAS_PRICE: u128 = 1_000_000_000;
const DEFAULT_MAX_FEE: u128 = 1_000_000_000;
const DEFAULT_MAX_PRIORITY: u128 = 0;
const BASE_FEE_L1: u64 = 0;

fn main() -> Result<()> {
    let workspace_root = locate_workspace_root()?;
    let fixtures_root = workspace_root.join("crates/arb-spec-tests/fixtures/arbos");
    if !fixtures_root.is_dir() {
        bail!("fixtures dir not found at {}", fixtures_root.display());
    }

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
        let file_name = path.file_name().and_then(|s| s.to_str()).unwrap_or("");
        if !file_name.starts_with("gate_") {
            continue;
        }
        match rewrite_fixture(path) {
            Ok(()) => touched.push(path.to_path_buf()),
            Err(e) => errors.push((path.to_path_buf(), e.to_string())),
        }
    }

    println!("touched ArbOS gate fixtures ({}):", touched.len());
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

    // Round-trip verification.
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
    println!("all gate fixtures parse cleanly via parse_incoming_l1_message");
    Ok(())
}

#[derive(Debug, Clone, Copy)]
enum FixturePlan {
    /// Just the deposit (no follow-up tx).
    DepositOnly,
    /// Deposit + a SignedL2Tx call to a target with calldata.
    DepositPlusSignedCall {
        target: Address,
        selector: [u8; 4],
        kind: ArgsKind,
        use_eip1559: bool,
    },
    /// Deposit + SubmitRetryable + Redeem signed tx.
    RetryableSubmitThenRedeem {
        use_eip1559: bool,
    },
    /// Two deposits (gate_build_1089_40 needs two blocks via two deposits).
    TwoDeposits,
    /// Deposit + a generic EOA→EOA transfer (framework gate fixtures).
    DepositPlusEoaTransfer { use_eip1559: bool },
}

#[derive(Debug, Clone, Copy)]
enum ArgsKind {
    None,
    Bool(bool),
    EmptyConstraintArray,
    /// Raw 32-byte zero-padded payload after the selector.
    RawZeroPad32,
}

fn classify(name: &str, arbos_version: u64) -> FixturePlan {
    let use_eip1559 = arbos_version >= 40;

    if name == "gate_build_1089_40" {
        return FixturePlan::TwoDeposits;
    }
    if name.starts_with("gate_arbnativetokenmanager") {
        return FixturePlan::DepositPlusSignedCall {
            target: ARB_NATIVE_TOKEN_MGR_ADDR,
            selector: [0xde, 0xad, 0xbe, 0xef],
            kind: ArgsKind::RawZeroPad32,
            use_eip1559,
        };
    }
    if name == "gate_arbowner_28_60" || name == "gate_arbowner_338_60" {
        return FixturePlan::DepositPlusSignedCall {
            target: ARB_OWNER_ADDR,
            selector: keccak4("setCollectTips(bool)"),
            kind: ArgsKind::Bool(true),
            use_eip1559,
        };
    }
    if name == "gate_arbowner_1318_multi_constraint_fix" {
        return FixturePlan::DepositPlusSignedCall {
            target: ARB_OWNER_ADDR,
            selector: keccak4("setGasPricingConstraints(uint64[3][])"),
            kind: ArgsKind::EmptyConstraintArray,
            use_eip1559,
        };
    }
    if name.starts_with("gate_arbretryabletx_") {
        return FixturePlan::RetryableSubmitThenRedeem { use_eip1559 };
    }
    if name == "gate_arbwasm_98_stylus_charging_fixes" {
        return FixturePlan::DepositPlusSignedCall {
            target: ARB_WASM_ADDR,
            selector: keccak4("minInitGas()"),
            kind: ArgsKind::None,
            use_eip1559,
        };
    }
    if name == "gate_lib_546_30" {
        return FixturePlan::DepositPlusSignedCall {
            target: P256_VERIFY_ADDR,
            selector: [0xde, 0xad, 0xbe, 0xef],
            kind: ArgsKind::RawZeroPad32,
            use_eip1559,
        };
    }
    if name == "gate_block_processor_300_50"
        || name == "gate_build_1531_50"
        || name == "gate_build_1549_50"
        || name == "gate_build_2157_multi_gas_constraints"
        || name == "gate_build_2347_multi_gas_constraints"
        || name == "gate_tx_processor_274_50"
        || name == "gate_tx_processor_327_10"
        || name == "gate_tx_processor_406_11"
        || name == "gate_tx_processor_760_11"
    {
        return FixturePlan::DepositPlusEoaTransfer { use_eip1559 };
    }

    FixturePlan::DepositOnly
}

fn keccak4(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn rewrite_fixture(path: &Path) -> Result<()> {
    let body = std::fs::read_to_string(path)
        .with_context(|| format!("read {}", path.display()))?;
    let mut value: serde_json::Value = serde_json::from_str(&body)
        .with_context(|| format!("parse {}", path.display()))?;

    let fixture_name = value
        .get("name")
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow!("fixture missing name"))?
        .to_string();
    let arbos_version = value
        .get("genesis")
        .and_then(|g| g.get("config"))
        .and_then(|c| c.get("arbitrum"))
        .and_then(|a| a.get("InitialArbOSVersion"))
        .and_then(|v| v.as_u64())
        .unwrap_or(60);

    let (orig_block, orig_ts) = first_message_timing(&value).unwrap_or((0, 1_700_000_000));

    let plan = classify(&fixture_name, arbos_version);
    let new_messages = build_messages(plan, orig_block, orig_ts)?;

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

fn build_messages(
    plan: FixturePlan,
    orig_block: u64,
    orig_ts: u64,
) -> Result<Vec<serde_json::Value>> {
    let dev = dev_address();
    let signing_key = dev_signing_key();

    let mut request_seq = 1u64;
    let next_req = |seq: &mut u64| -> u64 {
        let v = *seq;
        *seq += 1;
        v
    };

    let mut out: Vec<serde_json::Value> = Vec::new();

    let deposit = DepositBuilder {
        from: bridge_aliased_sender(),
        to: dev,
        amount: U256::from(DEFAULT_DEPOSIT_AMOUNT),
        l1_block_number: orig_block,
        timestamp: orig_ts,
        request_seq: next_req(&mut request_seq),
        base_fee_l1: BASE_FEE_L1,
    }
    .build()
    .map_err(|e| anyhow!("build deposit: {e}"))?;
    out.push(wrap_msg(&deposit, 1));

    let nonce: u64 = 0;
    let mut block = orig_block;
    let mut ts = orig_ts;
    block += 1;
    ts += 10;

    match plan {
        FixturePlan::DepositOnly => {}
        FixturePlan::TwoDeposits => {
            let dep2 = DepositBuilder {
                from: bridge_aliased_sender(),
                to: dev,
                amount: U256::from(1u64),
                l1_block_number: block,
                timestamp: ts,
                request_seq: next_req(&mut request_seq),
                base_fee_l1: BASE_FEE_L1,
            }
            .build()
            .map_err(|e| anyhow!("build 2nd deposit: {e}"))?;
            out.push(wrap_msg(&dep2, 2));
        }
        FixturePlan::DepositPlusSignedCall {
            target,
            selector,
            kind,
            use_eip1559,
        } => {
            let calldata = encode_call(selector, kind);
            let m = build_signed(
                signing_key,
                use_eip1559,
                nonce,
                Some(target),
                U256::ZERO,
                Bytes::from(calldata),
                DEFAULT_GAS_LIMIT,
                block,
                ts,
            )?;
            out.push(wrap_msg(&m, 1));
        }
        FixturePlan::DepositPlusEoaTransfer { use_eip1559 } => {
            let to: Address = address!("00000000000000000000000000000000000000bb");
            let m = build_signed(
                signing_key,
                use_eip1559,
                nonce,
                Some(to),
                U256::from(1u64),
                Bytes::new(),
                21_000,
                block,
                ts,
            )?;
            out.push(wrap_msg(&m, 1));
        }
        FixturePlan::RetryableSubmitThenRedeem { use_eip1559 } => {
            let retry_target: Address = address!("00000000000000000000000000000000000000bb");
            let retry_seq = next_req(&mut request_seq);
            let parent_id = arb_test_harness::messaging::encoding::request_id_from_seq(retry_seq);
            let retry = RetryableSubmitBuilder {
                l1_sender: dev,
                to: retry_target,
                l2_call_value: U256::ZERO,
                deposit_value: U256::from(2_000_000_000_000_000u64),
                max_submission_fee: U256::from(500_000_000_000u64),
                excess_fee_refund_address: dev,
                call_value_refund_address: dev,
                gas_limit: 100_000,
                max_fee_per_gas: U256::from(1_000_000_000u64),
                data: Bytes::new(),
                l1_block_number: block,
                timestamp: ts,
                request_id: Some(parent_id),
            }
            .build()
            .map_err(|e| anyhow!("build retryable submit: {e}"))?;
            out.push(wrap_msg(&retry, 1));

            block += 1;
            ts += 10;
            // Sub request id at index 0 = keccak256(parent_id || U256(0)).
            let mut preimage = [0u8; 64];
            preimage[..32].copy_from_slice(parent_id.as_slice());
            let ticket_id = B256::from(keccak256(preimage));

            let mut calldata = Vec::with_capacity(36);
            calldata.extend_from_slice(&keccak4("redeem(bytes32)"));
            calldata.extend_from_slice(ticket_id.as_slice());
            let m = build_signed(
                signing_key,
                use_eip1559,
                nonce,
                Some(ARB_RETRYABLE_ADDR),
                U256::ZERO,
                Bytes::from(calldata),
                DEFAULT_GAS_LIMIT,
                block,
                ts,
            )?;
            out.push(wrap_msg(&m, 1));
        }
    }

    Ok(out)
}

fn encode_call(selector: [u8; 4], kind: ArgsKind) -> Vec<u8> {
    let mut buf = Vec::new();
    buf.extend_from_slice(&selector);
    match kind {
        ArgsKind::None => {}
        ArgsKind::Bool(b) => {
            let mut word = [0u8; 32];
            if b {
                word[31] = 1;
            }
            buf.extend_from_slice(&word);
        }
        ArgsKind::EmptyConstraintArray => {
            // dynamic uint64[3][] empty: head offset = 0x20, then length = 0.
            let mut off = [0u8; 32];
            off[31] = 0x20;
            buf.extend_from_slice(&off);
            buf.extend_from_slice(&[0u8; 32]);
        }
        ArgsKind::RawZeroPad32 => {
            buf.extend_from_slice(&[0u8; 28]);
        }
    }
    buf
}

#[allow(clippy::too_many_arguments)]
fn build_signed(
    signing_key: B256,
    use_eip1559: bool,
    nonce: u64,
    to: Option<Address>,
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
        to,
        value,
        data,
        gas_limit,
        gas_price: DEFAULT_GAS_PRICE,
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
