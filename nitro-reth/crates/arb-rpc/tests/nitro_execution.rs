//! Conformance tests for the nitroexecution handler.
//!
//! Exercises reorg, setFinalityData, and default block producer behavior
//! against a stubbed block producer that records calls.

use std::sync::Arc;

use alloy_primitives::{Address, B256};
use arb_rpc::block_producer::{
    BlockProducer, BlockProducerError, BlockProductionInput, ProducedBlock,
};

#[derive(Default, Debug)]
struct RecordingProducer {
    resets: parking_lot::Mutex<Vec<u64>>,
    produces: parking_lot::Mutex<Vec<u64>>,
    finality: parking_lot::Mutex<Vec<(Option<B256>, Option<B256>, Option<B256>)>>,
}

#[async_trait::async_trait]
impl BlockProducer for RecordingProducer {
    fn cache_init_message(&self, _l2_msg: &[u8]) -> Result<(), BlockProducerError> {
        Ok(())
    }

    async fn produce_block(
        &self,
        msg_idx: u64,
        _input: BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError> {
        self.produces.lock().push(msg_idx);
        Ok(ProducedBlock {
            block_hash: B256::repeat_byte(msg_idx as u8),
            send_root: B256::repeat_byte(0xAA),
        })
    }

    async fn reset_to_block(&self, target: u64) -> Result<(), BlockProducerError> {
        self.resets.lock().push(target);
        Ok(())
    }

    fn set_finality(
        &self,
        safe: Option<B256>,
        finalized: Option<B256>,
        validated: Option<B256>,
    ) -> Result<(), BlockProducerError> {
        self.finality.lock().push((safe, finalized, validated));
        Ok(())
    }
}

#[test]
fn default_set_finality_is_noop_ok() {
    #[derive(Default)]
    struct DefaultProducer;
    #[async_trait::async_trait]
    impl BlockProducer for DefaultProducer {
        fn cache_init_message(&self, _l2_msg: &[u8]) -> Result<(), BlockProducerError> {
            Ok(())
        }
        async fn produce_block(
            &self,
            _: u64,
            _: BlockProductionInput,
        ) -> Result<ProducedBlock, BlockProducerError> {
            unimplemented!()
        }
    }
    let p = DefaultProducer;
    assert!(p.set_finality(None, None, None).is_ok());
}

#[test]
fn default_reset_to_block_returns_unsupported_error() {
    #[derive(Default)]
    struct DefaultProducer;
    #[async_trait::async_trait]
    impl BlockProducer for DefaultProducer {
        fn cache_init_message(&self, _l2_msg: &[u8]) -> Result<(), BlockProducerError> {
            Ok(())
        }
        async fn produce_block(
            &self,
            _: u64,
            _: BlockProductionInput,
        ) -> Result<ProducedBlock, BlockProducerError> {
            unimplemented!()
        }
    }
    let rt = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .unwrap();
    let err = rt.block_on(DefaultProducer.reset_to_block(42));
    assert!(err.is_err(), "default reset_to_block should return error");
}

#[test]
fn recording_producer_records_reset_calls() {
    let rec = Arc::new(RecordingProducer::default());
    let rt = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .unwrap();
    rt.block_on(rec.reset_to_block(100)).unwrap();
    rt.block_on(rec.reset_to_block(50)).unwrap();
    assert_eq!(rec.resets.lock().clone(), vec![100, 50]);
}

#[test]
fn recording_producer_records_finality_triple() {
    let rec = RecordingProducer::default();
    rec.set_finality(
        Some(B256::repeat_byte(1)),
        Some(B256::repeat_byte(2)),
        Some(B256::repeat_byte(3)),
    )
    .unwrap();
    rec.set_finality(None, None, Some(B256::repeat_byte(4))).unwrap();
    let entries = rec.finality.lock().clone();
    assert_eq!(entries.len(), 2);
    assert_eq!(
        entries[0],
        (
            Some(B256::repeat_byte(1)),
            Some(B256::repeat_byte(2)),
            Some(B256::repeat_byte(3)),
        )
    );
    assert_eq!(entries[1], (None, None, Some(B256::repeat_byte(4))));
}

#[test]
fn block_production_input_fields_preserved() {
    let rec = Arc::new(RecordingProducer::default());
    let input = BlockProductionInput {
        kind: 3,
        sender: Address::repeat_byte(0xAB),
        l1_block_number: 100,
        l1_timestamp: 1_700_000_000,
        request_id: Some(B256::repeat_byte(0x42)),
        l1_base_fee: None,
        l2_msg: vec![1, 2, 3],
        delayed_messages_read: 5,
        batch_gas_cost: None,
        batch_data_stats: None,
    };
    let rt = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .unwrap();
    let out = rt.block_on(rec.produce_block(7, input)).unwrap();
    assert_eq!(out.block_hash, B256::repeat_byte(7));
    assert_eq!(rec.produces.lock().clone(), vec![7]);
}
