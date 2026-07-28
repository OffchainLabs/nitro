//! A [`DaReader`] backed by a [`BlobReader`].
//!
//! Ports nitro's `daprovider.readerForBlobReader`: it recovers a blob-hashes
//! batch by parsing the EIP-4844 versioned hashes out of the sequencer message,
//! fetching the blobs, and decoding them back into the batch payload. Registered
//! under the blob-hashes header byte (`0x50`) in a `DaReaderSource` and supplied
//! to the MEL runner by the consensus node.

use std::sync::Arc;

use alloy_primitives::B256;
use arb_da_provider_client::{DaError, DaReader, Payload, PreimageType, Preimages, Result};

use crate::{BlobReader, blobs::decode_blobs};

/// Bytes before the blob versioned hashes in a sequencer message: the 40-byte L1
/// batch header plus the 1-byte DA header flag (`0x50`).
const BLOB_HASHES_OFFSET: usize = 41;

/// A [`DaReader`] that recovers blob-hashes batches through a [`BlobReader`].
pub struct ReaderForBlobReader {
    blob_reader: Arc<dyn BlobReader>,
}

impl ReaderForBlobReader {
    /// Wraps `blob_reader` as a [`DaReader`]. The blob reader must already be
    /// initialized (see [`BlobReader::initialize`]).
    pub fn new(blob_reader: Arc<dyn BlobReader>) -> Self {
        Self { blob_reader }
    }

    /// Shared implementation for the three `DaReader` methods, mirroring nitro's
    /// `recoverInternal`.
    async fn recover_internal(
        &self,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
        need_payload: bool,
        need_preimages: bool,
    ) -> Result<(Payload, Preimages)> {
        if sequencer_msg.len() < BLOB_HASHES_OFFSET {
            return Err(DaError::MalformedSequencerMessage(format!(
                "sequencer message too short for blob batch: expected at least {BLOB_HASHES_OFFSET} bytes, got {}",
                sequencer_msg.len()
            )));
        }
        let blob_hashes = &sequencer_msg[BLOB_HASHES_OFFSET..];
        if !blob_hashes.len().is_multiple_of(32) {
            return Err(DaError::MalformedSequencerMessage(
                "blob batch data is not a list of hashes as expected".to_string(),
            ));
        }
        let versioned_hashes: Vec<B256> =
            blob_hashes.chunks_exact(32).map(B256::from_slice).collect();

        let kzg_blobs = self
            .blob_reader
            .get_blobs(batch_block_hash, &versioned_hashes)
            .await
            .map_err(DaError::provider)?;

        let mut preimages = Preimages::new();
        if need_preimages {
            let map = preimages.entry(PreimageType::EthVersionedHash).or_default();
            for (hash, blob) in versioned_hashes.iter().zip(kzg_blobs.iter()) {
                map.insert(*hash, blob.0.as_slice().to_vec());
            }
        }

        let mut payload = Payload::new();
        if need_payload {
            match decode_blobs(&kzg_blobs) {
                Ok(decoded) => payload = decoded,
                Err(e) => {
                    // A KZG-valid blob can still carry data that fails to decode
                    // (e.g. a buggy/malicious sequencer). Nitro treats this as an
                    // empty batch rather than halting the chain, so we log and
                    // return nothing.
                    tracing::error!(
                        %batch_block_hash,
                        error = %e,
                        "failed to decode blobs; treating as empty batch",
                    );
                    return Ok((Payload::new(), Preimages::new()));
                }
            }
        }

        Ok((payload, preimages))
    }
}

#[async_trait::async_trait]
impl DaReader for ReaderForBlobReader {
    async fn recover_payload(
        &self,
        _batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Payload> {
        let (payload, _) = self
            .recover_internal(batch_block_hash, sequencer_msg, true, false)
            .await?;
        Ok(payload)
    }

    async fn collect_preimages(
        &self,
        _batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Preimages> {
        let (_, preimages) = self
            .recover_internal(batch_block_hash, sequencer_msg, false, true)
            .await?;
        Ok(preimages)
    }

    async fn recover_payload_and_preimages(
        &self,
        _batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<(Payload, Preimages)> {
        self.recover_internal(batch_block_hash, sequencer_msg, true, true)
            .await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{BlobError, MockBlobReader, blobs::encode_blobs};

    fn block_hash() -> B256 {
        B256::repeat_byte(0x11)
    }

    /// Builds a serialized blob-hashes sequencer message: a 40-byte L1 header, the
    /// `0x50` flag, then the concatenated versioned hashes.
    fn sequencer_msg(versioned_hashes: &[B256]) -> Vec<u8> {
        let mut msg = vec![0u8; 40];
        msg.push(0x50);
        for h in versioned_hashes {
            msg.extend_from_slice(h.as_slice());
        }
        msg
    }

    /// A reader whose blob store returns `encode_blobs(payload)` for the batch.
    fn reader_with(payload: &[u8], hashes: &[B256]) -> ReaderForBlobReader {
        let blobs = encode_blobs(payload).unwrap();
        assert_eq!(
            blobs.len(),
            hashes.len(),
            "test must supply one hash per blob"
        );
        let mut mock = MockBlobReader::new();
        mock.with_blobs(block_hash(), hashes, blobs);
        ReaderForBlobReader::new(Arc::new(mock))
    }

    #[tokio::test]
    async fn recovers_payload_from_blobs() {
        let payload = b"hello arbitrum blob batch".to_vec();
        let hashes = vec![B256::repeat_byte(0xaa)];
        let reader = reader_with(&payload, &hashes);

        let got = reader
            .recover_payload(1, block_hash(), &sequencer_msg(&hashes))
            .await
            .unwrap();
        assert_eq!(got, payload);
    }

    #[tokio::test]
    async fn collects_blobs_as_versioned_hash_preimages() {
        let payload = b"preimage payload".to_vec();
        let hashes = vec![B256::repeat_byte(0xcd)];
        let blobs = encode_blobs(&payload).unwrap();
        let reader = reader_with(&payload, &hashes);

        let preimages = reader
            .collect_preimages(1, block_hash(), &sequencer_msg(&hashes))
            .await
            .unwrap();

        let map = &preimages[&PreimageType::EthVersionedHash];
        assert_eq!(map.len(), 1);
        assert_eq!(map[&hashes[0]], blobs[0].0.as_slice());
    }

    #[tokio::test]
    async fn too_short_message_errors() {
        let reader = ReaderForBlobReader::new(Arc::new(MockBlobReader::new()));
        let err = reader
            .recover_payload(1, block_hash(), &[0u8; 40])
            .await
            .unwrap_err();
        assert!(matches!(err, DaError::MalformedSequencerMessage(_)));
    }

    #[tokio::test]
    async fn non_hash_length_data_errors() {
        let reader = ReaderForBlobReader::new(Arc::new(MockBlobReader::new()));
        // 41-byte header + flag, then 10 trailing bytes (not a multiple of 32).
        let mut msg = vec![0u8; 41];
        msg.extend_from_slice(&[0u8; 10]);
        let err = reader
            .recover_payload(1, block_hash(), &msg)
            .await
            .unwrap_err();
        assert!(matches!(err, DaError::MalformedSequencerMessage(_)));
    }

    #[tokio::test]
    async fn blob_reader_failure_keeps_the_concrete_error() {
        let hashes = vec![B256::repeat_byte(0xaa)];
        let mut mock = MockBlobReader::new();
        mock.with_error(block_hash(), &hashes, || BlobError::NotInitialized);
        let reader = ReaderForBlobReader::new(Arc::new(mock));

        let err = reader
            .recover_payload(1, block_hash(), &sequencer_msg(&hashes))
            .await
            .unwrap_err();

        let source = match &err {
            DaError::Provider(source) => source,
            other => panic!("expected Provider, got {other:?}"),
        };
        // The blob reader's own error survives the boxing, so a caller that
        // knows the concrete type can act on the specific failure.
        assert!(matches!(
            source.downcast_ref::<BlobError>(),
            Some(BlobError::NotInitialized)
        ));
        // ...and it still reads without downcasting.
        assert!(err.to_string().contains("not initialized"));
    }
}
