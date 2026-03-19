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
#[derive(Clone, Debug, Default, PartialEq, Eq)]
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
        if req.has_delayed_msg {
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

/// The `Vec<u8>` is assumed to be compressed using `brotli`, and must be decompressed before use.
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

#[cfg(test)]
mod tests {
    use super::*;

    fn make_user_wasm(data: &[u8]) -> UserWasm {
        let compressed = brotli::compress(data, 1, 22, brotli::Dictionary::Empty).unwrap();
        UserWasm::try_from(compressed).unwrap()
    }

    fn make_request() -> ValidationRequest {
        let block_hash = Bytes32([1u8; 32]);
        let send_root = Bytes32([2u8; 32]);

        let keccak_map = HashMap::from_iter([(Bytes32([0xAA; 32]), vec![10, 20, 30])]);
        let preimages = HashMap::from_iter([(PreimageType::Keccak256, keccak_map)]);

        let target_map = HashMap::from_iter([(Bytes32([0xBB; 32]), make_user_wasm(&[0, 1, 2, 3]))]);
        let user_wasms = HashMap::from_iter([("host".to_string(), target_map)]);

        ValidationRequest {
            id: 42,
            has_delayed_msg: true,
            delayed_msg_nr: 7,
            preimages,
            batch_info: vec![
                BatchInfo {
                    number: 0,
                    data: vec![1, 2, 3],
                },
                BatchInfo {
                    number: 1,
                    data: vec![4, 5, 6],
                },
            ],
            delayed_msg: vec![9, 8, 7],
            start_state: GoGlobalState {
                block_hash,
                send_root,
                batch: 100,
                pos_in_batch: 200,
            },
            user_wasms,
            debug_chain: false,
            max_user_wasm_size: 0,
        }
    }

    #[test]
    fn from_request_populates_globals() {
        let req = make_request();
        let input = ValidationInput::from_request(&req, "host");

        assert_eq!(input.small_globals, [100, 200]);
        assert_eq!(input.large_globals[0], [1u8; 32]);
        assert_eq!(input.large_globals[1], [2u8; 32]);
    }

    #[test]
    fn from_request_populates_sequencer_messages() {
        let req = make_request();
        let input = ValidationInput::from_request(&req, "host");

        assert_eq!(input.sequencer_messages.len(), 2);
        assert_eq!(input.sequencer_messages[&0], vec![1, 2, 3]);
        assert_eq!(input.sequencer_messages[&1], vec![4, 5, 6]);
    }

    #[test]
    fn from_request_includes_delayed_message_when_present() {
        let req = make_request();
        let input = ValidationInput::from_request(&req, "host");

        assert_eq!(input.delayed_messages.len(), 1);
        assert_eq!(input.delayed_messages[&7], vec![9, 8, 7]);
    }

    #[test]
    fn from_request_skips_delayed_message_when_flag_false() {
        let mut req = make_request();
        req.has_delayed_msg = false;
        let input = ValidationInput::from_request(&req, "host");

        assert!(input.delayed_messages.is_empty());
    }

    #[test]
    fn from_request_populates_preimages() {
        let req = make_request();
        let input = ValidationInput::from_request(&req, "host");

        let keccak_map = input
            .preimages
            .get(&(PreimageType::Keccak256 as u8))
            .unwrap();
        assert_eq!(keccak_map.len(), 1);
        assert_eq!(keccak_map[&[0xAA; 32]], vec![10, 20, 30]);
    }

    #[test]
    fn from_request_populates_module_asms_for_matching_target() {
        let req = make_request();
        let input = ValidationInput::from_request(&req, "host");

        assert_eq!(input.module_asms.len(), 1);
        assert_eq!(input.module_asms[&[0xBB; 32]], vec![0, 1, 2, 3]);
    }

    #[test]
    fn from_request_returns_empty_module_asms_for_unknown_target() {
        let req = make_request();
        let input = ValidationInput::from_request(&req, "nonexistent");

        assert!(input.module_asms.is_empty());
    }
}
