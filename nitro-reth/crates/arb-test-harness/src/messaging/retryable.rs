use alloy_primitives::{Address, Bytes, B256, U256};

use crate::{
    error::HarnessError,
    messaging::{
        encoding::{encode_address256, encode_uint256, request_id_from_seq},
        kinds, L1Message, L1MessageHeader, MessageBuilder, MAX_L2_MESSAGE_SIZE,
    },
};

/// L1→L2 address alias offset applied to L1 senders when their messages are
/// posted into the rollup inbox by an EOA. Adding the offset (mod 2^160)
/// produces the address that ArbOS sees as the message poster.
pub const L1_TO_L2_ALIAS_OFFSET: Address = Address::new([
    0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x11, 0x11,
]);

/// Returns `addr + 0x1111000000000000000000000000000000001111 (mod 2^160)`.
pub fn apply_l1_to_l2_alias(addr: Address) -> Address {
    let lhs = U256::from_be_slice(addr.as_slice());
    let rhs = U256::from_be_slice(L1_TO_L2_ALIAS_OFFSET.as_slice());
    // Wrapping add over U256 then mask to 160 bits.
    let sum = lhs.wrapping_add(rhs);
    let mut bytes = sum.to_be_bytes::<32>();
    let lo20 = &mut bytes[12..];
    Address::from_slice(lo20)
}

/// Builds an L1 message of kind 9 (SubmitRetryable). The body layout matches
/// `ParseSubmitRetryableMessage` exactly:
///
/// ```text
/// to                         (32, address-as-32)
/// l2_call_value              (32, big-endian uint)
/// deposit_value              (32, big-endian uint)
/// max_submission_fee         (32, big-endian uint)
/// excess_fee_refund_address  (32, address-as-32)
/// call_value_refund_address  (32, address-as-32)
/// gas_limit                  (32, big-endian uint)
/// max_fee_per_gas            (32, big-endian uint)
/// data_len                   (32, big-endian uint)
/// data                       (variable)
/// ```
///
/// The L1 message header sender is the L1-aliased version of `l1_sender`.
#[derive(Debug, Clone)]
pub struct RetryableSubmitBuilder {
    /// L1-side sender address, pre-alias. Aliasing is applied internally.
    pub l1_sender: Address,
    /// Retry destination. `Address::ZERO` builds a CREATE retryable.
    pub to: Address,
    pub l2_call_value: U256,
    pub deposit_value: U256,
    pub max_submission_fee: U256,
    pub excess_fee_refund_address: Address,
    pub call_value_refund_address: Address,
    pub gas_limit: u64,
    pub max_fee_per_gas: U256,
    pub data: Bytes,
    pub l1_block_number: u64,
    pub timestamp: u64,
    /// If `None`, a placeholder is derived from the timestamp; SubmitRetryable
    /// requires a non-zero request id.
    pub request_id: Option<B256>,
}

impl RetryableSubmitBuilder {
    /// Returns the aliased L2-side sender address recorded in the header.
    pub fn aliased_sender(&self) -> Address {
        apply_l1_to_l2_alias(self.l1_sender)
    }

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
        out.extend_from_slice(&encode_uint256(self.deposit_value));
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

impl MessageBuilder for RetryableSubmitBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body()?;
        let request_id = self
            .request_id
            .unwrap_or_else(|| request_id_from_seq(self.timestamp));
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_SUBMIT_RETRYABLE,
                sender: self.aliased_sender(),
                block_number: self.l1_block_number,
                timestamp: self.timestamp,
                request_id: Some(request_id),
                base_fee_l1: 0,
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

    fn sample(data: Bytes) -> RetryableSubmitBuilder {
        RetryableSubmitBuilder {
            l1_sender: address!("00000000000000000000000000000000000000a1"),
            to: address!("00000000000000000000000000000000000000b1"),
            l2_call_value: U256::from(1_000_000_000_000u64),
            deposit_value: U256::from(2_000_000_000_000u64),
            max_submission_fee: U256::from(500_000_000_000u64),
            excess_fee_refund_address: address!("00000000000000000000000000000000000000a2"),
            call_value_refund_address: address!("00000000000000000000000000000000000000a3"),
            gas_limit: 200_000,
            max_fee_per_gas: U256::from(1_000_000_000u64),
            data,
            l1_block_number: 200,
            timestamp: 1_800_000_000,
            request_id: Some(request_id_from_seq(11)),
        }
    }

    #[test]
    fn alias_applies_offset() {
        let pre = address!("00000000000000000000000000000000000000a1");
        let post = apply_l1_to_l2_alias(pre);
        // 0xa1 + 0x1111 = 0x11b2 in the low 16 bits.
        assert_eq!(post, address!("11110000000000000000000000000000000011b2"));
    }

    #[test]
    fn alias_wraps_at_2_pow_160() {
        let pre = Address::from_slice(&[0xff; 20]);
        let post = apply_l1_to_l2_alias(pre);
        // (2^160 - 1) + 0x1111000000000000000000000000000000001111
        //   == 0x1111000000000000000000000000000000001110 (mod 2^160).
        assert_eq!(post, address!("1111000000000000000000000000000000001110"));
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
            U256::from_be_slice(&body[64..96]),
            U256::from(2_000_000_000_000u64)
        );
        assert_eq!(
            U256::from_be_slice(&body[96..128]),
            U256::from(500_000_000_000u64)
        );
        assert_eq!(
            Address::from_slice(&body[128 + 12..160]),
            address!("00000000000000000000000000000000000000a2")
        );
        assert_eq!(
            Address::from_slice(&body[160 + 12..192]),
            address!("00000000000000000000000000000000000000a3")
        );
        assert_eq!(U256::from_be_slice(&body[192..224]), U256::from(200_000u64));
        assert_eq!(
            U256::from_be_slice(&body[224..256]),
            U256::from(1_000_000_000u64)
        );
        assert_eq!(
            U256::from_be_slice(&body[256..288]),
            U256::from(payload.len())
        );
        assert_eq!(&body[288..], payload.as_ref());
    }

    #[test]
    fn round_trips_through_arbos_parser() {
        let payload = Bytes::from(vec![0x01, 0x02, 0x03]);
        let s = sample(payload.clone());
        let msg = s.build().unwrap();
        assert_eq!(msg.header.kind, kinds::KIND_SUBMIT_RETRYABLE);
        assert_eq!(msg.header.sender, s.aliased_sender());

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
                // The "from" (poster) seen by arbos is the aliased address.
                assert_eq!(*from, s.aliased_sender());
                assert_eq!(*to, Some(s.to));
                assert_eq!(*deposit, s.deposit_value);
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
        assert_eq!(back.header.kind, kinds::KIND_SUBMIT_RETRYABLE);
        assert_eq!(back.l2_msg, msg.l2_msg);
    }
}
