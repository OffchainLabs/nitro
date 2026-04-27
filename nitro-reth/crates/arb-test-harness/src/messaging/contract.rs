use alloy_primitives::{Address, Bytes, U256};

use crate::messaging::{
    encoding::{encode_address256, encode_uint256, request_id_from_seq},
    kinds, L1Message, L1MessageHeader, MessageBuilder,
};

#[derive(Debug, Clone)]
pub struct ContractTxBuilder {
    pub from: Address,
    pub gas_limit: u64,
    pub max_fee_per_gas: U256,
    pub to: Address,
    pub value: U256,
    pub data: Bytes,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub request_seq: u64,
    pub base_fee_l1: u64,
}

impl ContractTxBuilder {
    pub fn encode_body(&self) -> Vec<u8> {
        let mut out = Vec::with_capacity(1 + 32 * 4 + self.data.len());
        out.push(kinds::KIND_CONTRACT_TX);
        out.extend_from_slice(&encode_uint256(U256::from(self.gas_limit)));
        out.extend_from_slice(&encode_uint256(self.max_fee_per_gas));
        out.extend_from_slice(&encode_address256(self.to));
        out.extend_from_slice(&encode_uint256(self.value));
        out.extend_from_slice(&self.data);
        out
    }
}

impl MessageBuilder for ContractTxBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body();
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_DEPOSIT,
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

    fn sample(data: Bytes, to: Address) -> ContractTxBuilder {
        ContractTxBuilder {
            from: address!("00000000000000000000000000000000000000c1"),
            gas_limit: 250_000,
            max_fee_per_gas: U256::from(2_000_000_000u64),
            to,
            value: U256::from(7_777u64),
            data,
            l1_block_number: 60,
            timestamp: 1_700_000_500,
            request_seq: 9,
            base_fee_l1: 0,
        }
    }

    #[test]
    fn body_starts_with_sub_kind_byte() {
        let s = sample(Bytes::new(), Address::ZERO);
        let msg = s.build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body[0], kinds::KIND_CONTRACT_TX);
        assert_eq!(body.len(), 1 + 32 * 4);
    }

    #[test]
    fn parses_into_contract_tx() {
        let to = address!("00000000000000000000000000000000000000bb");
        let s = sample(Bytes::from(vec![0x10, 0x20]), to);
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
        match &txs[0] {
            ParsedTransaction::ContractTx {
                from,
                to: parsed_to,
                value,
                gas,
                gas_fee_cap,
                data,
                request_id,
            } => {
                assert_eq!(*from, s.from);
                assert_eq!(*parsed_to, Some(to));
                assert_eq!(*value, s.value);
                assert_eq!(*gas, s.gas_limit);
                assert_eq!(*gas_fee_cap, s.max_fee_per_gas);
                assert_eq!(data.as_slice(), &[0x10, 0x20]);
                assert_eq!(*request_id, request_id_from_seq(s.request_seq));
            }
            other => panic!("unexpected parsed kind: {other:?}"),
        }
    }

    #[test]
    fn json_round_trip() {
        let to = address!("00000000000000000000000000000000000000bb");
        let msg = sample(Bytes::new(), to).build().unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_DEPOSIT);
        assert_eq!(back.l2_msg, msg.l2_msg);
    }
}
