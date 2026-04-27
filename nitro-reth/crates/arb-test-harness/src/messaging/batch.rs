use alloy_primitives::{Address, B256, U256};

use crate::messaging::{
    encoding::{
        encode_address, encode_hash, encode_uint256, encode_uint64, request_id_from_seq,
    },
    kinds, L1Message, L1MessageHeader, MessageBuilder,
};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BatchPostingVariant {
    V1,
    V2 { extra_gas: u64 },
}

#[derive(Debug, Clone)]
pub struct BatchBuilder {
    pub batch_poster: Address,
    pub batch_timestamp: u64,
    pub data_hash: B256,
    pub batch_number: u64,
    pub l1_base_fee: U256,
    pub variant: BatchPostingVariant,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub request_seq: u64,
    pub base_fee_l1: u64,
}

impl BatchBuilder {
    pub fn encode_body(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(32 + 20 + 32 + 32 + 32 + 8);
        out.extend_from_slice(&encode_uint256(U256::from(self.batch_timestamp)));
        out.extend_from_slice(&encode_address(self.batch_poster));
        out.extend_from_slice(&encode_hash(self.data_hash));
        out.extend_from_slice(&encode_uint256(U256::from(self.batch_number)));
        out.extend_from_slice(&encode_uint256(self.l1_base_fee));
        if let BatchPostingVariant::V2 { extra_gas } = self.variant {
            out.extend_from_slice(&encode_uint64(extra_gas));
        }
        out
    }
}

impl MessageBuilder for BatchBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body();
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_BATCH_POSTING_REPORT,
                sender: self.batch_poster,
                block_number: self.l1_block_number,
                timestamp: self.timestamp,
                request_id: Some(request_id_from_seq(self.request_seq)),
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
    use alloy_primitives::{address, b256};
    use arbos::arbos_types::parse_batch_posting_report_fields;

    fn sample(variant: BatchPostingVariant) -> BatchBuilder {
        BatchBuilder {
            batch_poster: address!("a4b000000000000000000073657175656e636572"),
            batch_timestamp: 1_700_000_000,
            data_hash: b256!(
                "5566778899aabbccddeeff00112233445566778899aabbccddeeff0011223344"
            ),
            batch_number: 42,
            l1_base_fee: U256::from(30_000_000_000u64),
            variant,
            l1_block_number: 300,
            timestamp: 1_700_000_010,
            request_seq: 13,
            base_fee_l1: 30_000_000_000,
        }
    }

    #[test]
    fn v1_body_size_excludes_extra_gas() {
        let msg = sample(BatchPostingVariant::V1).build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body.len(), 32 + 20 + 32 + 32 + 32);
    }

    #[test]
    fn v2_body_size_includes_extra_gas() {
        let msg = sample(BatchPostingVariant::V2 { extra_gas: 1234 })
            .build()
            .unwrap();
        let body = decode_body(&msg);
        assert_eq!(body.len(), 32 + 20 + 32 + 32 + 32 + 8);
        assert_eq!(
            u64::from_be_bytes(body[148..156].try_into().unwrap()),
            1234u64
        );
    }

    #[test]
    fn parses_into_batch_posting_report() {
        let s = sample(BatchPostingVariant::V2 { extra_gas: 9999 });
        let msg = s.build().unwrap();
        let parsed = round_trip(&msg);
        let fields = parse_batch_posting_report_fields(&parsed.l2_msg).unwrap();
        assert_eq!(fields.batch_timestamp, s.batch_timestamp);
        assert_eq!(fields.batch_poster, s.batch_poster);
        assert_eq!(fields.data_hash, s.data_hash);
        assert_eq!(fields.batch_number, s.batch_number);
        assert_eq!(fields.l1_base_fee_estimate, s.l1_base_fee);
        assert_eq!(fields.extra_gas, 9999);
    }

    #[test]
    fn v1_parses_with_zero_extra_gas() {
        let s = sample(BatchPostingVariant::V1);
        let msg = s.build().unwrap();
        let parsed = round_trip(&msg);
        let fields = parse_batch_posting_report_fields(&parsed.l2_msg).unwrap();
        assert_eq!(fields.extra_gas, 0);
    }

    #[test]
    fn json_round_trip() {
        let msg = sample(BatchPostingVariant::V2 { extra_gas: 5 })
            .build()
            .unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_BATCH_POSTING_REPORT);
        assert_eq!(back.l2_msg, msg.l2_msg);
    }
}
