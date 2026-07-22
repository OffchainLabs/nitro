use std::sync::OnceLock;

use alloy_primitives::B256;
use alloy_rpc_types_beacon::{config::SpecResponse, genesis::GenesisResponse};
use reqwest::{Url, header::AUTHORIZATION};
use serde::de::DeserializeOwned;

use crate::{Blob, BlobError, BlobReader, Result};

mod config;

pub use config::BeaconBlobReaderConfig;

const GENESIS_ENDPOINT: &str = "/eth/v1/beacon/genesis";
const SPEC_ENDPOINT: &str = "/eth/v1/config/spec";

#[derive(Debug)]
pub struct BeaconBlobReader {
    /// Configuration options for reading blobs from the beacon.
    config: BeaconBlobReaderConfig,
    /// HTTP client for making the RPC requests.
    client: reqwest::Client,
    /// Timing info loaded from the beacon in [`Self::initialize`].
    slot_clock: OnceLock<SlotClock>,
}

impl BeaconBlobReader {
    pub fn new(config: BeaconBlobReaderConfig) -> Self {
        Self {
            config,
            client: reqwest::Client::new(),
            slot_clock: OnceLock::new(),
        }
    }

    /// Send a GET request to the beacon, deserializing the result.
    async fn beacon_request<T: DeserializeOwned>(&self, path: &str) -> Result<T> {
        let primary = match self.fetch(&self.config.beacon_url, path).await {
            Ok(resp) => return Ok(resp.json::<T>().await?),
            Err(err) => Box::new(err),
        };
        let secondary = match &self.config.secondary_beacon_url {
            Some(url) => match self.fetch(url, path).await {
                Ok(resp) => return Ok(resp.json::<T>().await?),
                Err(err) => Some(Box::new(err)),
            },
            None => None,
        };
        Err(BlobError::Request { primary, secondary })
    }

    /// Issues one GET to `base` + `path`, returning the response only on 2xx.
    async fn fetch(&self, base: &Url, path: &str) -> Result<reqwest::Response> {
        let mut url = base.clone();
        url.set_path(path);
        let mut req = self.client.get(url.clone());
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
}

#[async_trait::async_trait]
impl BlobReader for BeaconBlobReader {
    async fn initialize(&self) -> Result<()> {
        let genesis: GenesisResponse = self.beacon_request(GENESIS_ENDPOINT).await?;
        let spec: SpecResponse = self.beacon_request(SPEC_ENDPOINT).await?;

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
        todo!()
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
