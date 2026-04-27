use alloy_primitives::{Address, Bytes, U256};

use crate::{
    error::HarnessError,
    messaging::{
        encoding::{encode_address256, encode_uint256, request_id_from_seq},
        kinds, L1Message, L1MessageHeader, MessageBuilder, MAX_L2_MESSAGE_SIZE,
    },
};

#[derive(Debug, Clone)]
pub struct RetryableBuilder {
    pub from: Address,
    pub to: Address,
    pub l2_call_value: U256,
    pub deposit: U256,
    pub max_submission_fee: U256,
    pub excess_fee_refund_address: Address,
    pub call_value_refund_address: Address,
    pub gas_limit: u64,
    pub max_fee_per_gas: U256,
    pub data: Bytes,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub request_seq: u64,
    pub base_fee_l1: u64,
}

impl RetryableBuilder {
    pub fn encode_body(&self) -> crate::Result<Vec<u8>> {
        if self.data.len() > MAX_L2_MESSAGE_SIZE {
            return Err(HarnessError::Invalid(format!(
                "retryable calldata length {} exceeds {} byte cap",
                self.data.len(),
                MAX_L2_MESSAGE_SIZE
            )));
        }
        let mut out = Vec::with_capacity(32 * 9 + self.data.len());
        out.extend_from_slice(&encode_address256(self.to));
        out.extend_from_slice(&encode_uint256(self.l2_call_value));
        out.extend_from_slice(&encode_uint256(self.deposit));
        out.extend_from_slice(&encode_uint256(self.max_submission_fee));
        out.extend_from_slice(&encode_address256(self.excess_fee_refund_address));
        out.extend_from_slice(&encode_address256(self.call_value_refund_address));
        out.extend_from_slice(&encode_uint256(U256::from(self.gas_limit)));
        out.extend_from_slice(&encode_uint256(self.max_fee_per_gas));
        out.extend_from_slice(&encode_uint256(U256::from(self.data.len() as u64)));
        out.extend_from_slice(&self.data);
        Ok(out)
    }
}

impl MessageBuilder for RetryableBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body()?;
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_RETRYABLE_TX,
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

    fn sample(data: Bytes) -> RetryableBuilder {
        RetryableBuilder {
            from: address!("00000000000000000000000000000000000000a1"),
            to: address!("00000000000000000000000000000000000000b1"),
            l2_call_value: U256::from(1_000_000_000_000u64),
            deposit: U256::from(2_000_000_000_000u64),
            max_submission_fee: U256::from(500_000_000_000u64),
            excess_fee_refund_address: address!("00000000000000000000000000000000000000a2"),
            call_value_refund_address: address!("00000000000000000000000000000000000000a3"),
            gas_limit: 200_000,
            max_fee_per_gas: U256::from(1_000_000_000u64),
            data,
            l1_block_number: 200,
            timestamp: 1_800_000_000,
            request_seq: 11,
            base_fee_l1: 30_000_000_000,
        }
    }

    #[test]
    fn body_layout_offsets() {
        let payload = Bytes::from(vec![0xde, 0xad, 0xbe, 0xef]);
        let s = sample(payload.clone());
        let msg = s.build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body.len(), 32 * 9 + payload.len());
        assert_eq!(
            Address::from_slice(&body[12..32]),
            address!("00000000000000000000000000000000000000b1")
        );
        assert_eq!(
            U256::from_be_slice(&body[32..64]),
            U256::from(1_000_000_000_000u64)
        );
        assert_eq!(
            U256::from_be_slice(&body[32 * 8..32 * 9]),
            U256::from(payload.len())
        );
        assert_eq!(&body[32 * 9..], payload.as_ref());
    }

    #[test]
    fn parses_into_submit_retryable() {
        let payload = Bytes::from(vec![0x01, 0x02, 0x03]);
        let s = sample(payload.clone());
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
            ParsedTransaction::SubmitRetryable {
                request_id,
                deposit,
                callvalue,
                gas_limit,
                max_submission_fee,
                from,
                to,
                fee_refund_addr,
                beneficiary,
                data,
                ..
            } => {
                assert_eq!(*from, s.from);
                assert_eq!(*to, Some(s.to));
                assert_eq!(*deposit, s.deposit);
                assert_eq!(*callvalue, s.l2_call_value);
                assert_eq!(*gas_limit, s.gas_limit);
                assert_eq!(*max_submission_fee, s.max_submission_fee);
                assert_eq!(*fee_refund_addr, s.excess_fee_refund_address);
                assert_eq!(*beneficiary, s.call_value_refund_address);
                assert_eq!(data.as_slice(), payload.as_ref());
                assert_eq!(*request_id, request_id_from_seq(11));
            }
            other => panic!("unexpected parsed kind: {other:?}"),
        }
    }

    #[test]
    fn create_retryable_uses_zero_to() {
        let mut s = sample(Bytes::new());
        s.to = Address::ZERO;
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
        match &txs[0] {
            ParsedTransaction::SubmitRetryable { to, .. } => assert!(to.is_none()),
            other => panic!("unexpected parsed kind: {other:?}"),
        }
    }

    #[test]
    fn rejects_oversized_calldata() {
        let mut s = sample(Bytes::new());
        s.data = Bytes::from(vec![0u8; MAX_L2_MESSAGE_SIZE + 1]);
        let err = s.build().unwrap_err();
        assert!(matches!(err, HarnessError::Invalid(_)));
    }

    #[test]
    fn json_round_trip() {
        let msg = sample(Bytes::from(vec![0xaa])).build().unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_RETRYABLE_TX);
        assert_eq!(back.l2_msg, msg.l2_msg);
    }
}
