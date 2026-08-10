use std::{
    fmt,
    sync::{Arc, OnceLock},
    time::{SystemTime, UNIX_EPOCH},
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
#[cfg(test)]
mod http_tests;
mod kzg;

pub use config::BeaconBlobReaderConfig;

const GENESIS_ENDPOINT: &str = "/eth/v1/beacon/genesis";
const SPEC_ENDPOINT: &str = "/eth/v1/config/spec";

/// Builds a request URL by joining `path` onto the base URL's existing path,
/// rather than replacing it, so a beacon URL carrying a prefix (e.g. an API
/// key or proxy route) is preserved. Mirrors Nitro's
/// `path.Join(beaconUrl.Path, beaconPath)`.
fn beacon_url(base: &Url, path: &str) -> Url {
    let mut url = base.clone();
    url.set_path(&format!("{}{path}", url.path().trim_end_matches('/')));
    url
}

/// A [`BlobReader`] backed by a beacon node's HTTP API, porting Nitro's
/// `BlobClient`.
///
/// [`BeaconBlobReader::initialize`] loads the beacon's genesis time and slot
/// duration once; [`BeaconBlobReader::get_blobs`] then maps a parent-chain
/// block to its slot and fetches that slot's blobs. A primary and optional
/// secondary beacon URL are tried in order for every request.
///
/// Note: genesis parsing is slightly stricter than Nitro's, which reads only
/// `genesis_time`. The underlying alloy type also requires
/// `genesis_validators_root` and `genesis_fork_version`, which standard beacon
/// nodes always return.
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
        let url = beacon_url(base, path);
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

    async fn get_blobs_by_slot(
        &self,
        slot_clock: &SlotClock,
        slot: u64,
        versioned_hashes: &[B256],
    ) -> Result<Vec<Blob>> {
        let path = format!("/eth/v1/beacon/blobs/{slot}");
        let query: Vec<(&str, String)> = versioned_hashes
            .iter()
            .map(|h| ("versioned_hashes", h.to_string()))
            .collect();

        let resp: GetBlobsResponse = match self.beacon_request(&path, &query).await {
            Ok(resp) => resp,
            Err(err) => {
                // A very old slot is likely past the beacon node's retention
                // window, which needs an archive endpoint. Surface that
                // distinctly so the operator isn't left guessing. Mirrors
                // Nitro's `roughAgeOfSlot` check in `getBlobs`.
                let age = slot_clock.slot_age(slot, unix_now());
                if age > slot_clock.blob_retention_secs() {
                    return Err(BlobError::BlobsExpired {
                        slot,
                        age,
                        source: Box::new(err),
                    });
                }
                return Err(err);
            }
        };

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
                if let Some(&expected) = versioned_hashes.get(i) {
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

        self.slot_clock
            .set(SlotClock {
                genesis_time,
                seconds_per_slot,
            })
            .map_err(|_| BlobError::AlreadyInitialized)?;
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
        self.get_blobs_by_slot(slot_clock, slot, versioned_hashes)
            .await
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
    /// The subtraction saturates so a header predating beacon genesis maps to
    /// slot 0 rather than panicking in debug builds (Go wraps silently here).
    fn slot_for(&self, timestamp: u64) -> u64 {
        timestamp.saturating_sub(self.genesis_time) / self.seconds_per_slot
    }

    /// Unix time at which `slot` begins.
    fn slot_start(&self, slot: u64) -> u64 {
        self.genesis_time + slot * self.seconds_per_slot
    }

    /// Rough age in seconds of `slot` relative to `now` (unix seconds); zero for
    /// slots at or in the future. Mirrors Nitro's `roughAgeOfSlot`, using a
    /// saturating subtraction so a future slot can't wrap to a huge age.
    fn slot_age(&self, slot: u64, now: u64) -> u64 {
        now.saturating_sub(self.slot_start(slot))
    }

    /// Age past which a non-archive beacon node is no longer expected to serve a
    /// slot's blobs. Mirrors Nitro's `secondsPerSlot * 32 * 4096`.
    fn blob_retention_secs(&self) -> u64 {
        self.seconds_per_slot * 32 * 4096
    }
}

/// Current unix time in seconds, saturating to 0 if the clock predates the epoch.
fn unix_now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn clock() -> SlotClock {
        SlotClock {
            genesis_time: 1_000,
            seconds_per_slot: 12,
        }
    }

    #[test]
    fn slot_at_genesis_is_zero() {
        assert_eq!(clock().slot_for(1_000), 0);
    }

    #[test]
    fn slot_rounds_down_within_a_slot() {
        // Slot 1 starts at genesis + 12s; timestamps before that are still slot 0.
        assert_eq!(clock().slot_for(1_011), 0);
        assert_eq!(clock().slot_for(1_012), 1);
        assert_eq!(clock().slot_for(1_023), 1);
    }

    #[test]
    fn slot_advances_across_many_slots() {
        assert_eq!(clock().slot_for(1_000 + 12 * 42), 42);
    }

    #[test]
    fn slot_before_genesis_saturates_to_zero() {
        // A timestamp earlier than genesis must not panic in debug builds.
        assert_eq!(clock().slot_for(500), 0);
    }

    #[test]
    fn slot_age_measures_seconds_since_slot_start() {
        // Slot 10 starts at genesis (1_000) + 10*12 = 1_120; at now=1_150 it is 30s old.
        assert_eq!(clock().slot_age(10, 1_150), 30);
    }

    #[test]
    fn slot_age_saturates_to_zero_for_future_slots() {
        // now precedes the slot start: age is 0, not a wrapped-around huge value.
        assert_eq!(clock().slot_age(100, 1_000), 0);
    }

    #[test]
    fn blob_retention_matches_nitro_formula() {
        assert_eq!(clock().blob_retention_secs(), 12 * 32 * 4096);
    }

    #[test]
    fn beacon_url_appends_to_bare_host() {
        let base = Url::parse("https://beacon.example").unwrap();
        let url = beacon_url(&base, GENESIS_ENDPOINT);
        assert_eq!(url.as_str(), "https://beacon.example/eth/v1/beacon/genesis");
    }

    #[test]
    fn beacon_url_preserves_base_path_prefix() {
        // A prefix (proxy route, API key, ...) on the configured URL must be
        // kept, not clobbered.
        let base = Url::parse("https://beacon.example/node/v2").unwrap();
        let url = beacon_url(&base, GENESIS_ENDPOINT);
        assert_eq!(
            url.as_str(),
            "https://beacon.example/node/v2/eth/v1/beacon/genesis"
        );
    }

    #[test]
    fn beacon_url_avoids_double_slash_on_trailing_base() {
        let base = Url::parse("https://beacon.example/node/").unwrap();
        let url = beacon_url(&base, GENESIS_ENDPOINT);
        assert_eq!(
            url.as_str(),
            "https://beacon.example/node/eth/v1/beacon/genesis"
        );
    }
}
