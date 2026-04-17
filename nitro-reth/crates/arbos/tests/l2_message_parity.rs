use alloy_primitives::Address;
use arbos::parse_l2::parse_l2_transactions;

const KIND_L2_MESSAGE: u8 = 3;
const KIND_L2_MESSAGE_BATCH_INNER: u8 = 3;

fn rlp_string_prefix(len: usize) -> Vec<u8> {
    if len < 56 {
        vec![0x80 + len as u8]
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
        out
    }
}

fn rlp_encode(payload: &[u8]) -> Vec<u8> {
    let mut out = rlp_string_prefix(payload.len());
    out.extend_from_slice(payload);
    out
}

fn build_batch_with_inner(inner_segments: Vec<Vec<u8>>) -> Vec<u8> {
    let mut payload = vec![KIND_L2_MESSAGE_BATCH_INNER];
    for seg in inner_segments {
        payload.extend_from_slice(&rlp_encode(&seg));
    }
    payload
}

/// At depth 0 wrapping a non-batch payload, parser should accept it.
#[test]
fn batch_at_depth_zero_accepts_inner_kind() {
    let inner_kind_payload = vec![6u8];
    let batch = build_batch_with_inner(vec![inner_kind_payload]);
    let result = parse_l2_transactions(KIND_L2_MESSAGE, Address::ZERO, &batch, None, None, 42_161);
    assert!(result.is_ok(), "kind=3 with inner kind=6 should parse");
}

/// Empty L2_MESSAGE payload: Nitro reads the kind byte first; if input is
/// empty, the read fails and Nitro returns an error. Ours returns Ok(vec![]).
#[test]
fn empty_l2_message_returns_err_like_nitro() {
    let res = parse_l2_transactions(KIND_L2_MESSAGE, Address::ZERO, &[], None, None, 42_161);
    assert!(
        res.is_err(),
        "Nitro errors on empty L2 message data; arbreth returns Ok([])"
    );
}

/// `parseEthDepositMessage`: Nitro errors when `header.RequestId == nil`.
/// We must not silently substitute zero.
#[test]
fn eth_deposit_without_request_id_returns_err() {
    let mut data = Vec::new();
    data.extend_from_slice(&[0u8; 20]);
    data.extend_from_slice(&[0u8; 32]);
    let res = parse_l2_transactions(12, Address::ZERO, &data, None, None, 42_161);
    assert!(
        res.is_err(),
        "Nitro errors on EthDeposit without request_id; got {res:?}"
    );
}

/// `parseSubmitRetryableMessage`: Nitro errors when `header.RequestId == nil`.
#[test]
fn submit_retryable_without_request_id_returns_err() {
    let mut data = Vec::new();
    for _ in 0..9 {
        data.extend_from_slice(&[0u8; 32]);
    }
    data.extend_from_slice(&[0u8; 32]);
    let res = parse_l2_transactions(9, Address::ZERO, &data, None, None, 42_161);
    assert!(
        res.is_err(),
        "Nitro errors on SubmitRetryable without request_id; got {res:?}"
    );
}

/// `L2FundedByL1`: Nitro errors when `header.RequestId == nil`.
#[test]
fn l2_funded_by_l1_without_request_id_returns_err() {
    let mut data = vec![0u8];
    data.extend_from_slice(&[0u8; 32 * 5]);
    let res = parse_l2_transactions(7, Address::ZERO, &data, None, None, 42_161);
    assert!(
        res.is_err(),
        "Nitro errors on L2FundedByL1 without request_id; got {res:?}"
    );
}
