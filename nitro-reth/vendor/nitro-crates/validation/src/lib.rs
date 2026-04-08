// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use arbutil::{Bytes32, PreimageType};
use brotli::BrotliStatus;
use serde::{Deserialize, Serialize};
use serde_with::{base64::Base64, As, DisplayFromStr};
use std::{
    collections::HashMap,
    io::{self, BufRead},
};

pub mod transfer;

pub type PreimageMap = HashMap<PreimageType, HashMap<Bytes32, Vec<u8>>>;

pub const TARGET_ARM_64: &str = "arm64";
pub const TARGET_AMD_64: &str = "amd64";
pub const TARGET_HOST: &str = "host";

/// Counterpart to Go `rawdb.LocalTarget()`.
pub fn local_target() -> &'static str {
    if cfg!(all(target_os = "linux", target_arch = "aarch64")) {
        TARGET_ARM_64
    } else if cfg!(all(target_os = "linux", target_arch = "x86_64")) {
        TARGET_AMD_64
    } else {
        TARGET_HOST
    }
}

/// Counterpart to Go `validator.GoGlobalState`.
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq, Deserialize)]
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
#[derive(Clone, Debug, PartialEq, Eq)]
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
#[derive(Clone, Debug, Default, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct ValidationInput {
    pub id: u64,
    pub has_delayed_msg: bool,
    pub delayed_msg_nr: u64,
    #[serde(
        rename = "PreimagesB64",
        with = "As::<HashMap<DisplayFromStr, HashMap<Base64, Base64>>>"
    )]
    pub preimages: PreimageMap,
    pub batch_info: Vec<BatchInfo>,
    #[serde(rename = "DelayedMsgB64", with = "As::<Base64>")]
    pub delayed_msg: Vec<u8>,
    pub start_state: GoGlobalState,
    #[serde(with = "As::<HashMap<DisplayFromStr, HashMap<DisplayFromStr, Base64>>>")]
    pub user_wasms: HashMap<String, HashMap<Bytes32, UserWasm>>,
    pub debug_chain: bool,
    #[serde(rename = "max-user-wasmSize", default)]
    pub max_user_wasm_size: u64,
}

impl ValidationInput {
    pub fn from_reader<R: BufRead>(mut reader: R) -> io::Result<Self> {
        Ok(serde_json::from_reader(&mut reader)?)
    }

    pub fn delayed_msg(&self) -> Option<BatchInfo> {
        self.has_delayed_msg.then(|| BatchInfo {
            number: self.delayed_msg_nr,
            data: self.delayed_msg.clone(),
        })
    }
}
