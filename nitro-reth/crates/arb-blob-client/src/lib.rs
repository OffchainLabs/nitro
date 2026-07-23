//! Fetches EIP-4844 blobs for MEL, porting Nitro's `BlobClient`
//! (`util/headerreader/blob_client.go`) to reth.
//!
//! [`BlobReader`] is the abstraction MEL reads blobs through. The default
//! implementation, [`BeaconBlobReader`], resolves a parent-chain block to a
//! beacon slot and fetches that slot's blobs from a beacon node over HTTP,
//! verifying each blob against its expected versioned hash. [`MockBlobReader`]
//! is an in-memory implementation for tests.

use alloy_primitives::B256;
use reqwest::{StatusCode, Url};

mod mock;
mod rpc;

pub use mock::MockBlobReader;
pub use rpc::{BeaconBlobReader, BeaconBlobReaderConfig};

pub type Blob = alloy_eips::eip4844::Blob;
pub type Result<T, E = BlobError> = std::result::Result<T, E>;

/// Reads EIP-4844 blobs by parent-chain block hash.
#[async_trait::async_trait]
pub trait BlobReader: Send + Sync {
    /// Loads any state the reader needs before serving requests.
    ///
    /// Must be called once, before [`BlobReader::get_blobs`]; calling
    /// `get_blobs` first fails with [`BlobError::NotInitialized`].
    async fn initialize(&self) -> Result<()>;

    /// Returns the blobs for the given parent-chain block, in the order of
    /// `versioned_hashes`.
    ///
    /// When `versioned_hashes` is non-empty, each returned blob is verified
    /// against its expected versioned hash and the count must match; an empty
    /// slice fetches without that verification.
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

    #[error(
        "no blobs returned for slot {slot} (~{age}s old); the slot is likely beyond the beacon node's retention window, an archive beacon endpoint is required (see https://docs.arbitrum.io/run-arbitrum-node/l1-ethereum-beacon-chain-rpc-providers)"
    )]
    BlobsExpired {
        slot: u64,
        age: u64,
        #[source]
        source: Box<BlobError>,
    },

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

    #[error("blob client is already initialized")]
    AlreadyInitialized,

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
