//! Encoding/decoding batch data to and from EIP-4844 blobs.
//!
//! Ports nitro's `util/blobs` (`EncodeBlobs`/`DecodeBlobs`). A blob is 4096
//! 32-byte field elements; each element must be a valid BLS scalar, so it can't
//! use all 256 bits. The scheme packs 254 usable bits per element: 31 whole
//! bytes in positions `1..32`, plus 6 "spare" bits in byte `0` (byte 0 stays
//! below the BLS modulus). The payload is RLP-encoded before packing, so the
//! encoded length is self-describing on decode.

use alloy_eips::eip4844::{BYTES_PER_BLOB, FIELD_ELEMENTS_PER_BLOB};
use alloy_primitives::Bytes;
use alloy_rlp::Decodable;

use crate::Blob;

/// Bits of each field element's byte 0 that carry data (`floor(log2(BLS_MODULUS)) % 8`).
const SPARE_BLOB_BITS: i32 = 6;

/// Something went wrong encoding or decoding blob data.
#[derive(Debug, thiserror::Error)]
pub enum BlobCodecError {
    /// The 6-bit accumulator didn't land on a byte boundary — the input isn't a
    /// whole number of packed field elements.
    #[error("leftover spare accumulator bits: {0}")]
    SpareBits(i32),
    /// The reassembled bytes weren't valid RLP.
    #[error("failed to RLP-decode blob data: {0}")]
    Rlp(alloy_rlp::Error),
}

fn field_elements() -> usize {
    FIELD_ELEMENTS_PER_BLOB as usize
}

/// Fills the 31-byte data region (`1..32`) of each field element from `data`,
/// returning the unconsumed remainder.
fn fill_blob_bytes<'a>(blob: &mut [u8; BYTES_PER_BLOB], mut data: &'a [u8]) -> &'a [u8] {
    for field_element in 0..field_elements() {
        let start = field_element * 32 + 1;
        let n = data.len().min(31);
        blob[start..start + n].copy_from_slice(&data[..n]);
        if data.len() <= 31 {
            return &[];
        }
        data = &data[31..];
    }
    data
}

/// Packs `data` bytes, 6 bits at a time, into byte 0 of each field element,
/// returning the unconsumed remainder.
fn fill_blob_bits<'a>(
    blob: &mut [u8; BYTES_PER_BLOB],
    mut data: &'a [u8],
) -> Result<&'a [u8], BlobCodecError> {
    let mut acc: u16 = 0;
    let mut acc_bits: i32 = 0;
    for field_element in 0..field_elements() {
        if acc_bits < SPARE_BLOB_BITS && !data.is_empty() {
            acc |= (data[0] as u16) << acc_bits;
            acc_bits += 8;
            data = &data[1..];
        }
        blob[field_element * 32] = (acc & ((1 << SPARE_BLOB_BITS) - 1)) as u8;
        acc_bits -= SPARE_BLOB_BITS;
        if acc_bits < 0 {
            // Out of data.
            break;
        }
        acc >>= SPARE_BLOB_BITS;
    }
    if acc_bits > 0 {
        return Err(BlobCodecError::SpareBits(acc_bits));
    }
    Ok(data)
}

/// Encodes `data` into EIP-4844 blobs. Mirrors nitro's `EncodeBlobs`.
pub fn encode_blobs(data: &[u8]) -> Result<Vec<Blob>, BlobCodecError> {
    let rlp = alloy_rlp::encode(Bytes::copy_from_slice(data));
    let mut remaining: &[u8] = &rlp;
    let mut blobs = Vec::new();
    while !remaining.is_empty() {
        let mut bytes = [0u8; BYTES_PER_BLOB];
        remaining = fill_blob_bytes(&mut bytes, remaining);
        remaining = fill_blob_bits(&mut bytes, remaining)?;
        blobs.push(Blob::new(bytes));
    }
    Ok(blobs)
}

/// Decodes `blobs` into the batch data encoded within them. Mirrors nitro's
/// `DecodeBlobs`.
pub fn decode_blobs(blobs: &[Blob]) -> Result<Vec<u8>, BlobCodecError> {
    let mut rlp_data: Vec<u8> = Vec::new();
    for blob in blobs {
        let bytes = &blob.0;
        // The 31 data bytes from each field element (positions 1..32).
        for field_index in 0..field_elements() {
            let start = field_index * 32 + 1;
            rlp_data.extend_from_slice(&bytes[start..start + 31]);
        }
        // Reassemble the bytes packed 6-bits-at-a-time into each element's byte 0.
        let mut acc: u16 = 0;
        let mut acc_bits: i32 = 0;
        for field_index in 0..field_elements() {
            acc |= (bytes[field_index * 32] as u16) << acc_bits;
            acc_bits += SPARE_BLOB_BITS;
            if acc_bits >= 8 {
                rlp_data.push(acc as u8);
                acc >>= 8;
                acc_bits -= 8;
            }
        }
        if acc_bits != 0 {
            return Err(BlobCodecError::SpareBits(acc_bits));
        }
    }
    // The payload is a single RLP byte string; trailing blob padding is ignored.
    let mut slice = rlp_data.as_slice();
    let decoded = Bytes::decode(&mut slice).map_err(BlobCodecError::Rlp)?;
    Ok(decoded.to_vec())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn round_trip(data: &[u8]) {
        let blobs = encode_blobs(data).unwrap();
        let decoded = decode_blobs(&blobs).unwrap();
        assert_eq!(decoded, data);
    }

    #[test]
    fn round_trips_small_payload() {
        round_trip(b"hello arbitrum blob batch");
    }

    #[test]
    fn round_trips_empty_payload() {
        round_trip(&[]);
    }

    #[test]
    fn round_trips_payload_spanning_multiple_blobs() {
        // Larger than one blob's worth of encodable data (254 * 4096 / 8 bytes).
        let data: Vec<u8> = (0..200_000u32).map(|i| (i % 251) as u8).collect();
        let blobs = encode_blobs(&data).unwrap();
        assert!(blobs.len() >= 2);
        assert_eq!(decode_blobs(&blobs).unwrap(), data);
    }

    #[test]
    fn decode_rejects_non_rlp_blob() {
        // A blob full of 0xff bytes doesn't reassemble into a valid RLP string.
        let junk = Blob::repeat_byte(0xff);
        assert!(matches!(decode_blobs(&[junk]), Err(BlobCodecError::Rlp(_))));
    }
}
