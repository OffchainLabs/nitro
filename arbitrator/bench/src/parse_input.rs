use arbutil::Bytes32;
use serde::{Deserialize, Serialize};
use serde_json;
use serde_with::base64::Base64;
use serde_with::As;
use serde_with::DisplayFromStr;
use std::{
    collections::HashMap,
    io::{self, BufRead},
};

mod prefixed_hex {
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(bytes: &Vec<u8>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_str(&format!("0x{}", hex::encode(bytes)))
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Vec<u8>, D::Error>
    where
        D: Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        if s.starts_with("0x") {
            hex::decode(&s[2..]).map_err(serde::de::Error::custom)
        } else {
            Err(serde::de::Error::custom("missing 0x prefix"))
        }
    }
}

#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct PreimageMap(HashMap<Bytes32, Vec<u8>>);

#[derive(Debug, Clone, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct BatchInfo {
    pub number: u64,
    #[serde(with = "As::<Base64>")]
    pub data_b64: Vec<u8>,
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct StartState {
    #[serde(with = "prefixed_hex")]
    pub block_hash: Vec<u8>,
    #[serde(with = "prefixed_hex")]
    pub send_root: Vec<u8>,
    pub batch: u64,
    pub pos_in_batch: u64,
}

#[derive(Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct FileData {
    pub id: u64,
    pub has_delayed_msg: bool,
    pub delayed_msg_nr: u64,
    #[serde(with = "As::<HashMap<DisplayFromStr, HashMap<Base64, Base64>>>")]
    pub preimages_b64: HashMap<u32, HashMap<Bytes32, Vec<u8>>>,
    pub batch_info: Vec<BatchInfo>,
    #[serde(with = "As::<Base64>")]
    pub delayed_msg_b64: Vec<u8>,
    pub start_state: StartState,
}

impl FileData {
    pub fn from_reader<R: BufRead>(mut reader: R) -> io::Result<Self> {
        let data = serde_json::from_reader(&mut reader)?;
        return Ok(data);
    }
}
