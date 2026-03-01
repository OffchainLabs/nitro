use alloy_primitives::{Address, B256, U256};
use std::io::{self, Cursor, Read};

use crate::util::{
    address_from_256_from_reader, address_from_reader, hash_from_reader, uint256_from_reader,
    uint64_from_reader,
};

/// L1 message type constants.
pub const L1_MESSAGE_TYPE_L2_MESSAGE: u8 = 3;
pub const L1_MESSAGE_TYPE_END_OF_BLOCK: u8 = 6;
pub const L1_MESSAGE_TYPE_L2_FUNDED_BY_L1: u8 = 7;
pub const L1_MESSAGE_TYPE_ROLLUP_EVENT: u8 = 8;
pub const L1_MESSAGE_TYPE_SUBMIT_RETRYABLE: u8 = 9;
pub const L1_MESSAGE_TYPE_BATCH_FOR_GAS_ESTIMATION: u8 = 10;
pub const L1_MESSAGE_TYPE_INITIALIZE: u8 = 11;
pub const L1_MESSAGE_TYPE_ETH_DEPOSIT: u8 = 12;
pub const L1_MESSAGE_TYPE_BATCH_POSTING_REPORT: u8 = 13;
pub const L1_MESSAGE_TYPE_INVALID: u8 = 0xFF;

/// Maximum size of an L2 message payload.
pub const MAX_L2_MESSAGE_SIZE: usize = 256 * 1024;

/// Default initial L1 base fee (used when chain config doesn't specify one).
pub const DEFAULT_INITIAL_L1_BASE_FEE: u64 = 50_000_000_000; // 50 Gwei

/// Header of an L1 incoming message.
#[derive(Debug, Clone)]
pub struct L1IncomingMessageHeader {
    pub kind: u8,
    pub poster: Address,
    pub block_number: u64,
    pub timestamp: u64,
    pub request_id: Option<B256>,
    pub l1_base_fee: Option<U256>,
}

/// Statistics about a batch of data (for L1 cost estimation).
#[derive(Debug, Clone, Copy, Default)]
pub struct BatchDataStats {
    pub length: u64,
    pub non_zeros: u64,
}

/// An L1 incoming message containing the header and L2 payload.
#[derive(Debug, Clone)]
pub struct L1IncomingMessage {
    pub header: L1IncomingMessageHeader,
    pub l2_msg: Vec<u8>,
    /// Batch-level gas cost fields (filled lazily).
    pub batch_gas_left: Option<u64>,
}

/// Parsed initialization message from the first L1 message.
#[derive(Debug, Clone)]
pub struct ParsedInitMessage {
    pub chain_id: U256,
    pub initial_l1_base_fee: U256,
    /// Serialized chain config JSON bytes (stored in ArbOS state).
    pub serialized_chain_config: Vec<u8>,
}

impl L1IncomingMessageHeader {
    /// Extracts the sequence number from the RequestId.
    pub fn seq_num(&self) -> Option<u64> {
        self.request_id.map(|id| {
            let bytes = id.as_slice();
            u64::from_be_bytes(bytes[24..32].try_into().unwrap_or([0; 8]))
        })
    }
}

impl L1IncomingMessage {
    /// Returns batch numbers this message depends on.
    ///
    /// Only BatchPostingReport messages reference past batches; all other
    /// message types return an empty list.
    pub fn past_batches_required(&self) -> io::Result<Vec<u64>> {
        if self.header.kind != L1_MESSAGE_TYPE_BATCH_POSTING_REPORT {
            return Ok(Vec::new());
        }
        let fields = parse_batch_posting_report_fields(&self.l2_msg)?;
        Ok(vec![fields.batch_number])
    }

    /// Serializes this message to bytes.
    pub fn serialize(&self) -> Vec<u8> {
        let mut buf = Vec::new();
        buf.push(self.header.kind);
        // poster (32 bytes, left-padded address)
        buf.extend_from_slice(
            B256::left_padding_from(self.header.poster.as_slice()).as_slice(),
        );
        // block number (8 bytes BE)
        buf.extend_from_slice(&self.header.block_number.to_be_bytes());
        // timestamp (8 bytes BE)
        buf.extend_from_slice(&self.header.timestamp.to_be_bytes());
        // request id (32 bytes, zero if none)
        match &self.header.request_id {
            Some(id) => buf.extend_from_slice(id.as_slice()),
            None => buf.extend_from_slice(&[0u8; 32]),
        }
        // l1 base fee (32 bytes BE, zero if none)
        match &self.header.l1_base_fee {
            Some(fee) => buf.extend_from_slice(&fee.to_be_bytes::<32>()),
            None => buf.extend_from_slice(&[0u8; 32]),
        }
        // l2 msg
        buf.extend_from_slice(&self.l2_msg);
        buf
    }
}

/// Parses an L1 incoming message from raw bytes.
pub fn parse_incoming_l1_message(data: &[u8]) -> io::Result<L1IncomingMessage> {
    if data.is_empty() {
        return Err(io::Error::new(io::ErrorKind::InvalidData, "empty message"));
    }
    let mut reader = Cursor::new(data);

    let mut kind_buf = [0u8; 1];
    reader.read_exact(&mut kind_buf)?;
    let kind = kind_buf[0];

    let poster = address_from_256_from_reader(&mut reader)?;
    let block_number = uint64_from_reader(&mut reader)?;
    let timestamp = uint64_from_reader(&mut reader)?;
    let request_id = hash_from_reader(&mut reader)?;
    let l1_base_fee = uint256_from_reader(&mut reader)?;

    let request_id = if request_id == B256::ZERO {
        None
    } else {
        Some(request_id)
    };
    let l1_base_fee = if l1_base_fee == U256::ZERO {
        None
    } else {
        Some(l1_base_fee)
    };

    let mut l2_msg = Vec::new();
    reader.read_to_end(&mut l2_msg)?;

    Ok(L1IncomingMessage {
        header: L1IncomingMessageHeader {
            kind,
            poster,
            block_number,
            timestamp,
            request_id,
            l1_base_fee,
        },
        l2_msg,
        batch_gas_left: None,
    })
}

/// Parses an initialization message to extract chain ID and initial L1 base fee.
pub fn parse_init_message(data: &[u8]) -> io::Result<ParsedInitMessage> {
    if data.is_empty() {
        return Ok(ParsedInitMessage {
            chain_id: U256::ZERO,
            initial_l1_base_fee: U256::from(DEFAULT_INITIAL_L1_BASE_FEE),
            serialized_chain_config: Vec::new(),
        });
    }

    let mut reader = Cursor::new(data);

    // Version byte
    let mut version_buf = [0u8; 1];
    reader.read_exact(&mut version_buf)?;
    let version = version_buf[0];

    match version {
        0 => {
            let chain_id = uint256_from_reader(&mut reader)?;
            let mut serialized_chain_config = Vec::new();
            reader.read_to_end(&mut serialized_chain_config)?;
            Ok(ParsedInitMessage {
                chain_id,
                initial_l1_base_fee: U256::from(DEFAULT_INITIAL_L1_BASE_FEE),
                serialized_chain_config,
            })
        }
        1 => {
            let chain_id = uint256_from_reader(&mut reader)?;
            let initial_l1_base_fee = uint256_from_reader(&mut reader)?;
            let mut serialized_chain_config = Vec::new();
            reader.read_to_end(&mut serialized_chain_config)?;
            Ok(ParsedInitMessage {
                chain_id,
                initial_l1_base_fee,
                serialized_chain_config,
            })
        }
        _ => Err(io::Error::new(
            io::ErrorKind::InvalidData,
            format!("unsupported init message version: {version}"),
        )),
    }
}

/// Returns data statistics (total bytes and non-zero byte count).
pub fn get_data_stats(data: &[u8]) -> BatchDataStats {
    let non_zeros = data.iter().filter(|&&b| b != 0).count() as u64;
    BatchDataStats {
        length: data.len() as u64,
        non_zeros,
    }
}

/// Estimates L1 gas cost using legacy pricing model.
pub fn legacy_cost_for_stats(stats: &BatchDataStats) -> u64 {
    let zeros = stats.length.saturating_sub(stats.non_zeros);
    // Calldata gas: 4 gas per zero byte, 16 gas per non-zero byte.
    let mut gas = zeros * 4 + stats.non_zeros * 16;
    // Poster also pays to keccak the batch and write a batch posting report.
    let keccak_words = (stats.length + 31) / 32;
    gas += 30 + keccak_words * 6; // Keccak256Gas + words * Keccak256WordGas
    gas += 2 * 20_000; // 2 × SstoreSetGasEIP2200
    gas
}

/// Parses fields from a batch posting report message.
pub fn parse_batch_posting_report_fields(data: &[u8]) -> io::Result<BatchPostingReportFields> {
    let mut reader = Cursor::new(data);

    let batch_timestamp_u256 = uint256_from_reader(&mut reader)?;
    let batch_timestamp: u64 = batch_timestamp_u256.try_into().map_err(|_| {
        io::Error::new(io::ErrorKind::InvalidData, "batch timestamp too large")
    })?;

    let batch_poster = address_from_reader(&mut reader)?;
    let data_hash = hash_from_reader(&mut reader)?;

    let batch_number_u256 = uint256_from_reader(&mut reader)?;
    let batch_number: u64 = batch_number_u256.try_into().map_err(|_| {
        io::Error::new(io::ErrorKind::InvalidData, "batch number too large")
    })?;

    let l1_base_fee_estimate = uint256_from_reader(&mut reader)?;

    let extra_gas = match uint64_from_reader(&mut reader) {
        Ok(v) => v,
        Err(e) if e.kind() == io::ErrorKind::UnexpectedEof => 0,
        Err(e) => return Err(e),
    };

    Ok(BatchPostingReportFields {
        batch_timestamp,
        batch_poster,
        data_hash,
        batch_number,
        l1_base_fee_estimate,
        extra_gas,
    })
}

/// Fields extracted from a batch posting report.
#[derive(Debug, Clone)]
pub struct BatchPostingReportFields {
    pub batch_timestamp: u64,
    pub batch_poster: Address,
    pub data_hash: B256,
    pub batch_number: u64,
    pub l1_base_fee_estimate: U256,
    pub extra_gas: u64,
}
