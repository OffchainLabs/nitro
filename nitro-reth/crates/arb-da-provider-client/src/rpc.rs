use std::{collections::HashMap, sync::OnceLock};

use alloy_primitives::{Bytes, B256, U64};
use alloy_rpc_client::RpcClient;
use base64::{engine::GeneralPurpose, Engine as _};
use serde::Deserialize;

use super::{DaReader, Payload, PreimageType, Preimages, Result};

/// A [`DaReader`] that talks to a DA provider over JSON-RPC.
#[derive(Debug)]
pub struct RpcDaReader {
    client: RpcClient,
}

impl From<RpcClient> for RpcDaReader {
    fn from(client: RpcClient) -> Self {
        Self { client }
    }
}

#[async_trait::async_trait]
impl DaReader for RpcDaReader {
    async fn recover_payload(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Payload> {
        let params = (
            U64::from(batch_num),
            batch_block_hash,
            Bytes::copy_from_slice(sequencer_msg),
        );
        let res: RpcPayloadResult = self
            .client
            .request("daprovider_recoverPayload", params)
            .await?;
        decode_payload(res.payload.as_deref())
    }

    async fn collect_preimages(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<Preimages> {
        let params = (
            U64::from(batch_num),
            batch_block_hash,
            Bytes::copy_from_slice(sequencer_msg),
        );
        let res: RpcPreimagesResult = self
            .client
            .request("daprovider_collectPreimages", params)
            .await?;
        decode_preimages(res.preimages)
    }

    async fn recover_payload_and_preimages(
        &self,
        batch_num: u64,
        batch_block_hash: B256,
        sequencer_msg: &[u8],
    ) -> Result<(Payload, Preimages)> {
        let params = (
            U64::from(batch_num),
            batch_block_hash,
            Bytes::copy_from_slice(sequencer_msg),
        );
        let res: RpcPayloadAndPreimagesResult = self
            .client
            .request("daprovider_recoverPayloadAndPreimages", params)
            .await?;
        Ok((
            decode_payload(res.payload.as_deref())?,
            decode_preimages(res.preimages)?,
        ))
    }
}

/// `daprovider_recoverPayload` response.
#[derive(Debug, Deserialize)]
struct RpcPayloadResult {
    #[serde(rename = "Payload", default)]
    payload: Option<String>,
}

/// `daprovider_collectPreimages` response.
#[derive(Debug, Deserialize)]
struct RpcPreimagesResult {
    #[serde(rename = "Preimages", default)]
    preimages: Option<RpcPreimages>,
}

/// `daprovider_recoverPayloadAndPreimages` response.
#[derive(Debug, Deserialize)]
struct RpcPayloadAndPreimagesResult {
    #[serde(rename = "Payload", default)]
    payload: Option<String>,
    #[serde(rename = "Preimages", default)]
    preimages: Option<RpcPreimages>,
}

/// The wire form of [`Preimages`]: preimage type to a map of hash to base64-encoded
/// bytes. The outer key is a preimage type, the inner key is a hash, and each value
/// is base64-encoded bytes. `serde_json` parses the integer and hash map keys; only
/// the base64 values need manual decoding.
type RpcPreimages = HashMap<u8, HashMap<B256, String>>;

/// Decodes a base64 payload, treating a missing/empty field as an empty payload.
fn decode_payload(payload: Option<&str>) -> Result<Payload> {
    match payload {
        Some(s) if !s.is_empty() => Ok(base64_engine().decode(s)?),
        _ => Ok(Vec::new()),
    }
}

/// Converts the wire preimages map into the domain [`Preimages`], base64-decoding
/// each value.
fn decode_preimages(preimages: Option<RpcPreimages>) -> Result<Preimages> {
    let mut out = Preimages::new();
    for (ty, inner) in preimages.unwrap_or_default() {
        let mut decoded = HashMap::with_capacity(inner.len());
        for (hash, value) in inner {
            decoded.insert(hash, base64_engine().decode(value)?);
        }
        out.insert(PreimageType(ty), decoded);
    }
    Ok(out)
}

/// A base64 engine that tolerates the padding variations a DA provider might emit.
fn base64_engine() -> &'static GeneralPurpose {
    use base64::{
        alphabet,
        engine::{DecodePaddingMode, GeneralPurposeConfig},
    };

    static ENGINE: OnceLock<GeneralPurpose> = OnceLock::new();
    ENGINE.get_or_init(|| {
        let cfg = GeneralPurposeConfig::new()
            .with_decode_padding_mode(DecodePaddingMode::Indifferent)
            .with_decode_allow_trailing_bits(true);
        GeneralPurpose::new(&alphabet::STANDARD, cfg)
    })
}
