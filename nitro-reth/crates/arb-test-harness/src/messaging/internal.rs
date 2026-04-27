use alloy_primitives::{Address, U256};

use crate::messaging::{
    encoding::request_id_from_seq, kinds, L1Message, L1MessageHeader, MessageBuilder,
};

#[derive(Debug, Clone)]
pub enum InternalTxKind {
    StartBlock {
        l1_base_fee: U256,
        l1_block_number: u64,
        l2_block_number: u64,
        time_passed: u64,
    },
    BatchPostingReport {
        batch_timestamp: u64,
        batch_poster: Address,
        batch_number: u64,
        batch_data_gas: u64,
        l1_base_fee: U256,
    },
    BatchPostingReportV2 {
        batch_timestamp: u64,
        batch_poster: Address,
        batch_number: u64,
        batch_calldata_length: u64,
        batch_calldata_non_zeros: u64,
        batch_extra_gas: u64,
        l1_base_fee: U256,
    },
}

impl InternalTxKind {
    pub fn encode(&self) -> Vec<u8> {
        match self {
            InternalTxKind::StartBlock {
                l1_base_fee,
                l1_block_number,
                l2_block_number,
                time_passed,
            } => arbos::internal_tx::encode_start_block(
                *l1_base_fee,
                *l1_block_number,
                *l2_block_number,
                *time_passed,
            ),
            InternalTxKind::BatchPostingReport {
                batch_timestamp,
                batch_poster,
                batch_number,
                batch_data_gas,
                l1_base_fee,
            } => arbos::internal_tx::encode_batch_posting_report(
                *batch_timestamp,
                *batch_poster,
                *batch_number,
                *batch_data_gas,
                *l1_base_fee,
            ),
            InternalTxKind::BatchPostingReportV2 {
                batch_timestamp,
                batch_poster,
                batch_number,
                batch_calldata_length,
                batch_calldata_non_zeros,
                batch_extra_gas,
                l1_base_fee,
            } => arbos::internal_tx::encode_batch_posting_report_v2(
                *batch_timestamp,
                *batch_poster,
                *batch_number,
                *batch_calldata_length,
                *batch_calldata_non_zeros,
                *batch_extra_gas,
                *l1_base_fee,
            ),
        }
    }
}

#[derive(Debug, Clone)]
pub struct InternalTxBuilder {
    pub sender: Address,
    pub l1_block_number: u64,
    pub timestamp: u64,
    pub request_seq: u64,
    pub base_fee_l1: u64,
    pub kind: InternalTxKind,
}

impl InternalTxBuilder {
    pub fn payload(&self) -> Vec<u8> {
        self.kind.encode()
    }
}

impl MessageBuilder for InternalTxBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_INTERNAL_TX,
                sender: self.sender,
                block_number: self.l1_block_number,
                timestamp: self.timestamp,
                request_id: Some(request_id_from_seq(self.request_seq)),
                base_fee_l1: self.base_fee_l1,
            },
            l2_msg: String::new(),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::messaging::test_support::round_trip;
    use alloy_primitives::address;
    use arbos::internal_tx::{
        decode_start_block_data, INTERNAL_TX_BATCH_POSTING_REPORT_METHOD_ID,
        INTERNAL_TX_BATCH_POSTING_REPORT_V2_METHOD_ID, INTERNAL_TX_START_BLOCK_METHOD_ID,
    };

    fn sample(kind: InternalTxKind) -> InternalTxBuilder {
        InternalTxBuilder {
            sender: Address::ZERO,
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: 1,
            base_fee_l1: 0,
            kind,
        }
    }

    #[test]
    fn start_block_payload_round_trips() {
        let kind = InternalTxKind::StartBlock {
            l1_base_fee: U256::from(30_000_000_000u64),
            l1_block_number: 100,
            l2_block_number: 5,
            time_passed: 12,
        };
        let s = sample(kind);
        let payload = s.payload();
        assert_eq!(&payload[..4], &INTERNAL_TX_START_BLOCK_METHOD_ID);
        let decoded = decode_start_block_data(&payload).unwrap();
        assert_eq!(decoded.l1_base_fee, U256::from(30_000_000_000u64));
        assert_eq!(decoded.l1_block_number, 100);
        assert_eq!(decoded.l2_block_number, 5);
        assert_eq!(decoded.time_passed, 12);
    }

    #[test]
    fn batch_report_v1_method_selector() {
        let kind = InternalTxKind::BatchPostingReport {
            batch_timestamp: 1_700_000_000,
            batch_poster: address!("a4b000000000000000000073657175656e636572"),
            batch_number: 1,
            batch_data_gas: 50_000,
            l1_base_fee: U256::from(30_000_000_000u64),
        };
        let payload = sample(kind).payload();
        assert_eq!(&payload[..4], &INTERNAL_TX_BATCH_POSTING_REPORT_METHOD_ID);
        assert_eq!(payload.len(), 4 + 32 * 5);
    }

    #[test]
    fn batch_report_v2_method_selector() {
        let kind = InternalTxKind::BatchPostingReportV2 {
            batch_timestamp: 1_700_000_000,
            batch_poster: address!("a4b000000000000000000073657175656e636572"),
            batch_number: 2,
            batch_calldata_length: 1024,
            batch_calldata_non_zeros: 600,
            batch_extra_gas: 1234,
            l1_base_fee: U256::from(30_000_000_000u64),
        };
        let payload = sample(kind).payload();
        assert_eq!(
            &payload[..4],
            &INTERNAL_TX_BATCH_POSTING_REPORT_V2_METHOD_ID
        );
        assert_eq!(payload.len(), 4 + 32 * 7);
    }

    #[test]
    fn empty_body_emits_no_transactions() {
        let kind = InternalTxKind::StartBlock {
            l1_base_fee: U256::ZERO,
            l1_block_number: 0,
            l2_block_number: 0,
            time_passed: 0,
        };
        let msg = sample(kind).build().unwrap();
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
        let kind = InternalTxKind::StartBlock {
            l1_base_fee: U256::from(1u64),
            l1_block_number: 1,
            l2_block_number: 1,
            time_passed: 0,
        };
        let msg = sample(kind).build().unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_INTERNAL_TX);
        assert_eq!(back.l2_msg, "");
    }
}
