//! Configuration for the [`crate::BeaconBlobReader`].

use std::fmt;

use reqwest::Url;

pub struct BeaconBlobReaderConfig {
    pub beacon_url: Url,
    pub secondary_beacon_url: Option<Url>,
    pub authorization: Option<String>,
    pub skip_blob_proof_verification: bool,
}

/// Custom Debug implementation to avoid leaking `authorization` field.
impl fmt::Debug for BeaconBlobReaderConfig {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        f.debug_struct("BeaconBlobReaderConfig")
            .field("beacon_url", &self.beacon_url)
            .field("secondary_beacon_url", &self.secondary_beacon_url)
            .field(
                "authorization",
                &self.authorization.as_ref().map(|_| "<REDACTED>"),
            )
            .field(
                "skip_blob_proof_verification",
                &self.skip_blob_proof_verification,
            )
            .finish()
    }
}
