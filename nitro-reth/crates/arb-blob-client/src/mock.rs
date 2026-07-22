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
