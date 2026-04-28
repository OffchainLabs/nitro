pub mod batch;
pub mod contract;
pub mod delayed;
pub mod deposit;
pub mod heartbeat;
pub mod internal;
pub mod kinds;
pub mod retryable;
pub mod signed_tx;
pub mod unsigned;

pub use batch::{BatchBuilder, BatchPostingVariant};
pub use contract::ContractTxBuilder;
pub use delayed::DelayedTxBuilder;
pub use deposit::DepositBuilder;
pub use heartbeat::{HeartbeatBody, HeartbeatBuilder};
pub use internal::{InternalTxBuilder, InternalTxKind};
pub use retryable::{apply_l1_to_l2_alias, RetryableSubmitBuilder, L1_TO_L2_ALIAS_OFFSET};
pub use signed_tx::{derive_address as l2_signing_key_to_address, L2TxKind, SignedL2TxBuilder};
pub use unsigned::UnsignedUserTxBuilder;

use alloy_primitives::Bytes;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct L1Message {
    pub header: L1MessageHeader,
    #[serde(rename = "l2Msg")]
    pub l2_msg: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct L1MessageHeader {
    pub kind: u8,
    pub sender: alloy_primitives::Address,
    #[serde(rename = "blockNumber")]
    pub block_number: u64,
    pub timestamp: u64,
    #[serde(rename = "requestId", skip_serializing_if = "Option::is_none")]
    pub request_id: Option<alloy_primitives::B256>,
    #[serde(rename = "baseFeeL1")]
    pub base_fee_l1: u64,
}

pub trait MessageBuilder {
    fn build(&self) -> crate::Result<L1Message>;
}

pub fn b64_l2_msg(bytes: &Bytes) -> String {
    use base64::Engine;
    base64::engine::general_purpose::STANDARD.encode(bytes.as_ref())
}

pub const MAX_L2_MESSAGE_SIZE: usize = 256 * 1024;

pub mod encoding {
    use alloy_primitives::{Address, B256, U256};

    pub fn encode_address(addr: Address) -> [u8; 20] {
        let mut buf = [0u8; 20];
        buf.copy_from_slice(addr.as_slice());
        buf
    }

    pub fn encode_address256(addr: Address) -> [u8; 32] {
        let mut buf = [0u8; 32];
        buf[12..].copy_from_slice(addr.as_slice());
        buf
    }

    pub fn encode_uint256(val: U256) -> [u8; 32] {
        val.to_be_bytes::<32>()
    }

    pub fn encode_uint64(val: u64) -> [u8; 8] {
        val.to_be_bytes()
    }

    pub fn encode_hash(hash: B256) -> [u8; 32] {
        hash.0
    }

    pub fn request_id_from_seq(seq: u64) -> B256 {
        B256::from(encode_uint256(U256::from(seq)))
    }
}

#[cfg(test)]
pub(crate) mod test_support {
    use super::*;
    use alloy_primitives::B256;
    use base64::Engine;

    pub fn decode_body(msg: &L1Message) -> Vec<u8> {
        base64::engine::general_purpose::STANDARD
            .decode(msg.l2_msg.as_bytes())
            .expect("valid base64 in test fixture")
    }

    pub fn round_trip(msg: &L1Message) -> arbos::arbos_types::L1IncomingMessage {
        let body = decode_body(msg);
        let mut wire = Vec::with_capacity(1 + 32 + 8 + 8 + 32 + 32 + body.len());
        wire.push(msg.header.kind);
        let mut padded = [0u8; 32];
        padded[12..].copy_from_slice(msg.header.sender.as_slice());
        wire.extend_from_slice(&padded);
        wire.extend_from_slice(&msg.header.block_number.to_be_bytes());
        wire.extend_from_slice(&msg.header.timestamp.to_be_bytes());
        wire.extend_from_slice(msg.header.request_id.unwrap_or(B256::ZERO).as_slice());
        let mut fee_buf = [0u8; 32];
        fee_buf[24..].copy_from_slice(&msg.header.base_fee_l1.to_be_bytes());
        wire.extend_from_slice(&fee_buf);
        wire.extend_from_slice(&body);
        arbos::arbos_types::parse_incoming_l1_message(&wire).expect("parses cleanly")
    }
}
