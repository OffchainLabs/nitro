use alloy_primitives::{Address, U256};

use crate::messaging::{
    encoding::{encode_address, encode_uint256, request_id_from_seq},
    kinds, L1Message, L1MessageHeader, MessageBuilder,
};

#[derive(Debug, Clone)]
pub struct DepositBuilder {
    pub from: Address,
    pub to: Address,
    pub amount: U256,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub request_seq: u64,
    pub base_fee_l1: u64,
}

impl DepositBuilder {
    pub fn encode_body(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(20 + 32);
        out.extend_from_slice(&encode_address(self.to));
        out.extend_from_slice(&encode_uint256(self.amount));
        out
    }
}

impl MessageBuilder for DepositBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body();
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_ETH_DEPOSIT,
                sender: self.from,
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
    use arbos::parse_l2::{parse_l2_transactions, ParsedTransaction};

    fn sample() -> DepositBuilder {
        DepositBuilder {
            from: address!("00000000000000000000000000000000000000aa"),
            to: address!("00000000000000000000000000000000000000bb"),
            amount: U256::from(1_000_000_000_000_000u64),
            l1_block_number: 100,
            timestamp: 1_700_000_000,
            request_seq: 7,
            base_fee_l1: 30_000_000_000,
        }
    }

    #[test]
    fn body_layout_matches_parser() {
        let msg = sample().build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body.len(), 52);
        let to = address!("00000000000000000000000000000000000000bb");
        assert_eq!(&body[..20], to.as_slice());
        assert_eq!(
            U256::from_be_slice(&body[20..52]),
            U256::from(1_000_000_000_000_000u64)
        );
    }

    #[test]
    fn json_round_trip_preserves_fields() {
        let msg = sample().build().unwrap();
        let json = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(json).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_ETH_DEPOSIT);
        assert_eq!(back.header.block_number, 100);
        assert_eq!(back.l2_msg, msg.l2_msg);
    }

    #[test]
    fn parses_into_eth_deposit() {
        let s = sample();
        let msg = s.build().unwrap();
        let parsed = round_trip(&msg);
        let txs = parse_l2_transactions(
            parsed.header.kind,
            parsed.header.poster,
            &parsed.l2_msg,
            parsed.header.request_id,
            parsed.header.l1_base_fee,
            42_161,
        )
        .unwrap();
        assert_eq!(txs.len(), 1);
        match &txs[0] {
            ParsedTransaction::EthDeposit {
                from, to, value, ..
            } => {
                assert_eq!(*from, s.from);
                assert_eq!(*to, s.to);
                assert_eq!(*value, s.amount);
            }
            other => panic!("unexpected parsed kind: {other:?}"),
        }
    }
}
