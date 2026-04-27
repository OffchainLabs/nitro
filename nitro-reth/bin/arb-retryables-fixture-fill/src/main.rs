//! Fill `_TODO_*` placeholders in retryable spec fixture `l2Msg` fields with
//! real built envelopes. Headers are preserved as-authored; only the
//! `l2Msg` string is replaced with the base64 of the appropriate body.

mod common;

use std::path::{Path, PathBuf};

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use anyhow::{anyhow, bail, Context, Result};
use arb_test_harness::messaging::{
    encoding::request_id_from_seq, kinds, DepositBuilder, L1Message, L2TxKind, MessageBuilder,
    RetryableSubmitBuilder, SignedL2TxBuilder,
};
use base64::Engine;
use walkdir::WalkDir;

use common::{bridge_aliased_sender, dev_address, dev_signing_key};

const ARB_RETRYABLE_ADDR: Address = address!("000000000000000000000000000000000000006e");
const ARB_OWNER_ADDR: Address = address!("0000000000000000000000000000000000000070");

const SEQUENCER_HEADER_SENDER: Address =
    address!("a4b000000000000000000073657175656e636572");

const CHAIN_ID: u64 = 421614;
const DEFAULT_DEPOSIT_AMOUNT: u128 = 1_000_000_000_000_000_000_000u128; // 1000 ETH
const DEFAULT_GAS_PRICE: u128 = 1_000_000_000;
const DEFAULT_MAX_FEE: u128 = 1_000_000_000;
const DEFAULT_MAX_PRIORITY: u128 = 0;
const BASE_FEE_L1: u64 = 0;

fn main() -> Result<()> {
    let workspace_root = locate_workspace_root()?;
    let fixtures_root = workspace_root.join("crates/arb-spec-tests/fixtures/retryables");
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
        let body = std::fs::read_to_string(path)
            .with_context(|| format!("read {}", path.display()))?;
        if !body.contains("_TODO_") {
            continue;
        }
        match rewrite_fixture(path) {
            Ok(()) => touched.push(path.to_path_buf()),
            Err(e) => errors.push((path.to_path_buf(), e.to_string())),
        }
    }

    println!("touched retryable fixtures ({}):", touched.len());
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
    println!("all retryable fixtures parse cleanly via parse_incoming_l1_message");
    Ok(())
}

fn rewrite_fixture(path: &Path) -> Result<()> {
    let body = std::fs::read_to_string(path)
        .with_context(|| format!("read {}", path.display()))?;
    let mut value: serde_json::Value = serde_json::from_str(&body)
        .with_context(|| format!("parse {}", path.display()))?;

    let messages = value
        .get_mut("messages")
        .and_then(|v| v.as_array_mut())
        .ok_or_else(|| anyhow!("fixture missing messages array"))?;

    // First pass: collect each message's header info so later messages can
    // reference earlier kind=9 request ids when building redeem/keepalive/cancel.
    let headers: Vec<MessageHeader> = messages
        .iter()
        .map(|m| read_header(m))
        .collect::<Result<Vec<_>>>()?;

    // Track signed-tx nonce across messages from the dev address.
    let mut signed_nonce: u64 = 0;

    for (idx, msg) in messages.iter_mut().enumerate() {
        let placeholder = msg
            .get("message")
            .and_then(|m| m.get("l2Msg"))
            .and_then(|s| s.as_str())
            .unwrap_or("")
            .to_string();
        if !placeholder.starts_with("_TODO_") {
            continue;
        }

        let header = &headers[idx];
        let new_l2_msg = build_l2_msg(&placeholder, header, &headers, idx, &mut signed_nonce)
            .with_context(|| {
                format!("msg {idx} placeholder {placeholder}")
            })?;

        msg.get_mut("message")
            .and_then(|m| m.as_object_mut())
            .ok_or_else(|| anyhow!("msg {idx}: message not object"))?
            .insert("l2Msg".into(), serde_json::Value::String(new_l2_msg));
    }

    let pretty = serde_json::to_string_pretty(&value)?;
    std::fs::write(path, pretty + "\n")
        .with_context(|| format!("write {}", path.display()))?;
    Ok(())
}

#[derive(Debug, Clone)]
struct MessageHeader {
    kind: u8,
    block_number: u64,
    timestamp: u64,
    request_id: Option<B256>,
    request_seq: Option<u64>,
}

fn read_header(msg: &serde_json::Value) -> Result<MessageHeader> {
    let header = msg
        .get("message")
        .and_then(|m| m.get("header"))
        .ok_or_else(|| anyhow!("missing header"))?;
    let kind = header
        .get("kind")
        .and_then(|v| v.as_u64())
        .ok_or_else(|| anyhow!("missing kind"))? as u8;
    let block_number = header
        .get("blockNumber")
        .and_then(|v| v.as_u64())
        .unwrap_or(0);
    let timestamp = header.get("timestamp").and_then(|v| v.as_u64()).unwrap_or(0);
    let request_id = header
        .get("requestId")
        .and_then(|v| v.as_str())
        .and_then(|s| {
            let raw = s.trim_start_matches("0x");
            let mut buf = [0u8; 32];
            for (i, b) in buf.iter_mut().enumerate() {
                let pair = raw.get(i * 2..i * 2 + 2)?;
                *b = u8::from_str_radix(pair, 16).ok()?;
            }
            Some(B256::from(buf))
        });
    let request_seq = request_id.and_then(|id| {
        // The fixtures use sequential ids encoded as big-endian u256; recover
        // the low u64 only when the upper 24 bytes are all zero.
        let bytes = id.as_slice();
        if bytes[..24].iter().any(|b| *b != 0) {
            None
        } else {
            let mut be = [0u8; 8];
            be.copy_from_slice(&bytes[24..]);
            Some(u64::from_be_bytes(be))
        }
    });
    Ok(MessageHeader {
        kind,
        block_number,
        timestamp,
        request_id,
        request_seq,
    })
}

/// Find the most recent kind=9 SubmitRetryable message strictly before `idx`
/// and return its request id (parent retryable id).
fn find_prior_retryable_parent_id(headers: &[MessageHeader], idx: usize) -> Option<B256> {
    for h in headers[..idx].iter().rev() {
        if h.kind == kinds::KIND_SUBMIT_RETRYABLE {
            return h.request_id;
        }
    }
    None
}

/// Sub-request id at index `sub_idx` for a kind=9 ticket: keccak256(parent || U256(sub_idx)).
fn ticket_id_for_sub(parent_id: B256, sub_idx: u64) -> B256 {
    let mut preimage = [0u8; 64];
    preimage[..32].copy_from_slice(parent_id.as_slice());
    let sub = U256::from(sub_idx).to_be_bytes::<32>();
    preimage[32..].copy_from_slice(&sub);
    B256::from(keccak256(preimage))
}

fn keccak4(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn build_l2_msg(
    placeholder: &str,
    header: &MessageHeader,
    headers: &[MessageHeader],
    idx: usize,
    signed_nonce: &mut u64,
) -> Result<String> {
    let dev = dev_address();
    let signing_key = dev_signing_key();
    let block = header.block_number;
    let ts = header.timestamp;

    let msg = match placeholder {
        "_TODO_deposit_to_dev" => build_deposit(
            U256::from(DEFAULT_DEPOSIT_AMOUNT),
            block,
            ts,
            header.request_seq.unwrap_or(1),
        )?,

        "_TODO_retryable_submit_basic_auto_redeem" => build_retryable_submit(
            RetryableArgs {
                to: address!("00000000000000000000000000000000000000bb"),
                l2_call_value: U256::from(1_000_000_000_000_000u64),
                deposit_value: U256::from(50_000_000_000_000_000u64),
                max_submission_fee: U256::from(500_000_000_000u64),
                gas_limit: 3_000_000,
                max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                data: Bytes::new(),
                beneficiary: dev,
                excess_fee_refund: dev,
            },
            block,
            ts,
            header.request_id,
        )?,

        "_TODO_retryable_submit_low_gas_no_auto_redeem" => build_retryable_submit(
            RetryableArgs {
                to: address!("00000000000000000000000000000000000000bb"),
                l2_call_value: U256::from(1_000_000_000_000_000u64),
                deposit_value: U256::from(2_000_000_000_000_000u64),
                max_submission_fee: U256::from(500_000_000_000u64),
                gas_limit: 21_000,
                max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                data: Bytes::new(),
                beneficiary: dev,
                excess_fee_refund: dev,
            },
            block,
            ts,
            header.request_id,
        )?,

        "_TODO_retryable_submit_low_gas_no_auto_redeem_beneficiary_dev" => build_retryable_submit(
            RetryableArgs {
                to: address!("00000000000000000000000000000000000000bb"),
                l2_call_value: U256::from(1_000_000_000_000_000u64),
                deposit_value: U256::from(2_000_000_000_000_000u64),
                max_submission_fee: U256::from(500_000_000_000u64),
                gas_limit: 21_000,
                max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                data: Bytes::new(),
                beneficiary: dev,
                excess_fee_refund: dev,
            },
            block,
            ts,
            header.request_id,
        )?,

        "_TODO_retryable_submit_with_backlog_growth_3M_gas" => build_retryable_submit(
            RetryableArgs {
                to: address!("00000000000000000000000000000000000000bb"),
                l2_call_value: U256::from(1_000_000_000_000_000u64),
                deposit_value: U256::from(50_000_000_000_000_000u64),
                max_submission_fee: U256::from(500_000_000_000u64),
                gas_limit: 3_000_000,
                max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                data: Bytes::new(),
                beneficiary: dev,
                excess_fee_refund: dev,
            },
            block,
            ts,
            header.request_id,
        )?,

        "_TODO_retryable_submit_with_backlog_growth_5M_gas" => build_retryable_submit(
            RetryableArgs {
                to: address!("00000000000000000000000000000000000000bb"),
                l2_call_value: U256::from(1_000_000_000_000_000u64),
                deposit_value: U256::from(50_000_000_000_000_000u64),
                max_submission_fee: U256::from(500_000_000_000u64),
                gas_limit: 5_000_000,
                max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                data: Bytes::new(),
                beneficiary: dev,
                excess_fee_refund: dev,
            },
            block,
            ts,
            header.request_id,
        )?,

        "_TODO_retryable_submit_underfunded_deposit_below_callvalue_plus_fees" => {
            // deposit << l2_call_value + max_submission_fee + gas_limit*max_fee_per_gas.
            build_retryable_submit(
                RetryableArgs {
                    to: address!("00000000000000000000000000000000000000bb"),
                    l2_call_value: U256::from(1_000_000_000_000_000_000u64),
                    deposit_value: U256::from(100u64),
                    max_submission_fee: U256::from(500_000_000_000u64),
                    gas_limit: 1_000_000,
                    max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                    data: Bytes::new(),
                    beneficiary: dev,
                    excess_fee_refund: dev,
                },
                block,
                ts,
                header.request_id,
            )?
        }

        "_TODO_retryable_submit_zero_callvalue_pure_call" => build_retryable_submit(
            RetryableArgs {
                to: address!("00000000000000000000000000000000000000bb"),
                l2_call_value: U256::ZERO,
                deposit_value: U256::from(50_000_000_000_000_000u64),
                max_submission_fee: U256::from(500_000_000_000u64),
                gas_limit: 3_000_000,
                max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                data: Bytes::new(),
                beneficiary: dev,
                excess_fee_refund: dev,
            },
            block,
            ts,
            header.request_id,
        )?,

        "_TODO_retryable_submit_inner_call_reverts" => {
            // Call into the EOA-shaped revert helper at 0x...dead with a calldata
            // that selects a non-existent function so the retry's inner call
            // reverts. The destination has no code so the auto-redeem will fail.
            let mut data = Vec::with_capacity(4);
            data.extend_from_slice(&keccak4("doRevert()"));
            build_retryable_submit(
                RetryableArgs {
                    to: address!("000000000000000000000000000000000000dead"),
                    l2_call_value: U256::from(1_000_000_000_000_000u64),
                    deposit_value: U256::from(50_000_000_000_000_000u64),
                    max_submission_fee: U256::from(500_000_000_000u64),
                    gas_limit: 3_000_000,
                    max_fee_per_gas: U256::from(DEFAULT_MAX_FEE),
                    data: Bytes::from(data),
                    beneficiary: dev,
                    excess_fee_refund: dev,
                },
                block,
                ts,
                header.request_id,
            )?
        }

        "_TODO_signed_redeem_ticket0_gas3M"
        | "_TODO_signed_redeem_ticket0_gas3M_drains_backlog" => {
            let parent = find_prior_retryable_parent_id(headers, idx)
                .ok_or_else(|| anyhow!("no prior kind=9 message before idx {idx}"))?;
            let ticket = ticket_id_for_sub(parent, 0);
            let calldata = encode_redeem_keepalive_cancel("redeem(bytes32)", ticket);
            let nonce = *signed_nonce;
            *signed_nonce += 1;
            build_signed_call(
                signing_key,
                true,
                nonce,
                ARB_RETRYABLE_ADDR,
                U256::ZERO,
                Bytes::from(calldata),
                3_000_000,
                block,
                ts,
            )?
        }

        "_TODO_signed_redeem_ticket0_gas2M_partial_backlog" => {
            let parent = find_prior_retryable_parent_id(headers, idx)
                .ok_or_else(|| anyhow!("no prior kind=9 message before idx {idx}"))?;
            let ticket = ticket_id_for_sub(parent, 0);
            let calldata = encode_redeem_keepalive_cancel("redeem(bytes32)", ticket);
            let nonce = *signed_nonce;
            *signed_nonce += 1;
            build_signed_call(
                signing_key,
                true,
                nonce,
                ARB_RETRYABLE_ADDR,
                U256::ZERO,
                Bytes::from(calldata),
                2_000_000,
                block,
                ts,
            )?
        }

        "_TODO_signed_keepalive_ticket0" => {
            let parent = find_prior_retryable_parent_id(headers, idx)
                .ok_or_else(|| anyhow!("no prior kind=9 message before idx {idx}"))?;
            let ticket = ticket_id_for_sub(parent, 0);
            let calldata = encode_redeem_keepalive_cancel("keepalive(bytes32)", ticket);
            let nonce = *signed_nonce;
            *signed_nonce += 1;
            build_signed_call(
                signing_key,
                true,
                nonce,
                ARB_RETRYABLE_ADDR,
                U256::ZERO,
                Bytes::from(calldata),
                500_000,
                block,
                ts,
            )?
        }

        "_TODO_signed_cancel_ticket0_from_beneficiary" => {
            let parent = find_prior_retryable_parent_id(headers, idx)
                .ok_or_else(|| anyhow!("no prior kind=9 message before idx {idx}"))?;
            let ticket = ticket_id_for_sub(parent, 0);
            let calldata = encode_redeem_keepalive_cancel("cancel(bytes32)", ticket);
            let nonce = *signed_nonce;
            *signed_nonce += 1;
            build_signed_call(
                signing_key,
                true,
                nonce,
                ARB_RETRYABLE_ADDR,
                U256::ZERO,
                Bytes::from(calldata),
                500_000,
                block,
                ts,
            )?
        }

        "_TODO_signed_set_gas_pricing_constraints_two_entries" => {
            let calldata = encode_set_gas_pricing_constraints_two_entries();
            let nonce = *signed_nonce;
            *signed_nonce += 1;
            build_signed_call(
                signing_key,
                true,
                nonce,
                ARB_OWNER_ADDR,
                U256::ZERO,
                Bytes::from(calldata),
                500_000,
                block,
                ts,
            )?
        }

        other => bail!("unknown placeholder: {other}"),
    };

    let raw = base64::engine::general_purpose::STANDARD
        .decode(msg.l2_msg.as_bytes())
        .map_err(|e| anyhow!("decode built l2_msg: {e}"))?;
    Ok(base64::engine::general_purpose::STANDARD.encode(raw))
}

struct RetryableArgs {
    to: Address,
    l2_call_value: U256,
    deposit_value: U256,
    max_submission_fee: U256,
    gas_limit: u64,
    max_fee_per_gas: U256,
    data: Bytes,
    beneficiary: Address,
    excess_fee_refund: Address,
}

fn build_deposit(
    amount: U256,
    block: u64,
    ts: u64,
    request_seq: u64,
) -> Result<L1Message> {
    DepositBuilder {
        from: bridge_aliased_sender(),
        to: dev_address(),
        amount,
        l1_block_number: block,
        timestamp: ts,
        request_seq,
        base_fee_l1: BASE_FEE_L1,
    }
    .build()
    .map_err(|e| anyhow!("build deposit: {e}"))
}

fn build_retryable_submit(
    args: RetryableArgs,
    block: u64,
    ts: u64,
    request_id: Option<B256>,
) -> Result<L1Message> {
    // The aliased sender in the header is preserved as-is by the caller; the
    // RetryableSubmitBuilder applies aliasing internally so we feed it the
    // pre-alias dev address (the kind=9 ticket's "from" lands as aliased dev).
    RetryableSubmitBuilder {
        l1_sender: dev_address(),
        to: args.to,
        l2_call_value: args.l2_call_value,
        deposit_value: args.deposit_value,
        max_submission_fee: args.max_submission_fee,
        excess_fee_refund_address: args.excess_fee_refund,
        call_value_refund_address: args.beneficiary,
        gas_limit: args.gas_limit,
        max_fee_per_gas: args.max_fee_per_gas,
        data: args.data,
        l1_block_number: block,
        timestamp: ts,
        request_id: Some(request_id.unwrap_or_else(|| request_id_from_seq(2))),
    }
    .build()
    .map_err(|e| anyhow!("build retryable submit: {e}"))
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
) -> Result<L1Message> {
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

fn encode_redeem_keepalive_cancel(sig: &str, ticket: B256) -> Vec<u8> {
    let mut buf = Vec::with_capacity(36);
    buf.extend_from_slice(&keccak4(sig));
    buf.extend_from_slice(ticket.as_slice());
    buf
}

/// `setGasPricingConstraints(uint64[3][])` calldata with two constraint
/// entries `[gasUsedHigh, gasUsedLow, decay]` populated with simple sentinel
/// values. ABI layout:
///   selector
///   offset to dynamic outer array (32 bytes, value 0x20)
///   outer array length (32 bytes, value 2)
///   tuple0 word0 (uint64, padded)
///   tuple0 word1 (uint64, padded)
///   tuple0 word2 (uint64, padded)
///   tuple1 word0 (uint64, padded)
///   tuple1 word1 (uint64, padded)
///   tuple1 word2 (uint64, padded)
fn encode_set_gas_pricing_constraints_two_entries() -> Vec<u8> {
    let mut buf = Vec::with_capacity(4 + 32 * 8);
    buf.extend_from_slice(&keccak4("setGasPricingConstraints(uint64[3][])"));
    let mut head = [0u8; 32];
    head[31] = 0x20;
    buf.extend_from_slice(&head);
    let mut len = [0u8; 32];
    len[31] = 2;
    buf.extend_from_slice(&len);
    let entries: [[u64; 3]; 2] = [[10_000_000, 8_000_000, 100], [20_000_000, 15_000_000, 200]];
    for entry in entries.iter() {
        for v in entry.iter() {
            let mut word = [0u8; 32];
            word[24..].copy_from_slice(&v.to_be_bytes());
            buf.extend_from_slice(&word);
        }
    }
    buf
}

fn verify_fixture(path: &Path) -> Result<()> {
    let body = std::fs::read_to_string(path)?;
    let value: serde_json::Value = serde_json::from_str(&body)?;
    let chain_id = value
        .get("genesis")
        .and_then(|g| g.get("config"))
        .and_then(|c| c.get("chainId"))
        .and_then(|v| v.as_u64())
        .unwrap_or(CHAIN_ID);
    let messages = value
        .get("messages")
        .and_then(|v| v.as_array())
        .ok_or_else(|| anyhow!("missing messages"))?;
    for (i, msg) in messages.iter().enumerate() {
        common::verify_l1_message(msg, i, chain_id)?;
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
