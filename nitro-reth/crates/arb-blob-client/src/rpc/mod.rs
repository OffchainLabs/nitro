use std::{
    fmt,
    sync::{Arc, OnceLock},
};

use alloy_primitives::B256;
use alloy_rpc_types_beacon::{
    config::SpecResponse, genesis::GenesisResponse, sidecar::GetBlobsResponse,
};
use arb_parent_chain_client::ParentChainReader;
use reqwest::{Url, header::AUTHORIZATION};
use serde::de::DeserializeOwned;

use crate::{Blob, BlobError, BlobReader, Result};

mod config;
mod kzg;

pub use config::BeaconBlobReaderConfig;

const GENESIS_ENDPOINT: &str = "/eth/v1/beacon/genesis";
const SPEC_ENDPOINT: &str = "/eth/v1/config/spec";

pub struct BeaconBlobReader {
    /// Configuration options for reading blobs from the beacon.
    config: BeaconBlobReaderConfig,
    /// HTTP client for making the RPC requests.
    client: reqwest::Client,
    /// Parent chain RPC
    parent_chain: Arc<dyn ParentChainReader>,
    /// Timing info loaded from the beacon in [`Self::initialize`].
    slot_clock: OnceLock<SlotClock>,
}

impl BeaconBlobReader {
    pub fn new(config: BeaconBlobReaderConfig, parent_chain: Arc<dyn ParentChainReader>) -> Self {
        Self {
            config,
            parent_chain,
            client: reqwest::Client::new(),
            slot_clock: OnceLock::new(),
        }
    }

    /// Send a GET request to the beacon, deserializing the result.
    async fn beacon_request<T: DeserializeOwned>(
        &self,
        path: &str,
        query: &[(&str, String)],
    ) -> Result<T> {
        let primary = match self.fetch(&self.config.beacon_url, path, query).await {
            Ok(resp) => return Ok(resp.json::<T>().await?),
            Err(err) => Box::new(err),
        };
        let secondary = match &self.config.secondary_beacon_url {
            Some(url) => match self.fetch(url, path, query).await {
                Ok(resp) => return Ok(resp.json::<T>().await?),
                Err(err) => Some(Box::new(err)),
            },
            None => None,
        };
        Err(BlobError::Request { primary, secondary })
    }

    /// Issues one GET to `base` + `path`, returning the response only on 2xx.
    async fn fetch(
        &self,
        base: &Url,
        path: &str,
        query: &[(&str, String)],
    ) -> Result<reqwest::Response> {
        let mut url = base.clone();
        url.set_path(path);
        let mut req = self.client.get(url.clone()).query(query);
        if let Some(auth) = &self.config.authorization {
            req = req.header(AUTHORIZATION, auth);
        }

        let resp = req.send().await?;
        if !resp.status().is_success() {
            return Err(BlobError::StatusNotOk {
                url,
                status: resp.status(),
            });
        }
        Ok(resp)
    }

    async fn get_blobs_by_slot(&self, slot: u64, versioned_hashes: &[B256]) -> Result<Vec<Blob>> {
        let path = format!("/eth/v1/beacon/blobs/{slot}");
        let query: Vec<(&str, String)> = versioned_hashes
            .iter()
            .map(|h| ("versioned_hashes", h.to_string()))
            .collect();

        let resp: GetBlobsResponse = self.beacon_request(&path, &query).await?;

        if !versioned_hashes.is_empty() && resp.data.len() != versioned_hashes.len() {
            return Err(BlobError::BlobCountMismatch {
                slot,
                expected: versioned_hashes.len(),
                got: resp.data.len(),
            });
        }

        resp.data
            .into_iter()
            .enumerate()
            .map(|(i, blob)| {
                if !self.config.skip_blob_proof_verification
                    && let Some(&expected) = versioned_hashes.get(i)
                {
                    let got = kzg::blob_to_versioned_hash(&blob)?;
                    if got != expected {
                        return Err(BlobError::VersionedHashMismatch {
                            index: i,
                            expected,
                            got,
                        });
                    }
                }
                Ok(blob)
            })
            .collect()
    }
}

impl fmt::Debug for BeaconBlobReader {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        f.debug_struct("BeaconBlobReader")
            .field("config", &self.config)
            .field("slot_clock", &self.slot_clock.get())
            .finish_non_exhaustive()
    }
}

#[async_trait::async_trait]
impl BlobReader for BeaconBlobReader {
    async fn initialize(&self) -> Result<()> {
        let genesis: GenesisResponse = self.beacon_request(GENESIS_ENDPOINT, &[]).await?;
        let spec: SpecResponse = self.beacon_request(SPEC_ENDPOINT, &[]).await?;

        let genesis_time = genesis.data.genesis_time;
        let seconds_per_slot: u64 = spec
            .data
            .get("SECONDS_PER_SLOT")
            .ok_or(BlobError::MissingSpecValue("SECONDS_PER_SLOT"))?
            .parse()
            .map_err(BlobError::InvalidSecondsPerSlot)?;

        if seconds_per_slot == 0 {
            return Err(BlobError::ZeroSecondsPerSlot);
        }

        let _ = self.slot_clock.set(SlotClock {
            genesis_time,
            seconds_per_slot,
        });
        Ok(())
    }

    async fn get_blobs(&self, block_hash: B256, versioned_hashes: &[B256]) -> Result<Vec<Blob>> {
        let slot_clock = self.slot_clock.get().ok_or(BlobError::NotInitialized)?;
        let header = self
            .parent_chain
            .header_by_hash(block_hash)
            .await?
            .ok_or(BlobError::BlockNotFound(block_hash))?;
        let slot = slot_clock.slot_for(header.timestamp);
        self.get_blobs_by_slot(slot, versioned_hashes).await
    }
}

/// Beacon-chain timing constants, fetched once by [`BeaconBlobReader::initialize`].
///
/// Together `genesis_time` and `seconds_per_slot` define the mapping from a
/// wall-clock timestamp to a beacon slot, which is how a parent-chain block
/// (identified by its timestamp) is located among the beacon node's blobs.
#[derive(Debug, Clone, Copy)]
struct SlotClock {
    /// Unix time of the beacon chain's genesis slot.
    genesis_time: u64,
    /// Slot duration in seconds (`SECONDS_PER_SLOT` from the beacon spec).
    seconds_per_slot: u64,
}

impl SlotClock {
    /// Returns the slot containing `timestamp`.
    ///
    /// Mirrors Nitro: `slot = (header.Time - genesisTime) / secondsPerSlot`.
    fn slot_for(&self, timestamp: u64) -> u64 {
        (timestamp - self.genesis_time) / self.seconds_per_slot
    }
}
