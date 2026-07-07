use std::collections::HashMap;

use alloy_primitives::B256;
use alloy_transport::{RpcError, TransportErrorKind};

/// Something went wrong while reading from the DA provider.
#[derive(Debug, thiserror::Error)]
pub enum DaError {
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
}

/// Return type of [`DAReader`].
pub type Result<T, E = DaError> = std::result::Result<T, E>;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct PreimageType(pub u8);

pub type Payload = Vec<u8>;
pub type Preimages = HashMap<PreimageType, HashMap<B256, Vec<u8>>>;

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
