use alloy_primitives::B256;
use reqwest::{StatusCode, Url};

mod mock;
mod rpc;

pub use mock::MockBlobReader;
pub use rpc::{BeaconBlobReader, BeaconBlobReaderConfig};

pub type Blob = c_kzg::Blob;

#[derive(Debug, thiserror::Error)]
pub enum BlobError {
    #[error(transparent)]
    Http(#[from] reqwest::Error),

    #[error("all beacon endpoints failed (primary: {primary})")]
    Request {
        primary: Box<BlobError>,
        secondary: Option<Box<BlobError>>,
    },

    #[error("beacon request to {url} returned status {status}")]
    StatusNotOk { status: StatusCode, url: Url },

    #[error("beacon spec has invalid SECONDS_PER_SLOT: {0}")]
    InvalidSecondsPerSlot(std::num::ParseIntError),

    #[error("beacon spec is missing {0}")]
    MissingSpecValue(&'static str),

    #[error("beacon reported SECONDS_PER_SLOT of zero")]
    ZeroSecondsPerSlot,
}

pub type Result<T, E = BlobError> = std::result::Result<T, E>;

#[async_trait::async_trait]
pub trait BlobReader: Send + Sync {
    async fn initialize(&self) -> Result<()>;
    async fn get_blobs(&self, block_hash: B256, versioned_hashes: &[B256]) -> Result<Vec<Blob>>;
}
