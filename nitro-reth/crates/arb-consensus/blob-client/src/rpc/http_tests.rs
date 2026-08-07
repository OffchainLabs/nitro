//! End-to-end tests for [`BeaconBlobReader`] against a mock beacon HTTP server
//! ([`wiremock`]) and a mock parent chain, covering the request/verification
//! paths that the pure unit tests in `mod.rs` can't reach.

use std::sync::Arc;

use alloy_primitives::B256;
use alloy_rpc_types_eth::Header;
use arb_parent_chain_client::MockParentChainReader;
use reqwest::Url;
use serde_json::json;
use wiremock::{
    Mock, MockServer, ResponseTemplate,
    matchers::{method, path},
};

use super::{BeaconBlobReader, BeaconBlobReaderConfig, kzg};
use crate::{Blob, BlobError, BlobReader};

fn config(beacon_url: &str) -> BeaconBlobReaderConfig {
    BeaconBlobReaderConfig {
        beacon_url: Url::parse(beacon_url).unwrap(),
        secondary_beacon_url: None,
        authorization: None,
    }
}

fn reader(config: BeaconBlobReaderConfig, parent: MockParentChainReader) -> BeaconBlobReader {
    BeaconBlobReader::new(config, Arc::new(parent))
}

fn header(hash: B256, timestamp: u64) -> Header {
    let mut h: Header = Header::default();
    h.inner.timestamp = timestamp;
    h.hash = hash;
    h
}

async fn mount_genesis(server: &MockServer, genesis_time: u64) {
    Mock::given(method("GET"))
        .and(path("/eth/v1/beacon/genesis"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({
            "data": {
                "genesis_time": genesis_time.to_string(),
                "genesis_validators_root": B256::ZERO,
                "genesis_fork_version": "0x00000000",
            }
        })))
        .mount(server)
        .await;
}

async fn mount_spec_raw(server: &MockServer, data: serde_json::Value) {
    Mock::given(method("GET"))
        .and(path("/eth/v1/config/spec"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({ "data": data })))
        .mount(server)
        .await;
}

async fn mount_spec(server: &MockServer, seconds_per_slot: u64) {
    mount_spec_raw(
        server,
        json!({ "SECONDS_PER_SLOT": seconds_per_slot.to_string() }),
    )
    .await;
}

/// Registers genesis + spec so [`BeaconBlobReader::initialize`] succeeds with
/// the given timing, and a parent-chain header so `get_blobs` resolves a slot.
async fn initialized(
    server: &MockServer,
    genesis_time: u64,
    seconds_per_slot: u64,
    block_hash: B256,
    timestamp: u64,
) -> BeaconBlobReader {
    mount_genesis(server, genesis_time).await;
    mount_spec(server, seconds_per_slot).await;
    let mut parent = MockParentChainReader::new();
    parent.with_header(header(block_hash, timestamp));
    let reader = reader(config(&server.uri()), parent);
    reader.initialize().await.unwrap();
    reader
}

#[tokio::test]
async fn initialize_populates_clock_and_rejects_second_call() {
    let server = MockServer::start().await;
    mount_genesis(&server, 0).await;
    mount_spec(&server, 12).await;
    let reader = reader(config(&server.uri()), MockParentChainReader::new());

    reader.initialize().await.unwrap();
    // Re-initializing is a programmer error, not a silent no-op.
    assert!(matches!(
        reader.initialize().await.unwrap_err(),
        BlobError::AlreadyInitialized
    ));
}

#[tokio::test]
async fn initialize_missing_seconds_per_slot() {
    let server = MockServer::start().await;
    mount_genesis(&server, 0).await;
    mount_spec_raw(&server, json!({})).await;
    let reader = reader(config(&server.uri()), MockParentChainReader::new());

    assert!(matches!(
        reader.initialize().await.unwrap_err(),
        BlobError::MissingSpecValue("SECONDS_PER_SLOT")
    ));
}

#[tokio::test]
async fn initialize_rejects_zero_seconds_per_slot() {
    let server = MockServer::start().await;
    mount_genesis(&server, 0).await;
    mount_spec(&server, 0).await;
    let reader = reader(config(&server.uri()), MockParentChainReader::new());

    assert!(matches!(
        reader.initialize().await.unwrap_err(),
        BlobError::ZeroSecondsPerSlot
    ));
}

#[tokio::test]
async fn falls_back_to_secondary_when_primary_fails() {
    let primary = MockServer::start().await;
    Mock::given(method("GET"))
        .respond_with(ResponseTemplate::new(500))
        .mount(&primary)
        .await;
    let secondary = MockServer::start().await;
    mount_genesis(&secondary, 0).await;
    mount_spec(&secondary, 12).await;

    let mut cfg = config(&primary.uri());
    cfg.secondary_beacon_url = Some(Url::parse(&secondary.uri()).unwrap());
    let reader = reader(cfg, MockParentChainReader::new());

    reader.initialize().await.unwrap();
}

#[tokio::test]
async fn primary_failure_without_secondary_surfaces_status() {
    let server = MockServer::start().await;
    Mock::given(method("GET"))
        .respond_with(ResponseTemplate::new(503))
        .mount(&server)
        .await;
    let reader = reader(config(&server.uri()), MockParentChainReader::new());

    let err = reader.initialize().await.unwrap_err();
    let BlobError::Request { primary, secondary } = err else {
        panic!("expected Request error, got {err:?}");
    };
    assert!(secondary.is_none());
    assert!(matches!(*primary, BlobError::StatusNotOk { .. }));
}

#[tokio::test]
async fn get_blobs_returns_verified_blobs() {
    let server = MockServer::start().await;
    let block_hash = B256::repeat_byte(0x22);
    // genesis 0, 12s slots, header at t=120 -> slot 10.
    let reader = initialized(&server, 0, 12, block_hash, 120).await;

    let blob = Blob::default();
    let versioned_hash = kzg::blob_to_versioned_hash(&blob).unwrap();
    Mock::given(method("GET"))
        .and(path("/eth/v1/beacon/blobs/10"))
        .respond_with(ResponseTemplate::new(200).set_body_json(json!({ "data": [blob] })))
        .mount(&server)
        .await;

    let got = reader
        .get_blobs(block_hash, &[versioned_hash])
        .await
        .unwrap();
    assert_eq!(got, vec![blob]);
}

#[tokio::test]
async fn get_blobs_detects_count_mismatch() {
    let server = MockServer::start().await;
    let block_hash = B256::repeat_byte(0x33);
    let reader = initialized(&server, 0, 12, block_hash, 0).await;

    // Server returns one blob, but two versioned hashes were requested.
    Mock::given(method("GET"))
        .and(path("/eth/v1/beacon/blobs/0"))
        .respond_with(
            ResponseTemplate::new(200).set_body_json(json!({ "data": [Blob::default()] })),
        )
        .mount(&server)
        .await;

    let err = reader
        .get_blobs(
            block_hash,
            &[B256::repeat_byte(0xa), B256::repeat_byte(0xb)],
        )
        .await
        .unwrap_err();
    assert!(matches!(
        err,
        BlobError::BlobCountMismatch {
            slot: 0,
            expected: 2,
            got: 1
        }
    ));
}

#[tokio::test]
async fn get_blobs_detects_versioned_hash_mismatch() {
    let server = MockServer::start().await;
    let block_hash = B256::repeat_byte(0x44);
    let reader = initialized(&server, 0, 12, block_hash, 0).await;

    // Count matches (1 == 1), so per-blob verification runs and catches the
    // hash that doesn't correspond to the returned blob.
    Mock::given(method("GET"))
        .and(path("/eth/v1/beacon/blobs/0"))
        .respond_with(
            ResponseTemplate::new(200).set_body_json(json!({ "data": [Blob::default()] })),
        )
        .mount(&server)
        .await;

    let wrong = B256::repeat_byte(0xee);
    let err = reader.get_blobs(block_hash, &[wrong]).await.unwrap_err();
    let BlobError::VersionedHashMismatch {
        index, expected, ..
    } = err
    else {
        panic!("expected VersionedHashMismatch, got {err:?}");
    };
    assert_eq!(index, 0);
    assert_eq!(expected, wrong);
}

#[tokio::test]
async fn get_blobs_reports_expired_for_ancient_slots() {
    let server = MockServer::start().await;
    let block_hash = B256::repeat_byte(0x55);
    // 1s slots -> retention window is 32*4096s; slot 0 anchored at unix 0 is far
    // older than that relative to the real wall clock, so a fetch failure is
    // reported as expired rather than a generic request error.
    let reader = initialized(&server, 0, 1, block_hash, 0).await;

    Mock::given(method("GET"))
        .and(path("/eth/v1/beacon/blobs/0"))
        .respond_with(ResponseTemplate::new(500))
        .mount(&server)
        .await;

    let err = reader
        .get_blobs(block_hash, &[B256::repeat_byte(0x1)])
        .await
        .unwrap_err();
    assert!(matches!(err, BlobError::BlobsExpired { slot: 0, .. }));
}

#[tokio::test]
async fn get_blobs_fails_when_not_initialized() {
    let server = MockServer::start().await;
    let block_hash = B256::repeat_byte(0x66);
    let mut parent = MockParentChainReader::new();
    parent.with_header(header(block_hash, 0));
    let reader = reader(config(&server.uri()), parent);

    // No initialize() call -> the slot clock is unset.
    assert!(matches!(
        reader
            .get_blobs(block_hash, &[B256::repeat_byte(0x1)])
            .await
            .unwrap_err(),
        BlobError::NotInitialized
    ));
}

#[tokio::test]
async fn get_blobs_fails_for_unknown_block() {
    let server = MockServer::start().await;
    let known = B256::repeat_byte(0x77);
    let reader = initialized(&server, 0, 12, known, 0).await;

    let unknown = B256::repeat_byte(0x88);
    let err = reader
        .get_blobs(unknown, &[B256::repeat_byte(0x1)])
        .await
        .unwrap_err();
    assert!(matches!(err, BlobError::BlockNotFound(h) if h == unknown));
}
