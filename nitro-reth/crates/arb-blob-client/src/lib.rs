use alloy_primitives::B256;

mod mock;
mod rpc;

pub use mock::MockBlobReader;
pub use rpc::BeaconBlobReader;

pub type Blob = c_kzg::Blob;

#[derive(Debug, thiserror::Error)]
pub enum BlobError {}

pub type Result<T, E = BlobError> = std::result::Result<T, E>;

#[async_trait::async_trait]
pub trait BlobReader: Send + Sync {
    async fn initialize(&self) -> Result<()>;
    async fn get_blobs(&self, block_hash: B256, versioned_hashes: &[B256]) -> Result<Vec<Blob>>;
}
