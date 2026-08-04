//! Integration tests for the `meldataprovider` surface: drive `MelApiHandler`
//! over a seeded `MockMelProvider` (the 9 primitives), exercising the shared
//! default-method logic (bounds, binary search, finalized, safe/finalized clamp)
//! and asserting the JSON wire shapes.

use std::{collections::HashMap, sync::Arc};

use alloy_primitives::{Address, B256, U256};
use arb_mel_types::{BatchMetadata, DelayedInboxMessage, MelState};
use arb_rpc::{
    L1BlockTag, MelApiHandler, MelApiServer, MelProvider, MelProviderError, MelProviderResult,
    mel::{
        RpcDelayedInboxMessage, RpcFinalizedDelayedResult, RpcFindInboxBatchResult,
        RpcSequencerMessageResult,
    },
};
use arbos::types::{L1IncomingMessage, L1IncomingMessageHeader};
use base64::Engine as _;

/// Seeded in-memory backing exercising the `MelProvider` default methods.
#[derive(Default)]
struct MockMelProvider {
    head: Option<MelState>,
    states: HashMap<u64, MelState>,
    delayed: HashMap<u64, DelayedInboxMessage>,
    batch_metas: HashMap<u64, BatchMetadata>,
    safe_block: u64,
    finalized_block: u64,
    seq_bytes: Option<(Vec<u8>, B256)>,
}

#[async_trait::async_trait]
impl MelProvider for MockMelProvider {
    async fn head_state(&self) -> MelProviderResult<MelState> {
        self.head
            .clone()
            .ok_or_else(|| MelProviderError::NotFound("head".into()))
    }
    async fn state(&self, block: u64) -> MelProviderResult<Option<MelState>> {
        Ok(self.states.get(&block).cloned())
    }
    async fn raw_delayed_message(&self, index: u64) -> MelProviderResult<DelayedInboxMessage> {
        self.delayed
            .get(&index)
            .cloned()
            .ok_or_else(|| MelProviderError::NotFound(format!("delayed {index}")))
    }
    async fn raw_batch_metadata(&self, seq: u64) -> MelProviderResult<BatchMetadata> {
        self.batch_metas
            .get(&seq)
            .cloned()
            .ok_or_else(|| MelProviderError::NotFound(format!("batch {seq}")))
    }
    async fn resolve_l1_block(&self, tag: L1BlockTag) -> MelProviderResult<u64> {
        Ok(match tag {
            L1BlockTag::Safe => self.safe_block,
            L1BlockTag::Finalized => self.finalized_block,
        })
    }
    async fn sequencer_message_bytes_for_parent_block(
        &self,
        _seq: u64,
        _parent: u64,
    ) -> MelProviderResult<(Vec<u8>, B256)> {
        self.seq_bytes
            .clone()
            .ok_or_else(|| MelProviderError::Backing("no seq bytes".into()))
    }
    async fn find_message_origin_mel_state(
        &self,
        _pos: u64,
    ) -> MelProviderResult<Option<MelState>> {
        Ok(self.head.clone())
    }
    async fn caught_up(&self) -> MelProviderResult<bool> {
        Ok(true)
    }
    async fn reorg_to(&self, _block: u64) -> MelProviderResult<()> {
        Ok(())
    }
}

fn mel_state(block: u64, batch_count: u64, delayed_seen: u64, msg_count: u64) -> MelState {
    MelState {
        parent_chain_block_number: block,
        batch_count,
        delayed_messages_seen: delayed_seen,
        msg_count,
        ..Default::default()
    }
}

fn delayed_msg(before_inbox_acc: B256, parent_block: u64) -> DelayedInboxMessage {
    DelayedInboxMessage {
        block_hash: B256::ZERO,
        before_inbox_acc,
        message: L1IncomingMessage {
            header: L1IncomingMessageHeader {
                kind: 3,
                poster: Address::ZERO,
                block_number: 0,
                timestamp: 0,
                request_id: Some(B256::ZERO),
                l1_base_fee: Some(U256::ZERO),
            },
            l2_msg: vec![].into(),
            legacy_batch_gas_cost: None,
            batch_data_stats: None,
        },
        parent_chain_block_number: parent_block,
    }
}

fn batch_meta(message_count: u64) -> BatchMetadata {
    BatchMetadata {
        accumulator: B256::ZERO,
        message_count,
        delayed_message_count: 0,
        parent_chain_block: 0,
    }
}

fn handler(m: MockMelProvider) -> MelApiHandler {
    MelApiHandler::new(Arc::new(m))
}

#[tokio::test]
async fn delayed_message_bounds() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 2, 0)),
        ..Default::default()
    };
    m.delayed.insert(0, delayed_msg(B256::ZERO, 0));
    let h = handler(m);
    assert!(h.get_delayed_message(0).await.is_ok());
    assert!(h.get_delayed_message(2).await.is_err()); // index == delayed_seen
}

#[tokio::test]
async fn batch_metadata_bounds() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 1, 0, 0)),
        ..Default::default()
    };
    m.batch_metas.insert(0, batch_meta(10));
    let h = handler(m);
    assert!(h.get_batch_metadata(0).await.is_ok());
    assert!(h.get_batch_metadata(1).await.is_err()); // seq == batch_count
}

#[tokio::test]
async fn find_inbox_batch_ladder() {
    // cumulative message counts: batch0->5, batch1->10, batch2->15
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 3, 0, 15)),
        ..Default::default()
    };
    m.batch_metas.insert(0, batch_meta(5));
    m.batch_metas.insert(1, batch_meta(10));
    m.batch_metas.insert(2, batch_meta(15));
    let h = handler(m);
    for (pos, want) in [(0, 0), (4, 0), (5, 1), (9, 1), (10, 2), (14, 2)] {
        let r = h.find_inbox_batch_containing_message(pos).await.unwrap();
        assert!(r.found, "pos {pos}");
        assert_eq!(r.seq_num, want, "pos {pos}");
    }
    for pos in [15, 100] {
        let r = h.find_inbox_batch_containing_message(pos).await.unwrap();
        assert!(!r.found, "pos {pos}");
    }
}

#[tokio::test]
async fn find_inbox_batch_empty() {
    let m = MockMelProvider {
        head: Some(mel_state(0, 0, 0, 0)),
        ..Default::default()
    };
    let r = handler(m)
        .find_inbox_batch_containing_message(0)
        .await
        .unwrap();
    assert!(!r.found);
}

#[tokio::test]
async fn finalized_delayed_success() {
    let acc = B256::repeat_byte(0xAA);
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 3, 0)),
        ..Default::default()
    };
    m.states.insert(100, mel_state(100, 0, 2, 0)); // finalized delayed count = 2
    m.delayed.insert(1, delayed_msg(acc, 7));
    let r = handler(m)
        .finalized_delayed_message_at_position(100, acc, 1)
        .await
        .unwrap();
    assert!(!r.not_yet_finalized);
    assert!(r.message.is_some());
    assert_eq!(r.parent_chain_block_number, 7);
}

#[tokio::test]
async fn finalized_delayed_not_yet_finalized_missing_state() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 3, 0)),
        ..Default::default()
    };
    m.delayed.insert(1, delayed_msg(B256::ZERO, 42));
    let r = handler(m)
        .finalized_delayed_message_at_position(999, B256::ZERO, 1)
        .await
        .unwrap();
    assert!(r.not_yet_finalized);
    assert!(r.message.is_none());
    assert_eq!(r.parent_chain_block_number, 42); // block number survives
}

#[tokio::test]
async fn finalized_delayed_pos_beyond_finalized_count() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 3, 0)),
        ..Default::default()
    };
    m.states.insert(100, mel_state(100, 0, 2, 0)); // finalized count = 2
    m.delayed.insert(2, delayed_msg(B256::ZERO, 9));
    let r = handler(m)
        .finalized_delayed_message_at_position(100, B256::ZERO, 2)
        .await
        .unwrap();
    assert!(r.not_yet_finalized); // pos 2 >= finalized count 2
    assert_eq!(r.parent_chain_block_number, 9);
}

#[tokio::test]
async fn finalized_delayed_accumulator_mismatch() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 3, 0)),
        ..Default::default()
    };
    m.states.insert(100, mel_state(100, 0, 2, 0));
    m.delayed.insert(1, delayed_msg(B256::repeat_byte(0xAA), 0));
    let err = handler(m)
        .finalized_delayed_message_at_position(100, B256::repeat_byte(0xBB), 1)
        .await
        .unwrap_err();
    assert!(
        err.message()
            .contains("delayed message accumulator mismatch")
    );
}

#[tokio::test]
async fn finalized_delayed_zero_accumulator_skips_check() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 3, 0)),
        ..Default::default()
    };
    m.states.insert(100, mel_state(100, 0, 2, 0));
    m.delayed.insert(1, delayed_msg(B256::repeat_byte(0xAA), 0)); // nonzero before_inbox_acc
    let r = handler(m)
        .finalized_delayed_message_at_position(100, B256::ZERO, 1) // zero acc -> no check
        .await
        .unwrap();
    assert!(!r.not_yet_finalized);
    assert!(r.message.is_some());
}

#[tokio::test]
async fn safe_finalized_msg_count_clamps_to_head() {
    let mut m = MockMelProvider {
        head: Some(mel_state(50, 0, 0, 500)),
        ..Default::default()
    };
    m.states.insert(50, mel_state(50, 0, 0, 500));
    m.states.insert(30, mel_state(30, 0, 0, 300));
    m.safe_block = 100; // > head 50 -> clamp to 50
    m.finalized_block = 30; // < head 50 -> use 30
    let h = handler(m);
    assert_eq!(h.get_safe_msg_count().await.unwrap(), 500);
    assert_eq!(h.get_finalized_msg_count().await.unwrap(), 300);
}

#[tokio::test]
async fn delayed_acc_equals_after_inbox_acc() {
    let msg = delayed_msg(B256::repeat_byte(0xAB), 0);
    let expected = msg.after_inbox_acc();
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 1, 0)),
        ..Default::default()
    };
    m.delayed.insert(0, msg);
    assert_eq!(handler(m).get_delayed_acc(0).await.unwrap(), expected);
}

#[tokio::test]
async fn find_parent_chain_block_containing_delayed_is_unimplemented() {
    let m = MockMelProvider {
        head: Some(mel_state(0, 0, 0, 0)),
        ..Default::default()
    };
    let err = handler(m)
        .find_parent_chain_block_containing_delayed(0)
        .await
        .unwrap_err();
    assert!(
        err.message()
            .contains("FindParentChainBlockContainingDelayed is not implemented by MEL")
    );
}

#[tokio::test]
async fn head_derived_counts_and_origin() {
    let m = MockMelProvider {
        head: Some(mel_state(0, 3, 5, 20)),
        ..Default::default()
    };
    let h = handler(m);
    assert_eq!(h.get_batch_count().await.unwrap(), 3);
    assert_eq!(h.get_delayed_count().await.unwrap(), 5);
    assert_eq!(h.get_msg_count().await.unwrap(), 20);
    assert!(h.find_message_origin_mel_state(0).await.unwrap().is_some());
}

#[tokio::test]
async fn batch_accessors() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 1, 0, 0)),
        ..Default::default()
    };
    m.batch_metas.insert(
        0,
        BatchMetadata {
            accumulator: B256::repeat_byte(0x22),
            message_count: 7,
            delayed_message_count: 3,
            parent_chain_block: 99,
        },
    );
    let h = handler(m);
    assert_eq!(h.get_batch_acc(0).await.unwrap(), B256::repeat_byte(0x22));
    assert_eq!(h.get_batch_message_count(0).await.unwrap(), 7);
    assert_eq!(h.get_batch_parent_chain_block(0).await.unwrap(), 99);
}

#[tokio::test]
async fn get_delayed_message_returns_converted() {
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 1, 0)),
        ..Default::default()
    };
    m.delayed.insert(0, delayed_msg(B256::repeat_byte(0x11), 5));
    let r = handler(m).get_delayed_message(0).await.unwrap().unwrap();
    assert_eq!(r.before_inbox_acc, B256::repeat_byte(0x11));
    assert_eq!(r.parent_chain_block_number, 5);
}

#[tokio::test]
async fn delayed_message_bytes_is_base64_of_serialize() {
    let mut d = delayed_msg(B256::ZERO, 0);
    d.message.l2_msg = vec![1, 2, 3, 4].into();
    let expected = base64::engine::general_purpose::STANDARD.encode(d.message.serialize());
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 0, 1, 0)),
        ..Default::default()
    };
    m.delayed.insert(0, d);
    assert_eq!(
        handler(m).get_delayed_message_bytes(0).await.unwrap(),
        expected
    );
}

#[tokio::test]
async fn sequencer_message_bytes_base64_and_block_hash() {
    let block_hash = B256::repeat_byte(0x99);
    let mut m = MockMelProvider {
        head: Some(mel_state(0, 1, 0, 0)), // batch_count 1 -> seq 0 in bounds
        ..Default::default()
    };
    m.batch_metas.insert(0, batch_meta(5));
    m.seq_bytes = Some((vec![9, 8, 7], block_hash));
    let h = handler(m);
    let r = h.get_sequencer_message_bytes(0).await.unwrap();
    assert_eq!(
        r.data,
        base64::engine::general_purpose::STANDARD.encode([9, 8, 7])
    );
    assert_eq!(r.block_hash, block_hash);
    // for-parent-block variant is the primitive path (no batch-count bounds)
    let r2 = h
        .get_sequencer_message_bytes_for_parent_block(0, 42)
        .await
        .unwrap();
    assert_eq!(r2.block_hash, block_hash);
}

#[tokio::test]
async fn get_state_present_and_absent() {
    let mut m = MockMelProvider::default();
    m.states.insert(7, mel_state(7, 0, 0, 0));
    let h = handler(m);
    assert!(h.get_state(7).await.unwrap().is_some());
    assert!(h.get_state(999).await.unwrap().is_none()); // absent -> JSON null
}

#[tokio::test]
async fn head_state_absent_errors() {
    let h = handler(MockMelProvider::default());
    assert!(h.get_head_state().await.is_err());
}

#[tokio::test]
async fn sync_progress_uses_head_counts() {
    let m = MockMelProvider {
        head: Some(mel_state(0, 4, 0, 40)),
        ..Default::default()
    };
    let p = handler(m).get_sync_progress().await.unwrap();
    assert_eq!(p.batch_seen, 4);
    assert_eq!(p.batch_processed, 4);
    assert_eq!(p.msg_count, 40);
}

#[tokio::test]
async fn lifecycle_passthroughs() {
    let h = handler(MockMelProvider {
        head: Some(mel_state(0, 0, 0, 0)),
        ..Default::default()
    });
    assert!(h.caught_up().await.unwrap());
    assert!(h.supports_pushing_finality_data().await.unwrap());
    assert!(h.reorg_to(5).await.is_ok());
}

#[test]
fn mel_state_json_is_pascal_case() {
    let v = serde_json::to_value(mel_state(1, 2, 3, 4)).unwrap();
    let o = v.as_object().unwrap();
    assert!(o.contains_key("ParentChainId"));
    assert!(o.contains_key("ParentChainPreviousBlockHash"));
    assert!(o.contains_key("DelayedMessageInboxAcc"));
}

#[test]
fn find_inbox_batch_result_json_is_camel_case() {
    let v = serde_json::to_value(RpcFindInboxBatchResult {
        seq_num: 5,
        found: true,
    })
    .unwrap();
    assert_eq!(v["seqNum"], 5);
    assert_eq!(v["found"], true);
}

#[test]
fn sequencer_message_result_json_is_camel_case() {
    let v = serde_json::to_value(RpcSequencerMessageResult {
        data: "AQID".into(),
        block_hash: B256::ZERO,
    })
    .unwrap();
    assert_eq!(v["data"], "AQID");
    assert!(v["blockHash"].is_string());
}

#[test]
fn finalized_delayed_result_json_is_camel_case() {
    let v = serde_json::to_value(RpcFinalizedDelayedResult {
        message: None,
        after_inbox_acc: B256::ZERO,
        parent_chain_block_number: 3,
        not_yet_finalized: true,
    })
    .unwrap();
    assert!(v.get("afterInboxAcc").is_some());
    assert_eq!(v["parentChainBlockNumber"], 3);
    assert_eq!(v["notYetFinalized"], true);
    assert!(v["message"].is_null());
}

#[test]
fn delayed_inbox_message_json_pascal_with_base64_l2msg() {
    let mut d = delayed_msg(B256::ZERO, 0);
    d.message.l2_msg = vec![1, 2, 3].into();
    let v = serde_json::to_value(RpcDelayedInboxMessage::from(&d)).unwrap();
    let o = v.as_object().unwrap();
    for k in [
        "BlockHash",
        "BeforeInboxAcc",
        "Message",
        "ParentChainBlockNumber",
    ] {
        assert!(o.contains_key(k), "missing {k}");
    }
    assert_eq!(
        v["Message"]["l2Msg"],
        base64::engine::general_purpose::STANDARD.encode([1, 2, 3])
    );
}
