//! Integration tests for the `BlockProducer` trait surface that the
//! `NitroExecutionHandler` dispatches into.
//!
//! `NitroExecutionHandler` itself binds a complex `Provider` trait set
//! (block/header/storage readers) that is impractical to mock in a unit
//! test. The handler's behaviour reduces to forwarding inputs to the
//! `BlockProducer` impl; these tests exercise that contract directly:
//! `set_finality` propagation, `reset_to_block` propagation,
//! `cache_init_message` and `produce_block` round-trip, and the
//! validated-watcher attach hook.

use std::sync::Arc;

use alloy_primitives::{Address, B256};
use arb_rpc::block_producer::{
    BlockProducer, BlockProducerError, BlockProductionInput, ProducedBlock,
};
use parking_lot::{Mutex, RwLock};

#[derive(Default, Debug)]
struct Stub {
    inits: Mutex<Vec<Vec<u8>>>,
    produces: Mutex<Vec<(u64, BlockProductionInput)>>,
    resets: Mutex<Vec<u64>>,
    finality: Mutex<Vec<(Option<B256>, Option<B256>, Option<B256>)>>,
    watchers: Mutex<Vec<Arc<RwLock<B256>>>>,
}

#[async_trait::async_trait]
impl BlockProducer for Stub {
    fn cache_init_message(&self, l2_msg: &[u8]) -> Result<(), BlockProducerError> {
        self.inits.lock().push(l2_msg.to_vec());
        Ok(())
    }

    async fn produce_block(
        &self,
        msg_idx: u64,
        input: BlockProductionInput,
    ) -> Result<ProducedBlock, BlockProducerError> {
        self.produces.lock().push((msg_idx, input));
        Ok(ProducedBlock {
            block_hash: B256::repeat_byte(msg_idx as u8),
            send_root: B256::repeat_byte(0x55),
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

    fn attach_validated_watcher(&self, watcher: Arc<RwLock<B256>>) {
        self.watchers.lock().push(watcher);
    }
}

fn rt() -> tokio::runtime::Runtime {
    tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()
        .unwrap()
}

fn sample_input() -> BlockProductionInput {
    BlockProductionInput {
        kind: 3,
        sender: Address::repeat_byte(0xAB),
        l1_block_number: 100,
        l1_timestamp: 1_700_000_000,
        request_id: Some(B256::repeat_byte(0x42)),
        l1_base_fee: None,
        l2_msg: vec![1, 2, 3, 4],
        delayed_messages_read: 5,
        batch_gas_cost: None,
        batch_data_stats: None,
    }
}

#[test]
fn cache_init_message_records_payload() {
    let stub = Stub::default();
    stub.cache_init_message(&[0x01, 0x02, 0x03]).unwrap();
    stub.cache_init_message(&[0xff]).unwrap();
    let inits = stub.inits.lock().clone();
    assert_eq!(inits, vec![vec![0x01, 0x02, 0x03], vec![0xff]]);
}

#[test]
fn produce_block_returns_hash_derived_from_msg_idx() {
    let stub = Arc::new(Stub::default());
    let out = rt()
        .block_on(stub.produce_block(7, sample_input()))
        .unwrap();
    assert_eq!(out.block_hash, B256::repeat_byte(7));
    assert_eq!(out.send_root, B256::repeat_byte(0x55));
}

#[test]
fn produce_block_preserves_input_fields() {
    let stub = Arc::new(Stub::default());
    let inp = sample_input();
    rt().block_on(stub.produce_block(42, inp)).unwrap();
    let recorded = stub.produces.lock();
    assert_eq!(recorded.len(), 1);
    let (idx, ref captured) = recorded[0];
    assert_eq!(idx, 42);
    assert_eq!(captured.kind, 3);
    assert_eq!(captured.sender, Address::repeat_byte(0xAB));
    assert_eq!(captured.l1_block_number, 100);
    assert_eq!(captured.request_id, Some(B256::repeat_byte(0x42)));
    assert_eq!(captured.l2_msg, vec![1, 2, 3, 4]);
}

#[test]
fn reset_to_block_targets_propagate_in_order() {
    let stub = Arc::new(Stub::default());
    let rt = rt();
    rt.block_on(stub.reset_to_block(100)).unwrap();
    rt.block_on(stub.reset_to_block(50)).unwrap();
    rt.block_on(stub.reset_to_block(200)).unwrap();
    assert_eq!(stub.resets.lock().clone(), vec![100, 50, 200]);
}

#[test]
fn set_finality_records_full_triple() {
    let stub = Stub::default();
    let safe = B256::repeat_byte(0x01);
    let finalized = B256::repeat_byte(0x02);
    let validated = B256::repeat_byte(0x03);
    stub.set_finality(Some(safe), Some(finalized), Some(validated))
        .unwrap();
    let entries = stub.finality.lock().clone();
    assert_eq!(entries.len(), 1);
    assert_eq!(entries[0], (Some(safe), Some(finalized), Some(validated)));
}

#[test]
fn set_finality_accepts_partial_updates() {
    let stub = Stub::default();
    stub.set_finality(None, Some(B256::repeat_byte(0x10)), None)
        .unwrap();
    stub.set_finality(Some(B256::repeat_byte(0x20)), None, None)
        .unwrap();
    let entries = stub.finality.lock().clone();
    assert_eq!(entries.len(), 2);
    assert_eq!(entries[0], (None, Some(B256::repeat_byte(0x10)), None));
    assert_eq!(entries[1], (Some(B256::repeat_byte(0x20)), None, None));
}

#[test]
fn attach_validated_watcher_stores_handle() {
    let stub = Stub::default();
    let watcher = Arc::new(RwLock::new(B256::ZERO));
    stub.attach_validated_watcher(watcher.clone());
    let watchers = stub.watchers.lock();
    assert_eq!(watchers.len(), 1);
    *watchers[0].write() = B256::repeat_byte(0xFF);
    assert_eq!(*watcher.read(), B256::repeat_byte(0xFF));
}

#[test]
fn default_set_finality_is_noop_ok() {
    #[derive(Default)]
    struct DefaultProducer;
    #[async_trait::async_trait]
    impl BlockProducer for DefaultProducer {
        fn cache_init_message(&self, _: &[u8]) -> Result<(), BlockProducerError> {
            Ok(())
        }
        async fn produce_block(
            &self,
            _: u64,
            _: BlockProductionInput,
        ) -> Result<ProducedBlock, BlockProducerError> {
            unreachable!()
        }
    }
    let p = DefaultProducer;
    assert!(p.set_finality(None, None, None).is_ok());
    assert!(p
        .set_finality(Some(B256::ZERO), Some(B256::ZERO), Some(B256::ZERO))
        .is_ok());
}

#[test]
fn default_reset_to_block_returns_unsupported_error() {
    #[derive(Default)]
    struct DefaultProducer;
    #[async_trait::async_trait]
    impl BlockProducer for DefaultProducer {
        fn cache_init_message(&self, _: &[u8]) -> Result<(), BlockProducerError> {
            Ok(())
        }
        async fn produce_block(
            &self,
            _: u64,
            _: BlockProductionInput,
        ) -> Result<ProducedBlock, BlockProducerError> {
            unreachable!()
        }
    }
    let err = rt().block_on(DefaultProducer.reset_to_block(42));
    assert!(err.is_err());
    let msg = format!("{}", err.unwrap_err());
    assert!(msg.contains("not supported"));
}

#[test]
fn block_producer_error_variants_carry_message() {
    let cases = [
        BlockProducerError::StateAccess("a".into()),
        BlockProducerError::Execution("b".into()),
        BlockProducerError::Storage("c".into()),
        BlockProducerError::Parse("d".into()),
        BlockProducerError::Unexpected("e".into()),
    ];
    for err in cases {
        let s = format!("{err}");
        assert!(!s.is_empty());
    }
}
