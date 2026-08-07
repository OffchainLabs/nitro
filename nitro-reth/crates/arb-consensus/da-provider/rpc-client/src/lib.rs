//! A live JSON-RPC [`DaReader`] for an Arbitrum data availability provider.
//!
//! Split out of `arb-da-provider` so the networking stack (alloy RPC client,
//! reqwest, tokio) stays out of that crate's wasm-compatible core. Speaks the
//! `daprovider_*` methods and decodes their base64 payloads/preimages.

use std::{collections::HashMap, sync::OnceLock};

use alloy_primitives::{B256, Bytes, U64};
use alloy_rpc_client::RpcClient;
use arb_da_provider::{DaError, DaReader, Payload, PreimageType, Preimages, Result};
use base64::{Engine as _, engine::GeneralPurpose};
use serde::Deserialize;

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
            .await
            .map_err(DaError::provider)?;
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
            .await
            .map_err(DaError::provider)?;
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
            .await
            .map_err(DaError::provider)?;
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
        let ty = PreimageType::try_from(ty)?;
        let mut decoded = HashMap::with_capacity(inner.len());
        for (hash, value) in inner {
            decoded.insert(hash, base64_engine().decode(value)?);
        }
        out.insert(ty, decoded);
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

#[cfg(test)]
mod tests {
    use alloy_provider::mock::Asserter;
    use arb_da_provider::{DaError, DaReader};

    use super::*;

    /// The 4 raw bytes used across the fixtures below.
    const BYTES: [u8; 4] = [0xde, 0xad, 0xbe, 0xef];
    /// Standard base64 of [`BYTES`], with padding.
    const B64: &str = "3q2+7w==";
    /// Standard base64 of [`BYTES`], without the trailing `==` padding.
    const B64_UNPADDED: &str = "3q2+7w";

    /// A reader backed by a mock transport that replays the queued responses.
    fn reader(asserter: Asserter) -> RpcDaReader {
        RpcClient::mocked(asserter).into()
    }

    #[test]
    fn decode_payload_decodes_base64() {
        assert_eq!(decode_payload(Some(B64)).unwrap(), BYTES);
    }

    #[test]
    fn decode_payload_none_is_empty() {
        assert!(decode_payload(None).unwrap().is_empty());
    }

    #[test]
    fn decode_payload_empty_string_is_empty() {
        assert!(decode_payload(Some("")).unwrap().is_empty());
    }

    #[test]
    fn decode_payload_invalid_base64_errors() {
        let err = decode_payload(Some("invalid payload!")).unwrap_err();
        assert!(matches!(err, DaError::Base64(_)));
    }

    #[test]
    fn decode_payload_tolerates_missing_padding() {
        assert_eq!(decode_payload(Some(B64_UNPADDED)).unwrap(), BYTES);
    }

    #[test]
    fn decode_payload_tolerates_trailing_bits() {
        // "3q2+7x" differs from B64_UNPADDED only in the final char, which
        // carries a stray low bit that a strict engine would reject.
        assert_eq!(decode_payload(Some("3q2+7x")).unwrap(), BYTES);
    }

    #[test]
    fn decode_preimages_none_is_empty() {
        assert!(decode_preimages(None).unwrap().is_empty());
    }

    #[test]
    fn decode_preimages_decodes_values_across_types() {
        let hash_a = B256::repeat_byte(0x11);
        let hash_b = B256::repeat_byte(0x22);
        let wire: RpcPreimages = HashMap::from([
            (0u8, HashMap::from([(hash_a, B64.to_string())])),
            (1u8, HashMap::from([(hash_b, B64.to_string())])),
        ]);

        let result = decode_preimages(Some(wire)).unwrap();
        assert_eq!(result.len(), 2);
        assert_eq!(result[&PreimageType::Keccak256][&hash_a], BYTES);
        assert_eq!(result[&PreimageType::Sha2_256][&hash_b], BYTES);
    }

    #[test]
    fn decode_preimages_keeps_empty_inner_map() {
        let wire: RpcPreimages = HashMap::from([(0u8, HashMap::new())]);

        let result = decode_preimages(Some(wire)).unwrap();
        assert!(result[&PreimageType::Keccak256].is_empty());
    }

    #[test]
    fn decode_preimages_invalid_base64_errors() {
        let wire: RpcPreimages = HashMap::from([(
            0u8,
            HashMap::from([(B256::repeat_byte(0x11), "invalid preimage!".to_string())]),
        )]);

        let err = decode_preimages(Some(wire)).unwrap_err();
        assert!(matches!(err, DaError::Base64(_)));
    }

    #[test]
    fn decode_preimages_unknown_type_errors() {
        let wire: RpcPreimages = HashMap::from([(
            99u8,
            HashMap::from([(B256::repeat_byte(0x11), B64.to_string())]),
        )]);

        let err = decode_preimages(Some(wire)).unwrap_err();
        assert!(matches!(err, DaError::UnknownPreimageType(99)));
    }

    #[tokio::test]
    async fn recover_payload_decodes_response() {
        let asserter = Asserter::new();
        asserter.push_success(&serde_json::json!({ "Payload": B64 }));

        let result = reader(asserter)
            .recover_payload(1, B256::repeat_byte(1), &[])
            .await
            .unwrap();
        assert_eq!(result, BYTES);
    }

    #[tokio::test]
    async fn recover_payload_missing_field_is_empty() {
        let asserter = Asserter::new();
        asserter.push_success(&serde_json::json!({}));

        let result = reader(asserter)
            .recover_payload(1, B256::repeat_byte(1), &[])
            .await
            .unwrap();
        assert!(result.is_empty());
    }

    #[tokio::test]
    async fn collect_preimages_decodes_response() {
        let hash = B256::repeat_byte(0x11);
        let asserter = Asserter::new();
        asserter.push_success(&serde_json::json!({
            "Preimages": { "0": { hash.to_string(): B64 } }
        }));

        let result = reader(asserter)
            .collect_preimages(1, B256::repeat_byte(1), &[])
            .await
            .unwrap();
        assert_eq!(result[&PreimageType::Keccak256][&hash], BYTES);
    }

    #[tokio::test]
    async fn collect_preimages_missing_field_is_empty() {
        let asserter = Asserter::new();
        asserter.push_success(&serde_json::json!({}));

        let result = reader(asserter)
            .collect_preimages(1, B256::repeat_byte(1), &[])
            .await
            .unwrap();
        assert!(result.is_empty());
    }

    #[tokio::test]
    async fn recover_payload_and_preimages_decodes_both() {
        let hash = B256::repeat_byte(0x11);
        let asserter = Asserter::new();
        asserter.push_success(&serde_json::json!({
            "Payload": B64,
            "Preimages": { "0": { hash.to_string(): B64 } },
        }));

        let (payload, preimages) = reader(asserter)
            .recover_payload_and_preimages(1, B256::repeat_byte(1), &[])
            .await
            .unwrap();
        assert_eq!(payload, BYTES);
        assert_eq!(preimages[&PreimageType::Keccak256][&hash], BYTES);
    }

    #[tokio::test]
    async fn recover_payload_and_preimages_missing_fields_are_empty() {
        let asserter = Asserter::new();
        asserter.push_success(&serde_json::json!({}));

        let (payload, preimages) = reader(asserter)
            .recover_payload_and_preimages(1, B256::repeat_byte(1), &[])
            .await
            .unwrap();
        assert!(payload.is_empty());
        assert!(preimages.is_empty());
    }

    #[tokio::test]
    async fn transport_error_surfaces_as_provider_error() {
        let asserter = Asserter::new();
        asserter.push_failure_msg("internal server error");

        let err = reader(asserter)
            .recover_payload(1, B256::repeat_byte(1), &[])
            .await
            .unwrap_err();
        assert!(matches!(err, DaError::Provider(_)));
    }
}
