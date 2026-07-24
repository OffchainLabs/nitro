//! Parsing delayed inbox messages from a parent-chain block.
//!
//! The "from origin" variant emits no data in the log; the data lives in the
//! calling transaction's `sendL2MessageFromOrigin(bytes)` calldata, which we
//! decode with the `sol!`-generated call type.

use std::collections::{BTreeMap, BTreeSet};

use alloy_consensus::{Header, Transaction};
use alloy_primitives::{Address, B256, U256, keccak256};
use alloy_rpc_types_eth::Log;
use alloy_sol_types::{SolCall, SolEvent, sol};
use arbos::arbos_types::{L1IncomingMessage, L1IncomingMessageHeader};

use crate::{DelayedInboxMessage, LogsFetcher, MelError, MelResult, MelState, TxFetcher};

/// Length of a Solidity ABI function selector (the leading 4 bytes of calldata).
const SELECTOR_LEN: usize = 4;

sol! {
    #[allow(missing_docs)]
    #[derive(Debug)]
    event MessageDelivered(
        uint256 indexed messageIndex,
        bytes32 indexed beforeInboxAcc,
        address inbox,
        uint8 kind,
        address sender,
        bytes32 messageDataHash,
        uint256 baseFeeL1,
        uint64 timestamp
    );

    #[allow(missing_docs)]
    #[derive(Debug)]
    event InboxMessageDelivered(uint256 indexed messageNum, bytes data);

    #[allow(missing_docs)]
    #[derive(Debug)]
    event InboxMessageDeliveredFromOrigin(uint256 indexed messageNum);

    #[allow(missing_docs)]
    function sendL2MessageFromOrigin(bytes messageData) external returns (uint256);
}

/// Parses all delayed inbox messages observed in a single parent-chain block.
#[allow(dead_code)] // Wired into MEL block processing in a later change.
pub(crate) async fn parse_delayed_messages_from_block<L, T>(
    mel_state: &MelState,
    parent_chain_header: &Header,
    tx_fetcher: &T,
    logs_fetcher: &L,
) -> MelResult<Vec<DelayedInboxMessage>>
where
    L: LogsFetcher,
    T: TxFetcher,
    T::Transaction: Transaction,
{
    let logs = logs_fetcher.logs_for_block_hash(parent_chain_header.hash_slow())?;

    let relevant: Vec<&Log> = logs
        .iter()
        .filter(|l| l.inner.address == mel_state.delayed_message_posting_target_address)
        .collect();

    let (mut scaffolds, parsed_events) =
        delayed_message_scaffolds_from_logs(parent_chain_header, &relevant)?;

    // Collect the set of inbox addresses and message ids referenced by the
    // scaffolds, so we can find the matching inbox-message data logs.
    let mut inbox_addresses: BTreeSet<Address> = BTreeSet::new();
    let mut message_ids: BTreeSet<B256> = BTreeSet::new();
    for ev in &parsed_events {
        inbox_addresses.insert(ev.inbox);
        message_ids.insert(message_id(ev.messageIndex));
    }

    let mut message_data: BTreeMap<B256, Vec<u8>> = BTreeMap::new();
    for log in &logs {
        if !inbox_addresses.contains(&log.inner.address) {
            continue;
        }
        let topics = log.topics();
        let Some(topic0) = topics.first() else {
            continue;
        };
        let is_inbox_msg = *topic0 == InboxMessageDelivered::SIGNATURE_HASH
            || *topic0 == InboxMessageDeliveredFromOrigin::SIGNATURE_HASH;
        if !is_inbox_msg {
            continue;
        }
        // topic1 is the indexed message number; skip messages we don't care about.
        let Some(topic1) = topics.get(1) else {
            continue;
        };
        if !message_ids.contains(topic1) {
            continue;
        }
        let (msg_num, data) = parse_delayed_message(log, tx_fetcher).await?;
        message_data.insert(message_id(msg_num), data);
    }

    // Fill each scaffold's L2 message data, verifying it against the hash in the
    // corresponding `MessageDelivered` event.
    for (scaffold, ev) in scaffolds.iter_mut().zip(parsed_events.iter()) {
        let key = message_id(ev.messageIndex);
        let data = message_data
            .get(&key)
            .ok_or(MelError::MessageDataNotFound { message_index: key })?;
        if keccak256(data) != ev.messageDataHash {
            return Err(MelError::MessageDataHashMismatch { message_index: key });
        }
        scaffold.message.l2_msg = data.clone();
    }

    scaffolds.sort_by_key(|m| m.message.header.request_id);
    Ok(scaffolds)
}

fn delayed_message_scaffolds_from_logs(
    parent_chain_header: &Header,
    logs: &[&Log],
) -> MelResult<(Vec<DelayedInboxMessage>, Vec<MessageDelivered>)> {
    let header_hash = parent_chain_header.hash_slow();
    let mut scaffolds = Vec::with_capacity(logs.len());
    let mut events = Vec::with_capacity(logs.len());

    for log in logs {
        let Some(topic0) = log.topics().first() else {
            continue;
        };
        if *topic0 != MessageDelivered::SIGNATURE_HASH {
            continue;
        }
        let ev = MessageDelivered::decode_log(&log.inner)
            .map_err(|source| MelError::AbiDecode {
                event: "MessageDelivered",
                source,
            })?
            .data;

        let request_id = message_id(ev.messageIndex);
        scaffolds.push(DelayedInboxMessage {
            // Logs come from this parent-chain block; fall back to the header
            // when the rpc log omits block context.
            block_hash: log.block_hash.unwrap_or(header_hash),
            before_inbox_acc: ev.beforeInboxAcc,
            message: L1IncomingMessage {
                header: L1IncomingMessageHeader {
                    kind: ev.kind,
                    poster: ev.sender,
                    block_number: parent_chain_header.number,
                    timestamp: ev.timestamp,
                    request_id: Some(request_id),
                    l1_base_fee: Some(ev.baseFeeL1),
                },
                l2_msg: Vec::new(),
                legacy_batch_gas_cost: None,
                batch_data_stats: None,
            },
            parent_chain_block_number: log.block_number.unwrap_or(parent_chain_header.number),
        });
        events.push(ev);
    }

    Ok((scaffolds, events))
}

/// Extracts the message number and data bytes from a single inbox-message log.
async fn parse_delayed_message<T>(log: &Log, tx_fetcher: &T) -> MelResult<(U256, Vec<u8>)>
where
    T: TxFetcher,
    T::Transaction: Transaction,
{
    let topic0 = *log.topics().first().ok_or(MelError::UnexpectedLogType)?;

    if topic0 == InboxMessageDelivered::SIGNATURE_HASH {
        let ev = InboxMessageDelivered::decode_log(&log.inner)
            .map_err(|source| MelError::AbiDecode {
                event: "InboxMessageDelivered",
                source,
            })?
            .data;
        Ok((ev.messageNum, ev.data.to_vec()))
    } else if topic0 == InboxMessageDeliveredFromOrigin::SIGNATURE_HASH {
        let ev = InboxMessageDeliveredFromOrigin::decode_log(&log.inner)
            .map_err(|source| MelError::AbiDecode {
                event: "InboxMessageDeliveredFromOrigin",
                source,
            })?
            .data;

        let tx = tx_fetcher.transaction_by_log(log).await?;
        let calldata = tx.input();
        if calldata.len() < SELECTOR_LEN {
            return Err(MelError::TxDataTooShort);
        }
        let call = sendL2MessageFromOriginCall::abi_decode(calldata).map_err(|source| {
            MelError::AbiDecode {
                event: "sendL2MessageFromOrigin",
                source,
            }
        })?;
        Ok((ev.messageNum, call.messageData.to_vec()))
    } else {
        Err(MelError::UnexpectedLogType)
    }
}

/// Encodes a message index as a 32-byte big-endian hash (Go: `common.BigToHash`).
fn message_id(message_index: U256) -> B256 {
    B256::from(message_index.to_be_bytes::<32>())
}

#[cfg(test)]
mod tests {
    use alloy_consensus::TxLegacy;
    use alloy_primitives::Bytes;

    use super::*;
    use crate::test_utils::{MockLogs, MockTx, rpc_log};

    const MESSAGE_INDEX: u64 = 7;

    fn target_and_inbox() -> (Address, Address) {
        (Address::repeat_byte(0xDD), Address::repeat_byte(0xEE))
    }

    fn message_delivered_log(target: Address, inbox: Address, data_hash: B256) -> Log {
        let ev = MessageDelivered {
            messageIndex: U256::from(MESSAGE_INDEX),
            beforeInboxAcc: B256::repeat_byte(0x01),
            inbox,
            kind: 3,
            sender: Address::repeat_byte(0x55),
            messageDataHash: data_hash,
            baseFeeL1: U256::from(1000u64),
            timestamp: 42,
        };
        rpc_log(target, ev.encode_log_data())
    }

    fn inbox_message_log(inbox: Address, data: &[u8]) -> Log {
        let ev = InboxMessageDelivered {
            messageNum: U256::from(MESSAGE_INDEX),
            data: Bytes::from(data.to_vec()),
        };
        rpc_log(inbox, ev.encode_log_data())
    }

    fn state_with_target(target: Address) -> MelState {
        MelState {
            delayed_message_posting_target_address: target,
            ..Default::default()
        }
    }

    #[tokio::test]
    async fn reconstructs_delayed_message() -> MelResult<()> {
        let (target, inbox) = target_and_inbox();
        let data = b"hello-delayed";
        let logs = MockLogs {
            block_logs: vec![
                message_delivered_log(target, inbox, keccak256(data)),
                inbox_message_log(inbox, data),
            ],
            ..Default::default()
        };
        let out = parse_delayed_messages_from_block(
            &state_with_target(target),
            &Header::default(),
            &MockTx,
            &logs,
        )
        .await?;
        assert_eq!(out.len(), 1);
        assert_eq!(out[0].message.l2_msg, data);
        assert_eq!(out[0].message.header.kind, 3);
        assert_eq!(out[0].before_inbox_acc, B256::repeat_byte(0x01));
        assert_eq!(
            out[0].message.header.request_id,
            Some(message_id(U256::from(MESSAGE_INDEX)))
        );
        Ok(())
    }

    #[tokio::test]
    async fn rejects_mismatched_data_hash() {
        let (target, inbox) = target_and_inbox();
        let logs = MockLogs {
            block_logs: vec![
                // Advertised hash does not match the delivered data.
                message_delivered_log(target, inbox, B256::repeat_byte(0xAB)),
                inbox_message_log(inbox, b"hello-delayed"),
            ],
            ..Default::default()
        };
        let result = parse_delayed_messages_from_block(
            &state_with_target(target),
            &Header::default(),
            &MockTx,
            &logs,
        )
        .await;
        assert!(matches!(
            result,
            Err(MelError::MessageDataHashMismatch { .. })
        ));
    }

    #[tokio::test]
    async fn errors_when_message_data_missing() {
        let (target, inbox) = target_and_inbox();
        let logs = MockLogs {
            // MessageDelivered with no matching inbox-message data log.
            block_logs: vec![message_delivered_log(target, inbox, keccak256(b"x"))],
            ..Default::default()
        };
        let result = parse_delayed_messages_from_block(
            &state_with_target(target),
            &Header::default(),
            &MockTx,
            &logs,
        )
        .await;
        assert!(matches!(result, Err(MelError::MessageDataNotFound { .. })));
    }

    #[tokio::test]
    async fn ignores_logs_from_other_addresses() -> MelResult<()> {
        let (target, inbox) = target_and_inbox();
        // A MessageDelivered emitted by some other contract is not the configured
        // delayed-message posting target, so it is skipped and nothing is returned.
        let logs = MockLogs {
            block_logs: vec![message_delivered_log(
                Address::repeat_byte(0x99),
                inbox,
                keccak256(b"x"),
            )],
            ..Default::default()
        };
        let out = parse_delayed_messages_from_block(
            &state_with_target(target),
            &Header::default(),
            &MockTx,
            &logs,
        )
        .await?;
        assert!(out.is_empty());
        Ok(())
    }

    struct MockOriginTx {
        input: Vec<u8>,
    }

    impl TxFetcher for MockOriginTx {
        type Transaction = TxLegacy;
        fn transaction_by_log(&self, _log: &Log) -> MelResult<TxLegacy> {
            Ok(TxLegacy {
                input: self.input.clone().into(),
                ..Default::default()
            })
        }
    }

    fn from_origin_log(inbox: Address) -> Log {
        let ev = InboxMessageDeliveredFromOrigin {
            messageNum: U256::from(MESSAGE_INDEX),
        };
        rpc_log(inbox, ev.encode_log_data())
    }

    fn message_delivered_log_idx(
        target: Address,
        inbox: Address,
        index: u64,
        data_hash: B256,
    ) -> Log {
        let ev = MessageDelivered {
            messageIndex: U256::from(index),
            beforeInboxAcc: B256::repeat_byte(0x01),
            inbox,
            kind: 3,
            sender: Address::repeat_byte(0x55),
            messageDataHash: data_hash,
            baseFeeL1: U256::from(1000u64),
            timestamp: 42,
        };
        rpc_log(target, ev.encode_log_data())
    }

    fn inbox_message_log_idx(inbox: Address, index: u64, data: &[u8]) -> Log {
        let ev = InboxMessageDelivered {
            messageNum: U256::from(index),
            data: Bytes::from(data.to_vec()),
        };
        rpc_log(inbox, ev.encode_log_data())
    }

    #[test]
    fn reconstructs_from_origin_message() -> MelResult<()> {
        let (target, inbox) = target_and_inbox();
        let data = b"foobar";
        let call = sendL2MessageFromOriginCall {
            messageData: Bytes::from(data.to_vec()),
        };
        let tx = MockOriginTx {
            input: call.abi_encode(),
        };
        let logs = MockLogs {
            block_logs: vec![
                message_delivered_log(target, inbox, keccak256(data)),
                from_origin_log(inbox),
            ],
            ..Default::default()
        };
        let out = parse_delayed_messages_from_block(
            &state_with_target(target),
            &Header::default(),
            &tx,
            &logs,
        )?;
        assert_eq!(out.len(), 1);
        assert_eq!(out[0].message.l2_msg, data);
        Ok(())
    }

    #[test]
    fn rejects_from_origin_tx_too_short() {
        let (target, inbox) = target_and_inbox();
        let tx = MockOriginTx {
            input: vec![1u8, 2],
        };
        let logs = MockLogs {
            block_logs: vec![
                message_delivered_log(target, inbox, keccak256(b"foobar")),
                from_origin_log(inbox),
            ],
            ..Default::default()
        };
        let result = parse_delayed_messages_from_block(
            &state_with_target(target),
            &Header::default(),
            &tx,
            &logs,
        );
        assert!(matches!(result, Err(MelError::TxDataTooShort)));
    }

    #[test]
    fn sorts_by_request_id() -> MelResult<()> {
        let (target, inbox) = target_and_inbox();
        let logs = MockLogs {
            block_logs: vec![
                message_delivered_log_idx(target, inbox, 2, keccak256(b"two")),
                message_delivered_log_idx(target, inbox, 1, keccak256(b"one")),
                inbox_message_log_idx(inbox, 2, b"two"),
                inbox_message_log_idx(inbox, 1, b"one"),
            ],
            ..Default::default()
        };
        let out = parse_delayed_messages_from_block(
            &state_with_target(target),
            &Header::default(),
            &MockTx,
            &logs,
        )?;
        assert_eq!(out.len(), 2);
        assert_eq!(
            out[0].message.header.request_id,
            Some(message_id(U256::from(1u64)))
        );
        assert_eq!(
            out[1].message.header.request_id,
            Some(message_id(U256::from(2u64)))
        );
        Ok(())
    }

    #[test]
    fn scaffolds_from_empty_and_topicless_logs() -> MelResult<()> {
        let (scaffolds, events) = delayed_message_scaffolds_from_logs(&Header::default(), &[])?;
        assert!(scaffolds.is_empty());
        assert!(events.is_empty());

        let topicless = rpc_log(
            Address::repeat_byte(0xDD),
            alloy_primitives::LogData::default(),
        );
        let refs = [&topicless];
        let (scaffolds, events) = delayed_message_scaffolds_from_logs(&Header::default(), &refs)?;
        assert!(scaffolds.is_empty());
        assert!(events.is_empty());
        Ok(())
    }
}
