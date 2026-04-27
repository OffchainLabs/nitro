//! Shared helpers: dev key derivation and L1Message round-trip verification.

use alloy_primitives::{keccak256, Address, B256};
use anyhow::{anyhow, bail, Result};
use arb_test_harness::messaging::l2_signing_key_to_address;
use base64::Engine;

const DEV_KEY_LABEL: &[u8] = b"arbreth-test-harness-dev-key-v1";

pub fn dev_signing_key() -> B256 {
    keccak256(DEV_KEY_LABEL)
}

pub fn dev_address() -> Address {
    l2_signing_key_to_address(dev_signing_key())
}

fn parse_address(s: &str) -> Option<Address> {
    let raw = s.trim_start_matches("0x");
    if raw.len() != 40 {
        return None;
    }
    let mut buf = [0u8; 20];
    if hex_into(raw, &mut buf).is_err() {
        return None;
    }
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

pub fn verify_l1_message(msg: &serde_json::Value, i: usize) -> Result<()> {
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
    let sender =
        parse_address(sender_str).ok_or_else(|| anyhow!("msg {i}: bad sender {sender_str}"))?;
    let block_number = header
        .get("blockNumber")
        .and_then(|v| v.as_u64())
        .unwrap_or(0);
    let timestamp = header.get("timestamp").and_then(|v| v.as_u64()).unwrap_or(0);
    let request_id = header
        .get("requestId")
        .and_then(|v| v.as_str())
        .map(|s| {
            let raw = s.trim_start_matches("0x");
            let mut buf = [0u8; 32];
            let _ = hex_into(raw, &mut buf);
            buf
        });
    let base_fee_l1 = header.get("baseFeeL1").and_then(|v| v.as_u64()).unwrap_or(0);
    let l2_msg_b64 = inner
        .get("l2Msg")
        .and_then(|v| v.as_str())
        .ok_or_else(|| anyhow!("msg {i}: missing l2Msg"))?;
    let l2_msg = base64::engine::general_purpose::STANDARD
        .decode(l2_msg_b64.as_bytes())
        .map_err(|e| anyhow!("msg {i}: bad base64: {e}"))?;

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
    Ok(())
}
