use arbutil::{Bytes32, PreimageType};
use brotli::BrotliStatus;
use serde::{Deserialize, Serialize};
use serde_with::{base64::Base64, As, DisplayFromStr, TryFromInto};
use std::{
    collections::HashMap,
    io::{self, BufRead},
};

/// Counterpart to Go `validator.GoGlobalState`.
#[derive(Clone, Debug, Serialize, Deserialize, Default)]
#[serde(rename_all = "PascalCase")]
pub struct GoGlobalState {
    #[serde(with = "As::<DisplayFromStr>")]
    pub block_hash: Bytes32,
    #[serde(with = "As::<DisplayFromStr>")]
    pub send_root: Bytes32,
    pub batch: u64,
    pub pos_in_batch: u64,
}

/// Counterpart to Go `validator.server_api.BatchInfoJson`.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct BatchInfo {
    pub number: u64,
    #[serde(rename = "DataB64", with = "As::<Base64>")]
    pub data: Vec<u8>,
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
impl TryFrom<Vec<u8>> for UserWasm {
    type Error = BrotliStatus;

    fn try_from(data: Vec<u8>) -> Result<Self, Self::Error> {
        Ok(Self(brotli::decompress(&data, brotli::Dictionary::Empty)?))
    }
}

/// Counterpart to Go `validator.server_api.InputJSON`.
#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct ValidationInput {
    pub id: u64,
    pub has_delayed_msg: bool,
    pub delayed_msg_nr: u64,
    #[serde(
        rename = "PreimagesB64",
        with = "As::<HashMap<TryFromInto<u8>, HashMap<Base64, Base64>>>"
    )]
    pub preimages: HashMap<PreimageType, HashMap<Bytes32, Vec<u8>>>,
    pub batch_info: Vec<BatchInfo>,
    #[serde(rename = "DelayedMsgB64", with = "As::<Base64>")]
    pub delayed_msg: Vec<u8>,
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
