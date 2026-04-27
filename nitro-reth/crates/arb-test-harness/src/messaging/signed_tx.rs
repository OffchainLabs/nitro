use alloy_primitives::{Address, Bytes};

use crate::messaging::{kinds, L1Message, L1MessageHeader, MessageBuilder};

#[derive(Debug, Clone)]
pub struct SignedTxBuilder {
    pub sender: Address,
    pub rlp_encoded_tx: Bytes,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub base_fee_l1: u64,
}

impl SignedTxBuilder {
    pub fn encode_body(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(1 + self.rlp_encoded_tx.len());
        out.push(kinds::KIND_SIGNED_L2_TX);
        out.extend_from_slice(&self.rlp_encoded_tx);
        out
    }
}

impl MessageBuilder for SignedTxBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body();
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_DEPOSIT,
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
    use alloy_consensus::{SignableTransaction, TxEnvelope, TxLegacy};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{address, Signature, U256};
    use arbos::parse_l2::{parse_l2_transactions, ParsedTransaction};

    fn sample_legacy_rlp() -> Bytes {
        let tx = TxLegacy {
            chain_id: Some(421614),
            nonce: 0,
            gas_price: 1_000_000_000u128,
            gas_limit: 21_000,
            to: alloy_primitives::TxKind::Call(address!(
                "00000000000000000000000000000000000000bb"
            )),
            value: U256::from(1_000u64),
            input: Bytes::new(),
        };
        let sig = Signature::new(
            U256::from(1u64),
            U256::from(2u64),
            false,
        );
        let envelope: TxEnvelope = tx.into_signed(sig).into();
        let mut buf = Vec::new();
        envelope.encode_2718(&mut buf);
        Bytes::from(buf)
    }

    fn sample() -> SignedTxBuilder {
        SignedTxBuilder {
            sender: address!("a4b000000000000000000073657175656e636572"),
            rlp_encoded_tx: sample_legacy_rlp(),
            l1_block_number: 10,
            timestamp: 1_700_000_000,
            base_fee_l1: 0,
        }
    }

    #[test]
    fn body_starts_with_sub_kind_byte() {
        let s = sample();
        let msg = s.build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body[0], kinds::KIND_SIGNED_L2_TX);
        assert_eq!(&body[1..], s.rlp_encoded_tx.as_ref());
    }

    #[test]
    fn parses_into_signed_tx() {
        let s = sample();
        let msg = s.build().unwrap();
        let parsed = round_trip(&msg);
        let txs = parse_l2_transactions(
            parsed.header.kind,
            parsed.header.poster,
            &parsed.l2_msg,
            parsed.header.request_id,
            parsed.header.l1_base_fee,
            421_614,
        )
        .unwrap();
        assert_eq!(txs.len(), 1);
        assert!(matches!(txs[0], ParsedTransaction::Signed(_)));
    }

    #[test]
    fn json_round_trip() {
        let msg = sample().build().unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_DEPOSIT);
        assert!(back.header.request_id.is_none());
        assert_eq!(back.l2_msg, msg.l2_msg);
    }
}
