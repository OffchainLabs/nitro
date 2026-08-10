use std::collections::HashMap;

use alloy_primitives::B256;

use crate::{Blob, BlobError, BlobReader, Result};

/// Mints a fresh [`BlobError`] on each read — `BlobError` isn't `Clone`, so a
/// registered failure is stored as a factory rather than a value.
type ErrorFactory = Box<dyn Fn() -> BlobError + Send + Sync>;

#[derive(Default)]
pub struct MockBlobReader {
    blobs: HashMap<(B256, Vec<B256>), Result<Vec<Blob>, ErrorFactory>>,
}

impl MockBlobReader {
    pub fn new() -> Self {
        Self::default()
    }

    /// Registers the blobs returned for a `(block_hash, versioned_hashes)` pair.
    pub fn with_blobs(
        &mut self,
        block_hash: B256,
        versioned_hashes: &[B256],
        blobs: Vec<Blob>,
    ) -> &mut Self {
        self.blobs
            .insert((block_hash, versioned_hashes.to_vec()), Ok(blobs));
        self
    }

    /// Registers a lookup that fails, minting the error from `error` each read.
    pub fn with_error(
        &mut self,
        block_hash: B256,
        versioned_hashes: &[B256],
        error: impl Fn() -> BlobError + Send + Sync + 'static,
    ) -> &mut Self {
        self.blobs.insert(
            (block_hash, versioned_hashes.to_vec()),
            Err(Box::new(error)),
        );
        self
    }
}

#[async_trait::async_trait]
impl BlobReader for MockBlobReader {
    async fn initialize(&self) -> Result<()> {
        Ok(())
    }

    async fn get_blobs(&self, block_hash: B256, versioned_hashes: &[B256]) -> Result<Vec<Blob>> {
        match self.blobs.get(&(block_hash, versioned_hashes.to_vec())) {
            Some(Ok(blobs)) => Ok(blobs.clone()),
            Some(Err(make_error)) => Err(make_error()),
            None => Ok(Vec::new()),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn block_hash() -> B256 {
        B256::repeat_byte(0x11)
    }

    fn hashes() -> Vec<B256> {
        vec![B256::repeat_byte(0xaa), B256::repeat_byte(0xbb)]
    }

    fn blob(byte: u8) -> Blob {
        Blob::repeat_byte(byte)
    }

    #[tokio::test]
    async fn registered_blobs_read_back() {
        let mut mock = MockBlobReader::new();
        mock.with_blobs(block_hash(), &hashes(), vec![blob(1), blob(2)]);

        let got = mock.get_blobs(block_hash(), &hashes()).await.unwrap();
        assert_eq!(got, vec![blob(1), blob(2)]);
    }

    #[tokio::test]
    async fn unregistered_reads_back_empty() {
        let mock = MockBlobReader::new();
        let got = mock.get_blobs(block_hash(), &hashes()).await.unwrap();
        assert!(got.is_empty());
    }

    #[tokio::test]
    async fn with_error_replays_on_each_read() {
        let mut mock = MockBlobReader::new();
        mock.with_error(block_hash(), &hashes(), || BlobError::NotInitialized);

        // The factory mints a fresh error on every read.
        for _ in 0..2 {
            let err = mock.get_blobs(block_hash(), &hashes()).await.unwrap_err();
            assert!(matches!(err, BlobError::NotInitialized));
        }
    }

    #[tokio::test]
    async fn lookup_uses_the_full_key() {
        let mut mock = MockBlobReader::new();
        mock.with_blobs(block_hash(), &hashes(), vec![blob(1)]);

        // Right block, wrong versioned hashes -> unregistered -> empty.
        assert!(
            mock.get_blobs(block_hash(), &[B256::ZERO])
                .await
                .unwrap()
                .is_empty()
        );
        // Wrong block, right versioned hashes -> unregistered -> empty.
        assert!(
            mock.get_blobs(B256::ZERO, &hashes())
                .await
                .unwrap()
                .is_empty()
        );
    }

    #[tokio::test]
    async fn initialize_is_ok() {
        assert!(MockBlobReader::new().initialize().await.is_ok());
    }
}
