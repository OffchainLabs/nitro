use alloy_eips::eip2718::{Decodable2718, Typed2718};
use alloy_primitives::{keccak256, Address, Bytes, B256, U256};
use arb_primitives::{
    signed_tx::ArbTransactionSigned,
    tx_types::{ArbContractTx, ArbDepositTx, ArbSubmitRetryableTx, ArbUnsignedTx},
};
use std::io::{self, Cursor, Read};

use crate::{
    arbos_types::{
        L1_MESSAGE_TYPE_BATCH_FOR_GAS_ESTIMATION, L1_MESSAGE_TYPE_BATCH_POSTING_REPORT,
        L1_MESSAGE_TYPE_END_OF_BLOCK, L1_MESSAGE_TYPE_ETH_DEPOSIT, L1_MESSAGE_TYPE_INITIALIZE,
        L1_MESSAGE_TYPE_L2_FUNDED_BY_L1, L1_MESSAGE_TYPE_L2_MESSAGE, L1_MESSAGE_TYPE_ROLLUP_EVENT,
        L1_MESSAGE_TYPE_SUBMIT_RETRYABLE,
    },
    util::{
        address_from_256_from_reader, address_from_reader, bytestring_from_reader,
        hash_from_reader, uint256_from_reader, uint64_from_reader,
    },
};

/// L2 message kind constants.
pub const L2_MESSAGE_KIND_UNSIGNED_USER_TX: u8 = 0;
pub const L2_MESSAGE_KIND_CONTRACT_TX: u8 = 1;
pub const L2_MESSAGE_KIND_NON_MUTATING_CALL: u8 = 2;
pub const L2_MESSAGE_KIND_BATCH: u8 = 3;
pub const L2_MESSAGE_KIND_SIGNED_TX: u8 = 4;
pub const L2_MESSAGE_KIND_HEARTBEAT: u8 = 6;
pub const L2_MESSAGE_KIND_SIGNED_COMPRESSED_TX: u8 = 7;

/// The ArbOS version at which heartbeat messages were disabled.
pub const HEARTBEATS_DISABLED_AT: u64 = 6;

/// Maximum size of an L2 message segment (256 KB).
pub const MAX_L2_MESSAGE_SIZE: usize = 256 * 1024;

/// Represents a parsed L2 transaction from an L1 message.
#[derive(Debug, Clone)]
pub enum ParsedTransaction {
    /// A signed Ethereum transaction (RLP-encoded).
    Signed(Vec<u8>),
    /// An unsigned user transaction (Arbitrum-specific).
    UnsignedUserTx {
        from: Address,
        to: Option<Address>,
        value: U256,
        gas: u64,
        gas_fee_cap: U256,
        nonce: u64,
        data: Vec<u8>,
    },
    /// A contract transaction (L1→L2 call).
    ContractTx {
        from: Address,
        to: Option<Address>,
        value: U256,
        gas: u64,
        gas_fee_cap: U256,
        data: Vec<u8>,
        request_id: B256,
    },
    /// An ETH deposit from L1.
    EthDeposit {
        from: Address,
        to: Address,
        value: U256,
        request_id: B256,
    },
    /// A submit retryable transaction.
    SubmitRetryable {
        request_id: B256,
        l1_base_fee: U256,
        deposit: U256,
        callvalue: U256,
        gas_feature_cap: U256,
        gas_limit: u64,
        max_submission_fee: U256,
        from: Address,
        to: Option<Address>,
        fee_refund_addr: Address,
        beneficiary: Address,
        data: Vec<u8>,
    },
    /// A batch posting report (internal tx).
    BatchPostingReport {
        batch_timestamp: u64,
        batch_poster: Address,
        data_hash: B256,
        batch_number: u64,
        l1_base_fee_estimate: U256,
        extra_gas: u64,
    },
    /// An internal start-block transaction.
    InternalStartBlock {
        l1_block_number: u64,
        l1_timestamp: u64,
    },
}

/// Parse L2 transactions from an L1 incoming message.
pub fn parse_l2_transactions(
    kind: u8,
    poster: Address,
    l2_msg: &[u8],
    request_id: Option<B256>,
    l1_base_fee: Option<U256>,
    chain_id: u64,
) -> Result<Vec<ParsedTransaction>, io::Error> {
    match kind {
        L1_MESSAGE_TYPE_L2_MESSAGE => parse_l2_message(l2_msg, poster, request_id, 0, chain_id),
        L1_MESSAGE_TYPE_END_OF_BLOCK => Ok(vec![]),
        L1_MESSAGE_TYPE_L2_FUNDED_BY_L1 => {
            let request_id = request_id.ok_or_else(|| {
                io::Error::new(
                    io::ErrorKind::InvalidData,
                    "cannot issue L2 funded by L1 tx without L1 request id",
                )
            })?;
            parse_l2_funded_by_l1(l2_msg, poster, request_id)
        }
        L1_MESSAGE_TYPE_SUBMIT_RETRYABLE => {
            let request_id = request_id.ok_or_else(|| {
                io::Error::new(
                    io::ErrorKind::InvalidData,
                    "cannot issue submit retryable tx without L1 request id",
                )
            })?;
            let l1_base_fee = l1_base_fee.unwrap_or(U256::ZERO);
            parse_submit_retryable_message(l2_msg, poster, request_id, l1_base_fee)
        }
        L1_MESSAGE_TYPE_ETH_DEPOSIT => {
            let request_id = request_id.ok_or_else(|| {
                io::Error::new(
                    io::ErrorKind::InvalidData,
                    "cannot issue deposit tx without L1 request id",
                )
            })?;
            parse_eth_deposit_message(l2_msg, poster, request_id)
        }
        L1_MESSAGE_TYPE_BATCH_POSTING_REPORT => {
            let request_id = request_id.unwrap_or(B256::ZERO);
            parse_batch_posting_report(l2_msg, poster, request_id)
        }
        L1_MESSAGE_TYPE_BATCH_FOR_GAS_ESTIMATION => Err(io::Error::new(
            io::ErrorKind::InvalidData,
            "L1 message type BatchForGasEstimation is unimplemented",
        )),
        L1_MESSAGE_TYPE_INITIALIZE | L1_MESSAGE_TYPE_ROLLUP_EVENT => Ok(vec![]),
        _ => Ok(vec![]),
    }
}

/// Batch-nesting limit matching Nitro (`depth >= 16` → error).
const MAX_L2_MESSAGE_BATCH_DEPTH: u32 = 16;

#[allow(clippy::only_used_in_recursion)]
fn parse_l2_message(
    data: &[u8],
    poster: Address,
    request_id: Option<B256>,
    depth: u32,
    chain_id: u64,
) -> Result<Vec<ParsedTransaction>, io::Error> {
    if data.is_empty() {
        return Err(io::Error::new(
            io::ErrorKind::UnexpectedEof,
            "L2 message is empty (missing kind byte)",
        ));
    }

    let kind = data[0];
    let payload = &data[1..];

    match kind {
        L2_MESSAGE_KIND_SIGNED_COMPRESSED_TX => Err(io::Error::new(
            io::ErrorKind::InvalidData,
            "L2 message kind SignedCompressedTx is unimplemented",
        )),
        L2_MESSAGE_KIND_SIGNED_TX => {
            // Reject Arbitrum internal types and blob txs. Chain ID is not
            // checked here — legacy txs with `v = 27/28` (no EIP-155 chain
            // ID) are valid (e.g. deterministic deploy txs).
            match ArbTransactionSigned::decode_2718(&mut &payload[..]) {
                Ok(tx) => {
                    let ty = tx.ty();
                    if ty >= 0x64 || ty == 3 {
                        return Err(io::Error::new(
                            io::ErrorKind::InvalidData,
                            format!("unsupported tx type: {ty}"),
                        ));
                    }
                    Ok(vec![ParsedTransaction::Signed(payload.to_vec())])
                }
                Err(_) => Err(io::Error::new(
                    io::ErrorKind::InvalidData,
                    "failed to decode signed transaction",
                )),
            }
        }
        L2_MESSAGE_KIND_UNSIGNED_USER_TX => {
            let tx = parse_unsigned_tx(payload, poster, request_id, kind)?;
            Ok(vec![tx])
        }
        L2_MESSAGE_KIND_CONTRACT_TX => {
            let tx = parse_unsigned_tx(payload, poster, request_id, kind)?;
            Ok(vec![tx])
        }
        L2_MESSAGE_KIND_BATCH => {
            if depth >= MAX_L2_MESSAGE_BATCH_DEPTH {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidData,
                    "L2 message batches have a max depth of 16",
                ));
            }
            let mut reader = Cursor::new(payload);
            let mut txs = Vec::new();
            let mut index: u64 = 0;
            while let Ok(segment) = bytestring_from_reader(&mut reader, MAX_L2_MESSAGE_SIZE as u64)
            {
                if segment.len() > MAX_L2_MESSAGE_SIZE {
                    break;
                }
                let sub_request_id = request_id.map(|parent_id| {
                    let mut preimage = [0u8; 64];
                    preimage[..32].copy_from_slice(parent_id.as_slice());
                    preimage[32..].copy_from_slice(&U256::from(index).to_be_bytes::<32>());
                    B256::from(keccak256(preimage))
                });
                index += 1;
                let mut sub_txs =
                    parse_l2_message(&segment, poster, sub_request_id, depth + 1, chain_id)?;
                txs.append(&mut sub_txs);
            }
            Ok(txs)
        }
        L2_MESSAGE_KIND_HEARTBEAT => Ok(vec![]),
        L2_MESSAGE_KIND_NON_MUTATING_CALL => Ok(vec![]),
        _ => Ok(vec![]),
    }
}

/// Parse an unsigned tx or contract tx from the binary format.
///
/// Field format (all 32-byte big-endian):
///   gasLimit: Hash (32 bytes) → u64
///   maxFeePerGas: Hash (32 bytes) → U256
///   nonce: Hash (32 bytes) → u64 (only for UnsignedUserTx kind)
///   to: AddressFrom256 (32 bytes) → Address
///   value: Hash (32 bytes) → U256
///   calldata: remaining bytes (ReadAll)
fn parse_unsigned_tx(
    data: &[u8],
    poster: Address,
    request_id: Option<B256>,
    kind: u8,
) -> Result<ParsedTransaction, io::Error> {
    let mut reader = Cursor::new(data);

    let gas_limit = uint256_from_reader(&mut reader)?;
    let gas_limit: u64 = gas_limit.try_into().map_err(|_| {
        io::Error::new(
            io::ErrorKind::InvalidData,
            "unsigned user tx gas limit >= 2^64",
        )
    })?;

    let max_fee_per_gas = uint256_from_reader(&mut reader)?;

    let nonce = if kind == L2_MESSAGE_KIND_UNSIGNED_USER_TX {
        let nonce_u256 = uint256_from_reader(&mut reader)?;
        let n: u64 = nonce_u256.try_into().map_err(|_| {
            io::Error::new(io::ErrorKind::InvalidData, "unsigned user tx nonce >= 2^64")
        })?;
        n
    } else {
        0
    };

    let to = address_from_256_from_reader(&mut reader)?;
    let destination = if to == Address::ZERO { None } else { Some(to) };

    let value = uint256_from_reader(&mut reader)?;

    let mut calldata = Vec::new();
    reader.read_to_end(&mut calldata)?;

    match kind {
        L2_MESSAGE_KIND_UNSIGNED_USER_TX => Ok(ParsedTransaction::UnsignedUserTx {
            from: poster,
            to: destination,
            value,
            gas: gas_limit,
            gas_fee_cap: max_fee_per_gas,
            nonce,
            data: calldata,
        }),
        L2_MESSAGE_KIND_CONTRACT_TX => {
            let req_id = request_id.ok_or_else(|| {
                io::Error::new(
                    io::ErrorKind::InvalidData,
                    "cannot issue contract tx without L1 request id",
                )
            })?;
            Ok(ParsedTransaction::ContractTx {
                from: poster,
                to: destination,
                value,
                gas: gas_limit,
                gas_fee_cap: max_fee_per_gas,
                data: calldata,
                request_id: req_id,
            })
        }
        _ => Err(io::Error::new(
            io::ErrorKind::InvalidData,
            "invalid L2 tx type in parseUnsignedTx",
        )),
    }
}

fn parse_l2_funded_by_l1(
    data: &[u8],
    poster: Address,
    request_id: B256,
) -> Result<Vec<ParsedTransaction>, io::Error> {
    if data.is_empty() {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            "L2FundedByL1 message has no data",
        ));
    }

    let kind = data[0];

    // Derive sub-request IDs: keccak256(requestId ++ U256(0)) and keccak256(requestId ++ U256(1))
    let mut deposit_preimage = [0u8; 64];
    deposit_preimage[..32].copy_from_slice(request_id.as_slice());
    // U256(0) is already zeroed
    let deposit_request_id = B256::from(keccak256(deposit_preimage));

    let mut unsigned_preimage = [0u8; 64];
    unsigned_preimage[..32].copy_from_slice(request_id.as_slice());
    unsigned_preimage[63] = 1; // U256(1) in big-endian
    let unsigned_request_id = B256::from(keccak256(unsigned_preimage));

    let tx = parse_unsigned_tx(&data[1..], poster, Some(unsigned_request_id), kind)?;

    // Extract value from the parsed tx for the deposit.
    let tx_value = match &tx {
        ParsedTransaction::UnsignedUserTx { value, .. } => *value,
        ParsedTransaction::ContractTx { value, .. } => *value,
        _ => U256::ZERO,
    };

    // L2FundedByL1 deposit: `from` is zero and `to` is the poster.
    let deposit = ParsedTransaction::EthDeposit {
        from: Address::ZERO,
        to: poster,
        value: tx_value,
        request_id: deposit_request_id,
    };

    Ok(vec![deposit, tx])
}

fn parse_eth_deposit_message(
    data: &[u8],
    poster: Address,
    request_id: B256,
) -> Result<Vec<ParsedTransaction>, io::Error> {
    let mut reader = Cursor::new(data);
    let to = address_from_reader(&mut reader)?;
    let value = uint256_from_reader(&mut reader)?;
    Ok(vec![ParsedTransaction::EthDeposit {
        from: poster,
        to,
        value,
        request_id,
    }])
}

fn parse_submit_retryable_message(
    data: &[u8],
    poster: Address,
    request_id: B256,
    l1_base_fee: U256,
) -> Result<Vec<ParsedTransaction>, io::Error> {
    let mut reader = Cursor::new(data);

    // Field order matches parseSubmitRetryableMessage exactly.
    let retry_to = address_from_256_from_reader(&mut reader)?;
    let callvalue = uint256_from_reader(&mut reader)?;
    let deposit = uint256_from_reader(&mut reader)?;
    let max_submission_fee = uint256_from_reader(&mut reader)?;
    let fee_refund_addr = address_from_256_from_reader(&mut reader)?;
    let beneficiary = address_from_256_from_reader(&mut reader)?;
    let gas_limit_u256 = uint256_from_reader(&mut reader)?;
    let gas_limit = gas_limit_u256
        .try_into()
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "gas limit too large"))?;
    let gas_feature_cap = uint256_from_reader(&mut reader)?;

    // Data length is encoded as a 32-byte hash, then raw bytes follow.
    // Cap the declared length at MAX_L2_MESSAGE_SIZE to prevent an
    // attacker from triggering a huge allocation (DoS).
    let data_length_hash = hash_from_reader(&mut reader)?;
    let data_length: usize = U256::from_be_bytes(data_length_hash.0)
        .try_into()
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "data length too large"))?;
    if data_length > MAX_L2_MESSAGE_SIZE {
        return Err(io::Error::new(
            io::ErrorKind::InvalidData,
            format!("data length {data_length} exceeds MAX_L2_MESSAGE_SIZE {MAX_L2_MESSAGE_SIZE}"),
        ));
    }
    let mut calldata = vec![0u8; data_length];
    if data_length > 0 {
        io::Read::read_exact(&mut reader, &mut calldata)?;
    }

    let to = if retry_to == Address::ZERO {
        None
    } else {
        Some(retry_to)
    };

    Ok(vec![ParsedTransaction::SubmitRetryable {
        request_id,
        l1_base_fee,
        deposit,
        callvalue,
        gas_feature_cap,
        gas_limit,
        max_submission_fee,
        from: poster,
        to,
        fee_refund_addr,
        beneficiary,
        data: calldata,
    }])
}

fn parse_batch_posting_report(
    data: &[u8],
    _poster: Address,
    _request_id: B256,
) -> Result<Vec<ParsedTransaction>, io::Error> {
    let mut reader = Cursor::new(data);

    // All fields use 32-byte Hash format except batchPosterAddr (20 bytes)
    // and extraGas (8-byte uint64, optional).
    let batch_timestamp_u256 = uint256_from_reader(&mut reader)?;
    let batch_timestamp: u64 = batch_timestamp_u256
        .try_into()
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "batch timestamp too large"))?;

    let batch_poster = address_from_reader(&mut reader)?;

    let data_hash = hash_from_reader(&mut reader)?;

    let batch_number_u256 = uint256_from_reader(&mut reader)?;
    let batch_number: u64 = batch_number_u256
        .try_into()
        .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "batch number too large"))?;

    let l1_base_fee_estimate = uint256_from_reader(&mut reader)?;

    // extraGas is optional — defaults to 0 on EOF.
    let extra_gas = match uint64_from_reader(&mut reader) {
        Ok(v) => v,
        Err(e) if e.kind() == io::ErrorKind::UnexpectedEof => 0,
        Err(e) => return Err(e),
    };

    Ok(vec![ParsedTransaction::BatchPostingReport {
        batch_timestamp,
        batch_poster,
        data_hash,
        batch_number,
        l1_base_fee_estimate,
        extra_gas,
    }])
}

// =====================================================================
// Conversion to ArbTransactionSigned
// =====================================================================

/// Convert a `ParsedTransaction` into an `ArbTransactionSigned`.
///
/// The `chain_id` is needed for constructing Arbitrum-specific tx envelopes.
/// Returns `None` for batch posting reports and internal start-block txs that
/// are constructed separately by the internal tx module.
pub fn parsed_tx_to_signed(
    parsed: &ParsedTransaction,
    chain_id: u64,
) -> Option<ArbTransactionSigned> {
    use arb_primitives::signed_tx::ArbTypedTransaction;

    let chain_id_u256 = U256::from(chain_id);

    let tx = match parsed {
        ParsedTransaction::Signed(rlp_bytes) => {
            // Standard signed Ethereum tx — decode via Decodable2718.
            use alloy_eips::Decodable2718;
            return ArbTransactionSigned::decode_2718(&mut rlp_bytes.as_slice()).ok();
        }
        ParsedTransaction::UnsignedUserTx {
            from,
            to,
            value,
            gas,
            gas_fee_cap,
            nonce,
            data,
        } => ArbTypedTransaction::Unsigned(ArbUnsignedTx {
            chain_id: chain_id_u256,
            from: *from,
            nonce: *nonce,
            gas_fee_cap: *gas_fee_cap,
            gas: *gas,
            to: *to,
            value: *value,
            data: Bytes::copy_from_slice(data),
        }),
        ParsedTransaction::ContractTx {
            from,
            to,
            value,
            gas,
            gas_fee_cap,
            data,
            request_id,
        } => ArbTypedTransaction::Contract(ArbContractTx {
            chain_id: chain_id_u256,
            request_id: *request_id,
            from: *from,
            gas_fee_cap: *gas_fee_cap,
            gas: *gas,
            to: *to,
            value: *value,
            data: Bytes::copy_from_slice(data),
        }),
        ParsedTransaction::EthDeposit {
            from,
            to,
            value,
            request_id,
        } => ArbTypedTransaction::Deposit(ArbDepositTx {
            chain_id: chain_id_u256,
            l1_request_id: *request_id,
            from: *from,
            to: *to,
            value: *value,
        }),
        ParsedTransaction::SubmitRetryable {
            request_id,
            l1_base_fee,
            deposit,
            callvalue,
            gas_feature_cap,
            gas_limit,
            max_submission_fee,
            from,
            to,
            fee_refund_addr,
            beneficiary,
            data,
        } => ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
            chain_id: chain_id_u256,
            request_id: *request_id,
            from: *from,
            l1_base_fee: *l1_base_fee,
            deposit_value: *deposit,
            gas_fee_cap: *gas_feature_cap,
            gas: *gas_limit,
            retry_to: *to,
            retry_value: *callvalue,
            beneficiary: *beneficiary,
            max_submission_fee: *max_submission_fee,
            fee_refund_addr: *fee_refund_addr,
            retry_data: Bytes::copy_from_slice(data),
        }),
        ParsedTransaction::BatchPostingReport { .. } => {
            // Batch posting reports become internal txs with ABI-encoded data.
            // These are constructed by the block producer, not this function.
            return None;
        }
        ParsedTransaction::InternalStartBlock { .. } => {
            // Start-block txs are constructed by the block producer.
            return None;
        }
    };

    let sig = alloy_primitives::Signature::new(U256::ZERO, U256::ZERO, false);
    Some(ArbTransactionSigned::new_unhashed(tx, sig))
}
