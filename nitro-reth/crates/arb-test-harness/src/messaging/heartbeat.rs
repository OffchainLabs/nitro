use alloy_primitives::{Address, U256};

use crate::messaging::{kinds, L1Message, L1MessageHeader, MessageBuilder};

#[derive(Debug, Clone)]
pub struct HeartbeatBuilder {
    pub sender: Address,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub chain_id: U256,
    pub body: HeartbeatBody,
    pub base_fee_l1: u64,
}

#[derive(Debug, Clone)]
pub enum HeartbeatBody {
    ChainIdOnly,
    V0 {
        chain_config: Vec<u8>,
    },
    V1 {
        initial_l1_base_fee: U256,
        chain_config: Vec<u8>,
    },
}

impl HeartbeatBuilder {
    pub fn encode_body(&self) -> Vec<u8> {
        let mut out = Vec::new();
        out.extend_from_slice(&self.chain_id.to_be_bytes::<32>());
        match &self.body {
            HeartbeatBody::ChainIdOnly => {}
            HeartbeatBody::V0 { chain_config } => {
                out.push(0);
                out.extend_from_slice(chain_config);
            }
            HeartbeatBody::V1 {
                initial_l1_base_fee,
                chain_config,
            } => {
                out.push(1);
                out.extend_from_slice(&initial_l1_base_fee.to_be_bytes::<32>());
                out.extend_from_slice(chain_config);
            }
        }
        out
    }
}

impl MessageBuilder for HeartbeatBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body();
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_HEARTBEAT,
                sender: self.sender,
                block_number: self.l1_block_number,
                timestamp: self.timestamp,
                request_id: None,
                base_fee_l1: self.base_fee_l1,
            },
            l2_msg: crate::messaging::b64_l2_msg(&body.into()),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::messaging::test_support::{decode_body, round_trip};
    use arbos::arbos_types::parse_init_message;

    fn sample(body: HeartbeatBody) -> HeartbeatBuilder {
        HeartbeatBuilder {
            sender: Address::ZERO,
            l1_block_number: 0,
            timestamp: 0,
            chain_id: U256::from(421_614u64),
            body,
            base_fee_l1: 0,
        }
    }

    #[test]
    fn chain_id_only_body_size() {
        let msg = sample(HeartbeatBody::ChainIdOnly).build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body.len(), 32);
    }

    #[test]
    fn v1_body_includes_basefee_byte_then_basefee() {
        let s = sample(HeartbeatBody::V1 {
            initial_l1_base_fee: U256::from(50_000_000_000u64),
            chain_config: b"{}".to_vec(),
        });
        let msg = s.build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body[32], 1);
        assert_eq!(
            U256::from_be_slice(&body[33..65]),
            U256::from(50_000_000_000u64)
        );
        assert_eq!(&body[65..], b"{}");
    }

    #[test]
    fn parses_into_init_message() {
        let s = sample(HeartbeatBody::V1 {
            initial_l1_base_fee: U256::from(30_000_000_000u64),
            chain_config: b"{\"chainId\":421614}".to_vec(),
        });
        let msg = s.build().unwrap();
        let parsed = round_trip(&msg);
        let init = parse_init_message(&parsed.l2_msg).unwrap();
        assert_eq!(init.chain_id, U256::from(421_614u64));
        assert_eq!(init.initial_l1_base_fee, U256::from(30_000_000_000u64));
        assert_eq!(init.serialized_chain_config, b"{\"chainId\":421614}");
    }

    #[test]
    fn json_round_trip() {
        let msg = sample(HeartbeatBody::ChainIdOnly).build().unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_HEARTBEAT);
        assert!(back.header.request_id.is_none());
        assert_eq!(back.l2_msg, msg.l2_msg);
    }
}
