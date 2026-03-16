// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use arbutil::{Bytes32, PreimageType};
use serde::{Deserialize, Serialize};
use serde_with::{base64::Base64, As, DisplayFromStr};
use std::{
    collections::{BTreeMap, HashMap},
    io::{self, BufRead},
};

pub mod transfer;

pub type Inbox = BTreeMap<u64, Vec<u8>>;
pub type Preimages = BTreeMap<u8, BTreeMap<[u8; 32], Vec<u8>>>;

/// The runtime data needed by any machine (JIT, SP1, Prover) to execute
/// a single block validation. Extracted from a `ValidationRequest` by
/// selecting a target architecture and stripping request metadata.
#[derive(Clone, Debug, Default)]
#[cfg_attr(
    feature = "rkyv",
    derive(rkyv::Archive, rkyv::Deserialize, rkyv::Serialize)
)]
pub struct ValidationInput {
    pub small_globals: [u64; 2],
    pub large_globals: [[u8; 32]; 2],
    pub preimages: Preimages,
    pub sequencer_messages: Inbox,
    pub delayed_messages: Inbox,
    pub module_asms: HashMap<[u8; 32], Vec<u8>>,
}

impl ValidationInput {
    /// Extract runtime data from a request for the given target architecture.
    pub fn from_request(req: &ValidationRequest, target: &str) -> Self {
        let mut sequencer_messages = Inbox::new();
        for batch in &req.batch_info {
            sequencer_messages.insert(batch.number, batch.data.clone());
        }

        let mut delayed_messages = Inbox::new();
        if req.delayed_msg_nr != 0 && !req.delayed_msg.is_empty() {
            delayed_messages.insert(req.delayed_msg_nr, req.delayed_msg.clone());
        }

        let mut preimages = Preimages::new();
        for (preimage_ty, inner_map) in &req.preimages {
            let map = preimages.entry(*preimage_ty as u8).or_default();
            for (hash, preimage) in inner_map {
                map.insert(**hash, preimage.clone());
            }
        }

        let mut module_asms = HashMap::new();
        if let Some(user_wasms) = req.user_wasms.get(target) {
            for (module_hash, wasm) in user_wasms {
                module_asms.insert(**module_hash, wasm.as_vec());
            }
        }

        Self {
            small_globals: [req.start_state.batch, req.start_state.pos_in_batch],
            large_globals: [req.start_state.block_hash.0, req.start_state.send_root.0],
            preimages,
            sequencer_messages,
            delayed_messages,
            module_asms,
        }
    }

    #[cfg(feature = "rkyv")]
    pub fn from_reader<R: io::Read>(mut reader: R) -> Result<Self, String> {
        let mut s = Vec::new();
        reader
            .read_to_end(&mut s)
            .map_err(|e| format!("IO Error: {e:?}"))?;
        let archived = rkyv::access::<ArchivedValidationInput, rkyv::rancor::Error>(&s[..])
            .map_err(|e| format!("rkyv access error: {e:?}"))?;
        rkyv::deserialize::<ValidationInput, rkyv::rancor::Error>(archived)
            .map_err(|e| format!("rkyv deserialize error: {e:?}"))
    }
}

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

/// `UserWasm` is a wrapper around `Vec<u8>`. It contains `brotli`-decompressed wasm module.
///
/// Note: The wrapped `Vec<u8>` is already `Base64` decoded before `from(Vec<u8>)` is called by `serde`.
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

impl TryFrom<Vec<u8>> for UserWasm {
    type Error = brotli::BrotliStatus;

    fn try_from(data: Vec<u8>) -> Result<Self, Self::Error> {
        Ok(Self(brotli::decompress(&data, brotli::Dictionary::Empty)?))
    }
}

/// Counterpart to Go `validator.server_api.InputJSON`.
#[derive(Clone, Debug, Default, PartialEq, Eq, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct ValidationRequest {
    pub id: u64,
    pub has_delayed_msg: bool,
    pub delayed_msg_nr: u64,
    #[serde(
        rename = "PreimagesB64",
        with = "As::<HashMap<DisplayFromStr, HashMap<Base64, Base64>>>"
    )]
    pub preimages: HashMap<PreimageType, HashMap<Bytes32, Vec<u8>>>,
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

impl ValidationRequest {
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
