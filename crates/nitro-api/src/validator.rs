//! Go `validator` module (../../validator/) related types.

use arbutil::Bytes32;
use serde::Deserialize;
use serde_with::base64::Base64;
use serde_with::As;
use serde_with::DisplayFromStr;
use std::collections::HashMap;
use std::io;
use std::io::BufRead;

/// Counterpart to Go `validator.GoGlobalState`.
#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct GoGlobalState {
    #[serde(with = "prefixed_hex")] //TODO
    pub block_hash: Vec<u8>,
    #[serde(with = "prefixed_hex")]
    pub send_root: Vec<u8>,
    pub batch: u64,
    pub pos_in_batch: u64,
}

// Counterpart to Go `validator.server_api.BatchInfoJson`.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct BatchInfo {
    pub number: u64,
    #[serde(with = "As::<Base64>")]
    pub data_b64: Vec<u8>,
}

/// `UserWasm` is a wrapper around `Vec<u8>`
///
/// It is useful for decompressing a `brotli`-compressed wasm module.
///
/// Note: The wrapped `Vec<u8>` is already `Base64` decoded before
/// `from(Vec<u8>)` is called by `serde`.
#[derive(Clone, Debug)]
pub struct UserWasm(Vec<u8>);

impl UserWasm {
    /// `as_vec` returns the decompressed wasm module as a `Vec<u8>`
    pub fn as_vec(&self) -> Vec<u8> {
        self.0.clone()
    }
}

impl AsRef<[u8]> for UserWasm {
    fn as_ref(&self) -> &[u8] {
        &self.0
    }
}

/// The `Vec<u8>` is assumed to be compressed using `brotli`, and must be decompressed before use.
impl From<Vec<u8>> for UserWasm {
    fn from(data: Vec<u8>) -> Self {
        Self(brotli::decompress(&data, brotli::Dictionary::Empty).unwrap())
    }
}

/// Counterpart to Go `validator.server_api.InputJSON`.
#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct ValidationInput {
    pub id: u64,
    pub has_delayed_msg: bool,
    pub delayed_msg_nr: u64,
    #[serde(with = "As::<HashMap<DisplayFromStr, HashMap<Base64, Base64>>>")]
    pub preimages_b64: HashMap<u32, HashMap<Bytes32, Vec<u8>>>,
    pub batch_info: Vec<BatchInfo>,
    #[serde(with = "As::<Base64>")]
    pub delayed_msg_b64: Vec<u8>,
    pub start_state: GoGlobalState,
    #[serde(with = "As::<HashMap<DisplayFromStr, HashMap<DisplayFromStr, Base64>>>")]
    pub user_wasms: HashMap<String, HashMap<Bytes32, UserWasm>>,
    pub debug_chain: bool,
    #[serde(rename = "max-user-wasmSize")]
    pub max_user_wasm_size: u64,
}

impl ValidationInput {
    pub fn from_reader<R: BufRead>(mut reader: R) -> io::Result<Self> {
        Ok(serde_json::from_reader(&mut reader)?)
    }
}

/// `prefixed_hex` deserializes hex strings which are prefixed with `0x`
///
/// The default hex deserializer does not support prefixed hex strings.
///
/// It is an error to use this deserializer on a string that does not
/// begin with `0x`.
mod prefixed_hex {
    use serde::{self, Deserialize, Deserializer};

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Vec<u8>, D::Error>
    where
        D: Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        if let Some(s) = s.strip_prefix("0x") {
            hex::decode(s).map_err(serde::de::Error::custom)
        } else {
            Err(serde::de::Error::custom("missing 0x prefix"))
        }
    }
}
