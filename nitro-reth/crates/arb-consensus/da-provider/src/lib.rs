//! The Arbitrum data availability (DA) provider abstraction.
//!
//! Provides [`DaReader`], a small trait for recovering batch payloads and
//! collecting preimages, plus its supporting types ([`DaError`], [`Payload`],
//! [`Preimages`], [`PreimageType`]) and a [`DaReaderRegistry`].
//!
//! This only reads data; it doesn't interpret any of it. Code that needs to do
//! deterministic, no-IO processing is expected to pull what it needs through
//! this reader first and then work on the results.
//!
//! Dependency-light and wasm-compatible so `arb-mel`'s message extraction can
//! depend on it. The live JSON-RPC reader lives in the sibling
//! `arb-da-provider-rpc-client` crate, keeping the networking stack out of the
//! wasm build.

use std::{collections::HashMap, error::Error};

use alloy_primitives::B256;

mod mock;
mod registry;

pub use mock::MockDaReader;
pub use registry::{DaReaderRegistry, DaReaderSource};

/// Something went wrong while reading from the DA provider.
#[derive(Debug, thiserror::Error)]
pub enum DaError {
    /// A base64-encoded byte field in the response could not be decoded.
    #[error(transparent)]
    Base64(#[from] base64::DecodeError),
    /// The provider returned a preimage type this client doesn't recognise.
    #[error("unknown preimage type: {0}")]
    UnknownPreimageType(u8),
    /// The sequencer message isn't in the shape the DA provider registered for
    /// its header byte expects, so no payload can be recovered from it.
    ///
    /// Permanent: the batch data is fixed on the parent chain, so reading again
    /// yields the same bytes and fails the same way.
    #[error("malformed sequencer message: {0}")]
    MalformedSequencerMessage(String),
    /// A DA provider failed while recovering a batch payload, e.g. while
    /// fetching the underlying data.
    ///
    /// The source is the provider's own error (`BlobError` for the blob reader),
    /// kept as a trait object because this crate sits below the providers that
    /// implement [`DaReader`]. Callers needing more than the message can walk
    /// [`Error::source`] or downcast the boxed error to the provider's concrete
    /// type. Build one with [`DaError::provider`].
    #[error("data availability provider error: {0}")]
    Provider(#[source] Box<dyn Error + Send + Sync + 'static>),
}

impl DaError {
    /// Wraps a DA provider's own error as [`DaError::Provider`].
    pub fn provider<E>(err: E) -> Self
    where
        E: Error + Send + Sync + 'static,
    {
        Self::Provider(Box::new(err))
    }
}

/// Return type of [`DaReader`].
pub type Result<T, E = DaError> = std::result::Result<T, E>;

/// Identifies the hashing scheme a preimage is keyed under.
///
/// Mirrors Nitro's `arbutil.PreimageType`. The variant discriminants are the
/// on-the-wire values.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
#[repr(u8)]
pub enum PreimageType {
    Keccak256 = 0,
    Sha2_256 = 1,
    EthVersionedHash = 2,
    DaCertificate = 3,
}

impl TryFrom<u8> for PreimageType {
    type Error = DaError;

    fn try_from(value: u8) -> Result<Self> {
        match value {
            0 => Ok(Self::Keccak256),
            1 => Ok(Self::Sha2_256),
            2 => Ok(Self::EthVersionedHash),
            3 => Ok(Self::DaCertificate),
            other => Err(DaError::UnknownPreimageType(other)),
        }
    }
}

pub type Payload = Vec<u8>;
pub type Preimages = HashMap<PreimageType, HashMap<B256, Vec<u8>>>;

/// Read payloads and preimages from the DA provider.
///
/// Each method is a wrapper around one JSON-RPC call.
#[async_trait::async_trait]
pub trait DaReader: Send + Sync {
    /// Fetches the underlying payload from the DA provider given the batch header info.
    async fn recover_payload(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Payload>;

    /// Collects preimages from the DA provider given the batch header info.
    async fn collect_preimages(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Preimages>;

    /// Fetches the underlying payload and collects preimages given the batch header info.
    async fn recover_payload_and_preimages(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<(Payload, Preimages)>;
}
