use alloy_primitives::B256;
use reqwest::{StatusCode, Url};

mod mock;
mod rpc;

pub use mock::MockBlobReader;
pub use rpc::{BeaconBlobReader, BeaconBlobReaderConfig};

pub type Blob = alloy_eips::eip4844::Blob;
pub type Result<T, E = BlobError> = std::result::Result<T, E>;

#[async_trait::async_trait]
pub trait BlobReader: Send + Sync {
    async fn initialize(&self) -> Result<()>;
    async fn get_blobs(&self, block_hash: B256, versioned_hashes: &[B256]) -> Result<Vec<Blob>>;
}

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

    #[error(transparent)]
    ParentChain(#[from] arb_parent_chain_client::ParentChainError),

    #[error(transparent)]
    Kzg(#[from] c_kzg::Error),

    #[error("beacon spec has invalid SECONDS_PER_SLOT: {0}")]
    InvalidSecondsPerSlot(std::num::ParseIntError),

    #[error("beacon spec is missing {0}")]
    MissingSpecValue(&'static str),

    #[error("beacon reported SECONDS_PER_SLOT of zero")]
    ZeroSecondsPerSlot,

    #[error("blob client is not initialized")]
    NotInitialized,

    #[error("no parent-chain block found for hash {0}")]
    BlockNotFound(B256),

    #[error("expected {expected} blobs for slot {slot} but got {got}")]
    BlobCountMismatch {
        slot: u64,
        expected: usize,
        got: usize,
    },

    #[error("blob {index} versioned hash mismatch: expected {expected}, got {got}")]
    VersionedHashMismatch {
        index: usize,
        expected: B256,
        got: B256,
    },
}
