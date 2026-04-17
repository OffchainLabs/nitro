use alloy_primitives::{Address, B256, U256};
use arbos::{
    arbos_types::{
        get_data_stats, legacy_cost_for_stats, parse_batch_posting_report_fields,
        parse_incoming_l1_message, parse_init_message, DEFAULT_INITIAL_L1_BASE_FEE,
        L1_MESSAGE_TYPE_BATCH_POSTING_REPORT, L1_MESSAGE_TYPE_INITIALIZE,
        L1_MESSAGE_TYPE_L2_MESSAGE,
    },
    parse_l2::{
        parse_l2_transactions, ParsedTransaction, L2_MESSAGE_KIND_BATCH, L2_MESSAGE_KIND_HEARTBEAT,
        L2_MESSAGE_KIND_NON_MUTATING_CALL, L2_MESSAGE_KIND_UNSIGNED_USER_TX,
    },
};

const CHAIN_ID: u64 = 42_161;

fn noop_poster() -> Address {
    Address::ZERO
}

// ======================================================================
// parse_init_message
// ======================================================================

#[test]
fn init_msg_len_32_is_chain_id_only_default_base_fee() {
    let mut data = [0u8; 32];
    data[31] = 42;
    let r = parse_init_message(&data).expect("parse");
    assert_eq!(r.chain_id, U256::from(42u64));
    assert_eq!(
        r.initial_l1_base_fee,
        U256::from(DEFAULT_INITIAL_L1_BASE_FEE)
    );
    assert!(r.serialized_chain_config.is_empty());
}

#[test]
fn init_msg_empty_errors() {
    assert!(parse_init_message(&[]).is_err());
}

#[test]
fn init_msg_len_between_33_inclusive_tests_version_zero() {
    let mut data = vec![0u8; 33];
    data[31] = 1;
    data[32] = 0;
    let r = parse_init_message(&data).expect("parse");
    assert_eq!(r.chain_id, U256::from(1u64));
    assert_eq!(
        r.initial_l1_base_fee,
        U256::from(DEFAULT_INITIAL_L1_BASE_FEE)
    );
    assert!(r.serialized_chain_config.is_empty());
}

#[test]
fn init_msg_version_0_captures_chain_config_tail() {
    let mut data = vec![0u8; 33 + 5];
    data[31] = 1;
    data[32] = 0;
    data[33..].copy_from_slice(b"HELLO");
    let r = parse_init_message(&data).expect("parse");
    assert_eq!(r.serialized_chain_config, b"HELLO");
}

#[test]
fn init_msg_version_1_reads_l1_base_fee_then_chain_config() {
    let mut data = vec![0u8; 33 + 32 + 3];
    data[31] = 7;
    data[32] = 1;
    let mut fee_bytes = [0u8; 32];
    fee_bytes[31] = 77;
    data[33..65].copy_from_slice(&fee_bytes);
    data[65..].copy_from_slice(b"CFG");
    let r = parse_init_message(&data).expect("parse");
    assert_eq!(r.chain_id, U256::from(7u64));
    assert_eq!(r.initial_l1_base_fee, U256::from(77u64));
    assert_eq!(r.serialized_chain_config, b"CFG");
}

#[test]
fn init_msg_unsupported_version_errors() {
    let mut data = vec![0u8; 33];
    data[32] = 99;
    assert!(parse_init_message(&data).is_err());
}

#[test]
fn init_msg_len_between_1_and_31_errors() {
    assert!(parse_init_message(&[0u8; 1]).is_err());
    assert!(parse_init_message(&[0u8; 20]).is_err());
    assert!(parse_init_message(&[0u8; 31]).is_err());
}

// ======================================================================
// parse_l2_transactions
// ======================================================================

#[test]
fn end_of_block_l1_message_returns_empty() {
    let r = parse_l2_transactions(6, noop_poster(), &[], None, None, CHAIN_ID).expect("parse");
    assert!(r.is_empty());
}

#[test]
fn initialize_l1_message_returns_empty() {
    let r = parse_l2_transactions(
        L1_MESSAGE_TYPE_INITIALIZE,
        noop_poster(),
        &[],
        None,
        None,
        CHAIN_ID,
    )
    .expect("parse");
    assert!(r.is_empty());
}

#[test]
fn rollup_event_l1_message_returns_empty() {
    let r = parse_l2_transactions(8, noop_poster(), &[0x11], None, None, CHAIN_ID).expect("parse");
    assert!(r.is_empty());
}

#[test]
fn unknown_l1_message_kind_returns_empty() {
    let r =
        parse_l2_transactions(0xAB, noop_poster(), &[0x11], None, None, CHAIN_ID).expect("parse");
    assert!(r.is_empty());
}

#[test]
fn batch_for_gas_estimation_errors() {
    let r = parse_l2_transactions(10, noop_poster(), &[], None, None, CHAIN_ID);
    assert!(r.is_err());
}

#[test]
fn eth_deposit_requires_request_id() {
    let data = vec![0u8; 52];
    let err = parse_l2_transactions(12, noop_poster(), &data, None, None, CHAIN_ID);
    assert!(err.is_err());
}

#[test]
fn l2_funded_by_l1_requires_request_id() {
    let data = vec![0u8; 32];
    let err = parse_l2_transactions(7, noop_poster(), &data, None, None, CHAIN_ID);
    assert!(err.is_err());
}

#[test]
fn submit_retryable_requires_request_id() {
    let data = vec![0u8; 32];
    let err = parse_l2_transactions(9, noop_poster(), &data, None, None, CHAIN_ID);
    assert!(err.is_err());
}

// ======================================================================
// L2 message inner kinds
// ======================================================================

#[test]
fn l2_message_empty_errors() {
    let err = parse_l2_transactions(
        L1_MESSAGE_TYPE_L2_MESSAGE,
        noop_poster(),
        &[],
        None,
        None,
        CHAIN_ID,
    );
    assert!(err.is_err());
}

#[test]
fn l2_message_heartbeat_yields_empty() {
    let data = vec![L2_MESSAGE_KIND_HEARTBEAT];
    let r = parse_l2_transactions(
        L1_MESSAGE_TYPE_L2_MESSAGE,
        noop_poster(),
        &data,
        None,
        None,
        CHAIN_ID,
    )
    .expect("parse");
    assert!(r.is_empty());
}

#[test]
fn l2_message_non_mutating_call_yields_empty() {
    let data = vec![L2_MESSAGE_KIND_NON_MUTATING_CALL];
    let r = parse_l2_transactions(
        L1_MESSAGE_TYPE_L2_MESSAGE,
        noop_poster(),
        &data,
        None,
        None,
        CHAIN_ID,
    )
    .expect("parse");
    assert!(r.is_empty());
}

#[test]
fn l2_message_signed_compressed_tx_errors() {
    let data = vec![7u8];
    let err = parse_l2_transactions(
        L1_MESSAGE_TYPE_L2_MESSAGE,
        noop_poster(),
        &data,
        None,
        None,
        CHAIN_ID,
    );
    assert!(err.is_err());
}

#[test]
fn l2_message_unknown_inner_kind_yields_empty() {
    let data = vec![0xFE];
    let r = parse_l2_transactions(
        L1_MESSAGE_TYPE_L2_MESSAGE,
        noop_poster(),
        &data,
        None,
        None,
        CHAIN_ID,
    )
    .expect("parse");
    assert!(r.is_empty());
}

#[test]
fn l2_message_unsigned_tx_parses_all_fields() {
    let mut payload = vec![L2_MESSAGE_KIND_UNSIGNED_USER_TX];
    payload.extend_from_slice(&[0u8; 32]);
    payload.extend_from_slice(&[0u8; 32]);
    payload.extend_from_slice(&[0u8; 32]);
    payload.extend_from_slice(&[0u8; 32]);
    payload.extend_from_slice(&[0u8; 32]);
    let r = parse_l2_transactions(
        L1_MESSAGE_TYPE_L2_MESSAGE,
        noop_poster(),
        &payload,
        Some(B256::repeat_byte(0x33)),
        None,
        CHAIN_ID,
    )
    .expect("parse");
    assert_eq!(r.len(), 1);
    assert!(matches!(r[0], ParsedTransaction::UnsignedUserTx { .. }));
}

#[test]
fn l2_batch_at_depth_zero_accepts() {
    let inner = vec![L2_MESSAGE_KIND_HEARTBEAT];
    let mut inner_encoded = rlp_bytes(&inner);
    let mut batch = vec![L2_MESSAGE_KIND_BATCH];
    batch.append(&mut inner_encoded);
    let r = parse_l2_transactions(
        L1_MESSAGE_TYPE_L2_MESSAGE,
        noop_poster(),
        &batch,
        None,
        None,
        CHAIN_ID,
    )
    .expect("parse");
    assert!(r.is_empty());
}

fn rlp_bytes(payload: &[u8]) -> Vec<u8> {
    let len = payload.len();
    if len < 56 {
        let mut out = vec![0x80 + len as u8];
        out.extend_from_slice(payload);
        out
    } else {
        let mut be = Vec::new();
        let mut x = len;
        while x > 0 {
            be.push((x & 0xFF) as u8);
            x >>= 8;
        }
        be.reverse();
        let mut out = vec![0xb7 + be.len() as u8];
        out.extend_from_slice(&be);
        out.extend_from_slice(payload);
        out
    }
}

// ======================================================================
// parse_incoming_l1_message
// ======================================================================

#[test]
fn incoming_l1_message_empty_errors() {
    assert!(parse_incoming_l1_message(&[]).is_err());
}

#[test]
fn incoming_l1_message_roundtrip_via_serialize() {
    let mut msg_bytes = Vec::new();
    msg_bytes.push(L1_MESSAGE_TYPE_L2_MESSAGE);
    msg_bytes.extend_from_slice(&[0u8; 32]);
    msg_bytes.extend_from_slice(&42u64.to_be_bytes());
    msg_bytes.extend_from_slice(&1_700_000_000u64.to_be_bytes());
    msg_bytes.extend_from_slice(&[0u8; 32]);
    msg_bytes.extend_from_slice(&[0u8; 32]);
    msg_bytes.extend_from_slice(&[0xAB, 0xCD]);
    let parsed = parse_incoming_l1_message(&msg_bytes).expect("parse");
    assert_eq!(parsed.header.kind, L1_MESSAGE_TYPE_L2_MESSAGE);
    assert_eq!(parsed.header.block_number, 42);
    assert_eq!(parsed.header.timestamp, 1_700_000_000);
    assert!(parsed.header.request_id.is_none());
    assert!(parsed.header.l1_base_fee.is_none());
    assert_eq!(parsed.l2_msg, vec![0xAB, 0xCD]);
    let serialized = parsed.serialize();
    assert_eq!(serialized, msg_bytes);
}

#[test]
fn seq_num_extracts_last_8_bytes_of_request_id() {
    use arbos::arbos_types::L1IncomingMessageHeader;
    let mut rid = [0u8; 32];
    rid[24..32].copy_from_slice(&42u64.to_be_bytes());
    let h = L1IncomingMessageHeader {
        kind: 0,
        poster: Address::ZERO,
        block_number: 0,
        timestamp: 0,
        request_id: Some(B256::from(rid)),
        l1_base_fee: None,
    };
    assert_eq!(h.seq_num(), Some(42));
}

// ======================================================================
// data stats + cost
// ======================================================================

#[test]
fn data_stats_counts_zeros_and_nonzeros_correctly() {
    let d = [0u8, 1, 0, 2, 0, 3, 0];
    let s = get_data_stats(&d);
    assert_eq!(s.length, 7);
    assert_eq!(s.non_zeros, 3);
}

#[test]
fn data_stats_empty() {
    let s = get_data_stats(&[]);
    assert_eq!(s.length, 0);
    assert_eq!(s.non_zeros, 0);
}

#[test]
#[allow(clippy::identity_op)]
fn legacy_cost_empty_is_just_overhead() {
    let c = legacy_cost_for_stats(&get_data_stats(&[]));
    assert_eq!(c, 30 + 0 + 2 * 20_000);
}

#[test]
fn legacy_cost_scales_with_nonzero_bytes_at_16_gas_each() {
    let s = get_data_stats(&[1, 2, 3]);
    let expected = 3 * 16 + 30 + ((3u64).div_ceil(32) * 6) + 2 * 20_000;
    assert_eq!(legacy_cost_for_stats(&s), expected);
}

#[test]
fn legacy_cost_zero_bytes_are_4_gas_each() {
    let s = get_data_stats(&[0u8; 10]);
    let expected = 10 * 4 + 30 + ((10u64).div_ceil(32) * 6) + 2 * 20_000;
    assert_eq!(legacy_cost_for_stats(&s), expected);
}

// ======================================================================
// parse_batch_posting_report_fields
// ======================================================================

#[test]
fn batch_posting_report_parses_all_fields() {
    let mut data = Vec::new();
    let mut ts = [0u8; 32];
    ts[31] = 100;
    data.extend_from_slice(&ts);
    data.extend_from_slice(&Address::repeat_byte(0xAB).into_array());
    data.extend_from_slice(&[0x77u8; 32]);
    let mut batch_num = [0u8; 32];
    batch_num[31] = 7;
    data.extend_from_slice(&batch_num);
    data.extend_from_slice(&[0u8; 32]);
    data.extend_from_slice(&[0u8; 32]);
    data.extend_from_slice(&[0u8; 32]);
    let f = parse_batch_posting_report_fields(&data).expect("parse");
    assert_eq!(f.batch_timestamp, 100);
    assert_eq!(f.batch_poster, Address::repeat_byte(0xAB));
    assert_eq!(f.data_hash, B256::repeat_byte(0x77));
    assert_eq!(f.batch_number, 7);
}

#[test]
fn batch_posting_report_truncated_errors() {
    let data = vec![0u8; 32];
    let err = parse_batch_posting_report_fields(&data);
    assert!(err.is_err());
}

#[test]
fn past_batches_for_nonreport_msg_is_empty() {
    use arbos::arbos_types::{L1IncomingMessage, L1IncomingMessageHeader};
    let msg = L1IncomingMessage {
        header: L1IncomingMessageHeader {
            kind: L1_MESSAGE_TYPE_L2_MESSAGE,
            poster: Address::ZERO,
            block_number: 0,
            timestamp: 0,
            request_id: None,
            l1_base_fee: None,
        },
        l2_msg: vec![],
        batch_gas_left: None,
    };
    assert!(msg.past_batches_required().unwrap().is_empty());
}

#[test]
fn past_batches_for_report_msg_has_number() {
    use arbos::arbos_types::{L1IncomingMessage, L1IncomingMessageHeader};
    let mut data = Vec::new();
    let mut ts = [0u8; 32];
    ts[31] = 1;
    data.extend_from_slice(&ts);
    data.extend_from_slice(&Address::ZERO.into_array());
    data.extend_from_slice(&[0u8; 32]);
    let mut batch_num = [0u8; 32];
    batch_num[31] = 11;
    data.extend_from_slice(&batch_num);
    data.extend_from_slice(&[0u8; 32]);
    data.extend_from_slice(&[0u8; 32]);
    data.extend_from_slice(&[0u8; 32]);
    let msg = L1IncomingMessage {
        header: L1IncomingMessageHeader {
            kind: L1_MESSAGE_TYPE_BATCH_POSTING_REPORT,
            poster: Address::ZERO,
            block_number: 0,
            timestamp: 0,
            request_id: None,
            l1_base_fee: None,
        },
        l2_msg: data,
        batch_gas_left: None,
    };
    assert_eq!(msg.past_batches_required().unwrap(), vec![11]);
}
