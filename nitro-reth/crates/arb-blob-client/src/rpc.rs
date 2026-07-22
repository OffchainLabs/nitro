use alloy_primitives::B256;

use crate::{Blob, BlobReader, Result};

#[derive(Debug)]
pub struct BeaconBlobReader {
    client: reqwest::Client,
}

#[async_trait::async_trait]
impl BlobReader for BeaconBlobReader {
    async fn initialize(&self) -> Result<()> {
        todo!()
    }

    async fn get_blobs(&self, block_hash: B256, versioned_hashes: &[B256]) -> Result<Vec<Blob>> {
        todo!()
    }
}
