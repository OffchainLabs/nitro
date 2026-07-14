//! L1 incoming-message wire codec, used only by the legacy `d` table. Mirrors nitro's
//! `arbostypes.ParseIncomingL1Message` / `L1IncomingMessage.Serialize` byte-for-byte.

use alloy_primitives::{Address, B256, Bytes, U256};

use crate::{
    ConsensusDbError, Result,
    codecs::rlp::NilList,
    schema::{L1IncomingMessage, L1IncomingMessageHeader},
};

/// Fixed wire-header size: kind(1) + poster(32) + block(8) + timestamp(8) + requestId(32) + baseFee(32).
const WIRE_HEADER_LEN: usize = 1 + 32 + 8 + 8 + 32 + 32;

pub fn encode_l1_message_wire(msg: &L1IncomingMessage) -> Vec<u8> {
    let header = &msg.header;
    let mut buf = Vec::with_capacity(WIRE_HEADER_LEN + msg.l2msg.len());
    buf.push(header.kind);
    // poster: left-padded into a 32-byte word (address occupies the low 20 bytes).
    buf.extend_from_slice(B256::left_padding_from(header.poster.as_slice()).as_slice());
    buf.extend_from_slice(&header.block_number.to_be_bytes());
    buf.extend_from_slice(&header.timestamp.to_be_bytes());
    // request_id: 32 bytes. Nitro refuses to serialize a nil request id, and a legacy
    // `d` entry always has one, so `None` (never expected here) is written as zeroes.
    let request_id = header.request_id.0.map_or([0u8; 32], |id| id.0);
    buf.extend_from_slice(&request_id);
    buf.extend_from_slice(&header.l1_base_fee.to_be_bytes::<32>());
    buf.extend_from_slice(&msg.l2msg);
    buf
}

pub fn decode_l1_message_wire(bytes: &[u8]) -> Result<L1IncomingMessage> {
    if bytes.len() < WIRE_HEADER_LEN {
        return Err(ConsensusDbError::InvalidStoredValue);
    }
    let kind = bytes[0];
    // poster occupies the low 20 bytes of the 32-byte word at [1, 33).
    let poster = Address::from_slice(&bytes[13..33]);
    let block_number = u64::from_be_bytes(bytes[33..41].try_into().expect("8 bytes"));
    let timestamp = u64::from_be_bytes(bytes[41..49].try_into().expect("8 bytes"));
    let request_id = B256::from_slice(&bytes[49..81]);
    let l1_base_fee = U256::from_be_slice(&bytes[81..WIRE_HEADER_LEN]);
    let l2msg = Bytes::copy_from_slice(&bytes[WIRE_HEADER_LEN..]);
    Ok(L1IncomingMessage {
        header: L1IncomingMessageHeader {
            kind,
            poster,
            block_number,
            timestamp,
            // Nitro always populates request_id on parse (even if zero), so mirror that.
            request_id: NilList(Some(request_id)),
            l1_base_fee,
        },
        l2msg,
        legacy_batch_gas_cost: None,
        batch_data_stats: None,
    })
}
