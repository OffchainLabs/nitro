use alloy_primitives::{Address, Bytes};

use crate::messaging::{
    encoding::request_id_from_seq, kinds, L1Message, L1MessageHeader, MessageBuilder,
};

#[derive(Debug, Clone)]
pub struct DelayedTxBuilder {
    pub sender: Address,
    pub payload: Bytes,
    pub timeout_blocks: Option<u16>,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub request_seq: u64,
    pub base_fee_l1: u64,
}

impl DelayedTxBuilder {
    pub fn encode_body(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(self.payload.len() + 2);
        out.extend_from_slice(&self.payload);
        if let Some(timeout) = self.timeout_blocks {
            out.extend_from_slice(&timeout.to_be_bytes());
        }
        out
    }
}

impl MessageBuilder for DelayedTxBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body();
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_DELAYED_TX,
                sender: self.sender,
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
    use alloy_primitives::address;

    fn sample(payload: Bytes, timeout: Option<u16>) -> DelayedTxBuilder {
        DelayedTxBuilder {
            sender: address!("00000000000000000000000000000000000000ee"),
            payload,
            timeout_blocks: timeout,
            l1_block_number: 77,
            timestamp: 1_700_000_777,
            request_seq: 8,
            base_fee_l1: 0,
        }
    }

    #[test]
    fn body_includes_2byte_timeout_when_present() {
        let s = sample(Bytes::from(vec![0xa0, 0xb0]), Some(0x1234));
        let msg = s.build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body.len(), 4);
        assert_eq!(&body[..2], &[0xa0, 0xb0]);
        assert_eq!(&body[2..], &[0x12, 0x34]);
    }

    #[test]
    fn body_omits_timeout_when_absent() {
        let s = sample(Bytes::from(vec![0xa0, 0xb0]), None);
        let msg = s.build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body, vec![0xa0, 0xb0]);
    }

    #[test]
    fn parser_emits_no_transactions() {
        let s = sample(Bytes::from(vec![0xff, 0xfe]), Some(60));
        let msg = s.build().unwrap();
        let parsed = round_trip(&msg);
        let txs = arbos::parse_l2::parse_l2_transactions(
            parsed.header.kind,
            parsed.header.poster,
            &parsed.l2_msg,
            parsed.header.request_id,
            parsed.header.l1_base_fee,
            421_614,
        )
        .unwrap();
        assert!(txs.is_empty());
    }

    #[test]
    fn json_round_trip() {
        let msg = sample(Bytes::from(vec![1, 2, 3]), Some(10))
            .build()
            .unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_DELAYED_TX);
        assert_eq!(back.l2_msg, msg.l2_msg);
    }
}
