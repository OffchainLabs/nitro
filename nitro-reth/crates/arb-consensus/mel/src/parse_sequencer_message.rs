//! Parsing a serialized sequencer batch into a [`SequencerMessage`].
//!
//! Port of `arbstate.ParseSequencerMessage` (`arbstate/inbox.go`). A serialized
//! batch is laid out as:
//!
//! ```text
//! [ 40-byte header | payload ]
//! header  = min_timestamp(8) | max_timestamp(8) | min_l1_block(8)
//!           | max_l1_block(8) | after_delayed_messages(8)   (all big-endian)
//! payload = <1 header byte> <encoded data>
//! ```
//!
//! The payload's first byte selects an encoding / data-availability strategy.
// The parser and its header-byte helpers are a complete, tested port that is not
// yet wired into a caller; allow dead code until the MEL pipeline consumes it.
#![allow(dead_code)]
use std::mem::MaybeUninit;

use alloy_primitives::B256;
use arb_da_provider::DaReaderSource;
use tracing::warn;

use crate::{MelError, MelResult};

/// Maximum number of segments parsed from a single sequencer message.
pub const MAX_SEGMENTS_PER_SEQUENCER_MESSAGE: usize = 100 * 1024;

/// Default cap on the decompressed size of a batch payload (16 MiB), matching
/// `params.DefaultMaxUncompressedBatchSize`.
pub const DEFAULT_MAX_UNCOMPRESSED_BATCH_SIZE: usize = 16 * 1024 * 1024;

/// Length in bytes of the fixed batch header preceding the payload.
const HEADER_LEN: usize = 40;

// Payload header-byte flags (port of the `daprovider` constants).
const ANYTRUST_FLAG: u8 = 0x80;
const ANYTRUST_TREE_FLAG: u8 = 0x08;
const L1_AUTHENTICATED_FLAG: u8 = 0x40;
const ZEROHEAVY_FLAG: u8 = 0x20;
const BLOB_HASHES_FLAG: u8 = L1_AUTHENTICATED_FLAG | 0x10; // 0x50
const DACERT_FLAG: u8 = 0x01;
const BROTLI_HEADER_BYTE: u8 = 0x00;
const KNOWN_HEADER_BITS: u8 = ANYTRUST_FLAG
    | ANYTRUST_TREE_FLAG
    | L1_AUTHENTICATED_FLAG
    | ZEROHEAVY_FLAG
    | BLOB_HASHES_FLAG
    | DACERT_FLAG;

fn has_bits(value: u8, bits: u8) -> bool {
    value & bits == bits
}
fn is_l1_authenticated(b: u8) -> bool {
    has_bits(b, L1_AUTHENTICATED_FLAG)
}
fn is_anytrust(b: u8) -> bool {
    has_bits(b, ANYTRUST_FLAG)
}
fn is_zeroheavy(b: u8) -> bool {
    has_bits(b, ZEROHEAVY_FLAG)
}
fn is_blob_hashes(b: u8) -> bool {
    has_bits(b, BLOB_HASHES_FLAG)
}
fn is_dacert(b: u8) -> bool {
    b == DACERT_FLAG
}
fn is_brotli(b: u8) -> bool {
    b == BROTLI_HEADER_BYTE
}
fn is_known_header_byte(b: u8) -> bool {
    b & !KNOWN_HEADER_BITS == 0
}

/// A parsed sequencer batch: the L1 time/block bounds plus the raw L2 message segments.
#[derive(Debug, Clone, Default)]
pub struct SequencerMessage {
    pub min_timestamp: u64,
    pub max_timestamp: u64,
    pub min_l1_block: u64,
    pub max_l1_block: u64,
    pub after_delayed_messages: u64,
    pub segments: Vec<Vec<u8>>,
}

/// Parses a serialized sequencer batch into a [`SequencerMessage`].
///
/// When the payload's header byte selects a data-availability provider, the
/// underlying payload is recovered through `da_reader_source` and then decoded
/// like any other batch (mirroring `arbstate.ParseSequencerMessage`).
pub(crate) async fn parse_sequencer_message(
    batch_num: u64,
    batch_block_hash: B256,
    data: &[u8],
    max_uncompressed_batch_size: usize,
    da_reader_source: &dyn DaReaderSource,
) -> MelResult<SequencerMessage> {
    if data.len() < HEADER_LEN {
        return Err(MelError::SequencerMessageTooShort);
    }

    let read_u64 = |start: usize| -> u64 {
        let mut buf = [0u8; 8];
        buf.copy_from_slice(&data[start..start + 8]);
        u64::from_be_bytes(buf)
    };
    let mut parsed = SequencerMessage {
        min_timestamp: read_u64(0),
        max_timestamp: read_u64(8),
        min_l1_block: read_u64(16),
        max_l1_block: read_u64(24),
        after_delayed_messages: read_u64(32),
        segments: Vec::new(),
    };

    let payload = &data[HEADER_LEN..];
    if payload.is_empty() {
        return Ok(parsed);
    }
    let header_byte = payload[0];

    // An L1-authenticated batch with an unrecognised header byte means this node
    // is behind the on-chain protocol.
    if is_l1_authenticated(header_byte) && !is_known_header_byte(header_byte) {
        return Err(MelError::NodeOutOfDate {
            batch_num,
            header_byte,
        });
    }

    // Data-availability recovery: if the header byte selects a DA provider,
    // recover the underlying payload through the registered reader and continue
    // decoding it. A DA header byte with no registered reader is unsupported.
    let recovered: Vec<u8>;
    let payload: &[u8] =
        if is_anytrust(header_byte) || is_blob_hashes(header_byte) || is_dacert(header_byte) {
            match da_reader_source.get_reader(header_byte) {
                Some(reader) => {
                    recovered = reader
                        .recover_payload(batch_num, batch_block_hash, data)
                        .await?;
                    // An empty recovered payload is an empty batch (e.g. a DA
                    // certificate that failed validation and is skipped).
                    if recovered.is_empty() {
                        return Ok(parsed);
                    }
                    &recovered
                }
                None => {
                    return Err(MelError::UnsupportedDaHeaderByte {
                        batch_num,
                        header_byte,
                    });
                }
            }
        } else {
            payload
        };
    let header_byte = payload[0];

    if is_zeroheavy(header_byte) {
        return Err(MelError::UnsupportedEncoding("zeroheavy"));
    }

    if is_brotli(header_byte) {
        match decompress_brotli(&payload[1..], max_uncompressed_batch_size) {
            Ok(decompressed) => parsed.segments = parse_segments(&decompressed),
            Err(error) => warn!("Sequencer msg decompression failed: {error:?}"),
        }
        return Ok(parsed);
    }

    Ok(parsed)
}

/// Brotli-decompresses `compressed`, failing if the stream is malformed or the
/// output exceeds `max_size` bytes.
pub(crate) fn decompress_brotli(compressed: &[u8], max_size: usize) -> MelResult<Vec<u8>> {
    let mut buf = vec![MaybeUninit::<u8>::uninit(); max_size];
    let out = nitro_brotli::decompress_fixed(compressed, &mut buf, nitro_brotli::Dictionary::Empty)
        .map_err(|_| MelError::BatchDecompressionFailed)?;
    Ok(out.to_vec())
}

fn parse_segments(decompressed: &[u8]) -> Vec<Vec<u8>> {
    let mut segments = Vec::new();
    let mut buf: &[u8] = decompressed;
    while !buf.is_empty() {
        let header = match alloy_rlp::Header::decode(&mut buf) {
            Ok(h) => h,
            Err(_) => break,
        };
        // Segments are byte strings; a list or an out-of-bounds length is
        // malformed and ends parsing, as does hitting the segment cap.
        if header.list
            || header.payload_length > buf.len()
            || segments.len() >= MAX_SEGMENTS_PER_SEQUENCER_MESSAGE
        {
            break;
        }
        let (segment, rest) = buf.split_at(header.payload_length);
        segments.push(segment.to_vec());
        buf = rest;
    }
    segments
}

#[cfg(test)]
mod tests {
    use std::sync::Arc;

    use arb_da_provider::{DaReaderRegistry, MockDaReader, Preimages};

    use super::*;
    use crate::test_utils::brotli_compress;

    /// An empty DA source for the non-DA test cases.
    fn no_da() -> DaReaderRegistry {
        DaReaderRegistry::new()
    }

    /// Builds the fixed 40-byte batch header followed by `payload`.
    fn frame(header: [u64; 5], payload: &[u8]) -> Vec<u8> {
        let mut data = Vec::with_capacity(HEADER_LEN + payload.len());
        for v in header {
            data.extend_from_slice(&v.to_be_bytes());
        }
        data.extend_from_slice(payload);
        data
    }

    /// RLP-encodes each segment as a byte string and concatenates them, the
    /// on-wire form `parse_segments` expects.
    fn rlp_segments(segments: &[&[u8]]) -> Vec<u8> {
        let mut out = Vec::new();
        for seg in segments {
            out.extend_from_slice(&alloy_rlp::encode(*seg));
        }
        out
    }

    #[test]
    fn header_bit_helpers() {
        assert!(is_l1_authenticated(L1_AUTHENTICATED_FLAG));
        assert!(is_anytrust(ANYTRUST_FLAG));
        assert!(is_zeroheavy(ZEROHEAVY_FLAG));
        assert!(is_blob_hashes(BLOB_HASHES_FLAG));
        assert!(is_dacert(DACERT_FLAG));
        assert!(is_brotli(BROTLI_HEADER_BYTE));
        assert!(is_known_header_byte(BROTLI_HEADER_BYTE));
        // A blob-hashes byte is L1-authenticated but still a known header byte.
        assert!(is_known_header_byte(BLOB_HASHES_FLAG));
        // An L1-authenticated byte with an unknown low bit is not recognised.
        assert!(!is_known_header_byte(L1_AUTHENTICATED_FLAG | 0x02));
    }

    #[tokio::test]
    async fn rejects_data_shorter_than_header() {
        let result =
            parse_sequencer_message(0, B256::ZERO, &[0u8; HEADER_LEN - 1], usize::MAX, &no_da())
                .await;
        assert!(matches!(result, Err(MelError::SequencerMessageTooShort)));
    }

    #[tokio::test]
    async fn parses_header_fields_with_empty_payload() -> MelResult<()> {
        let data = frame([1, 2, 3, 4, 5], &[]);
        let msg = parse_sequencer_message(7, B256::ZERO, &data, usize::MAX, &no_da()).await?;
        assert_eq!(msg.min_timestamp, 1);
        assert_eq!(msg.max_timestamp, 2);
        assert_eq!(msg.min_l1_block, 3);
        assert_eq!(msg.max_l1_block, 4);
        assert_eq!(msg.after_delayed_messages, 5);
        assert!(msg.segments.is_empty());
        Ok(())
    }

    #[tokio::test]
    async fn parses_brotli_encoded_segments() -> MelResult<()> {
        let seg_a: &[u8] = b"hello";
        let seg_b: &[u8] = b"world!!";
        let raw = rlp_segments(&[seg_a, seg_b]);
        let mut payload = vec![BROTLI_HEADER_BYTE];
        payload.extend_from_slice(&brotli_compress(&raw));

        let data = frame([10, 20, 30, 40, 2], &payload);
        let msg = parse_sequencer_message(
            1,
            B256::ZERO,
            &data,
            DEFAULT_MAX_UNCOMPRESSED_BATCH_SIZE,
            &no_da(),
        )
        .await?;
        assert_eq!(msg.after_delayed_messages, 2);
        assert_eq!(msg.segments, vec![seg_a.to_vec(), seg_b.to_vec()]);
        Ok(())
    }

    #[tokio::test]
    async fn oversized_decompression_yields_no_segments() -> MelResult<()> {
        // Cap decompression below the real output; the decode fails (Go
        // parity: no truncation) and the batch parses as empty segments.
        let raw = rlp_segments(&[b"aaaaaaaaaaaaaaaaaaaa"]);
        let mut payload = vec![BROTLI_HEADER_BYTE];
        payload.extend_from_slice(&brotli_compress(&raw));
        let data = frame([0; 5], &payload);
        let msg = parse_sequencer_message(1, B256::ZERO, &data, 1, &no_da()).await?;
        assert!(msg.segments.is_empty());
        Ok(())
    }

    #[tokio::test]
    async fn oversized_decompression_does_not_yield_truncated_segments() -> MelResult<()> {
        // Cap decompression at exactly the encoded length of the first segment,
        // so a decode that truncated to the buffer instead of failing would
        // still parse into one well-formed segment. The batch must come back
        // empty rather than half-parsed.
        let first: &[u8] = b"first-segment-payload";
        let second: &[u8] = b"second-segment-payload";
        let cap = alloy_rlp::encode(first).len();
        let raw = rlp_segments(&[first, second]);
        assert!(raw.len() > cap, "fixture must decompress past the cap");
        // The prefix the buffer would hold is itself a valid single segment,
        // which is what makes truncation observable here.
        assert_eq!(parse_segments(&raw[..cap]), vec![first.to_vec()]);

        let mut payload = vec![BROTLI_HEADER_BYTE];
        payload.extend_from_slice(&brotli_compress(&raw));
        let data = frame([0; 5], &payload);

        let msg = parse_sequencer_message(1, B256::ZERO, &data, cap, &no_da()).await?;
        assert!(msg.segments.is_empty());
        Ok(())
    }

    #[tokio::test]
    async fn rejects_zeroheavy_encoding() {
        let data = frame([0; 5], &[ZEROHEAVY_FLAG]);
        let result = parse_sequencer_message(3, B256::ZERO, &data, usize::MAX, &no_da()).await;
        assert!(matches!(
            result,
            Err(MelError::UnsupportedEncoding("zeroheavy"))
        ));
    }

    #[tokio::test]
    async fn da_header_byte_without_reader_is_unsupported() {
        // A DA header byte with no registered reader is rejected.
        for byte in [ANYTRUST_FLAG, BLOB_HASHES_FLAG, DACERT_FLAG] {
            let data = frame([0; 5], &[byte]);
            let result = parse_sequencer_message(9, B256::ZERO, &data, usize::MAX, &no_da()).await;
            assert!(
                matches!(result, Err(MelError::UnsupportedDaHeaderByte { batch_num: 9, header_byte }) if header_byte == byte),
                "expected UnsupportedDaHeaderByte for header byte {byte:#04x}"
            );
        }
    }

    #[tokio::test]
    async fn recovers_da_payload_and_parses_segments() -> MelResult<()> {
        // The payload recovered from the DA provider is itself a brotli-compressed
        // sequencer payload, decoded like any other batch after recovery.
        let seg: &[u8] = b"da-recovered-segment";
        let mut recovered = vec![BROTLI_HEADER_BYTE];
        recovered.extend_from_slice(&brotli_compress(&rlp_segments(&[seg])));

        // A batch whose payload header byte selects an AnyTrust DA provider.
        let data = frame([1, 2, 3, 4, 5], &[ANYTRUST_FLAG]);
        let block_hash = B256::repeat_byte(0x11);

        let mut reader = MockDaReader::new();
        reader.with_batch(1, block_hash, &data, recovered, Preimages::new());
        let mut registry = DaReaderRegistry::new();
        registry.register(ANYTRUST_FLAG, Arc::new(reader)).unwrap();

        let msg = parse_sequencer_message(
            1,
            block_hash,
            &data,
            DEFAULT_MAX_UNCOMPRESSED_BATCH_SIZE,
            &registry,
        )
        .await?;
        assert_eq!(msg.segments, vec![seg.to_vec()]);
        Ok(())
    }

    #[tokio::test]
    async fn rejects_node_out_of_date_header_byte() {
        // L1-authenticated flag set together with an unknown bit.
        let byte = L1_AUTHENTICATED_FLAG | 0x02;
        let data = frame([0; 5], &[byte]);
        let result = parse_sequencer_message(5, B256::ZERO, &data, usize::MAX, &no_da()).await;
        assert!(matches!(
            result,
            Err(MelError::NodeOutOfDate { batch_num: 5, header_byte }) if header_byte == byte
        ));
    }

    #[tokio::test]
    async fn unknown_non_authenticated_header_byte_yields_empty_batch() -> MelResult<()> {
        // A non-authenticated, non-brotli byte falls through to an empty batch.
        let data = frame([0; 5], &[0x04]);
        let msg = parse_sequencer_message(1, B256::ZERO, &data, usize::MAX, &no_da()).await?;
        assert!(msg.segments.is_empty());
        Ok(())
    }
}
