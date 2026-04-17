//! `arbtimeboost_*` and `arbtimeboostauctioneer_*` RPC namespaces.
//!
//! Arbitrum Timeboost is an optional sequencer-side priority-lane
//! feature: an auctioneer resolves bids for express-lane slots, and
//! winning txs are submitted through `arbtimeboost_*` with round +
//! sequence metadata. The feature is off by default.
//!
//! When timeboost isn't configured on the node, both namespaces
//! return "not enabled" errors (matching Nitro's behavior when the
//! `txPublisher` has no timeboost backend set).

use alloy_primitives::{Address, Bytes, B256, U256};
use jsonrpsee::{
    core::RpcResult,
    proc_macros::rpc,
    types::{error::INTERNAL_ERROR_CODE, ErrorObject},
};
use serde::{Deserialize, Serialize};

fn not_enabled(feature: &str) -> ErrorObject<'static> {
    ErrorObject::owned(
        INTERNAL_ERROR_CODE,
        format!("{feature} is not enabled on this node"),
        None::<()>,
    )
}

/// Wire format of an express-lane submission. Matches Nitro's
/// `timeboost.JsonExpressLaneSubmission` (see
/// `timeboost/express_lane_service.go`). Fields are accepted as-is
/// and passed through to the transaction publisher — we don't
/// validate the signature ourselves.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ExpressLaneSubmission {
    pub chain_id: U256,
    pub round: u64,
    pub auction_contract_address: Address,
    pub sequence: u64,
    pub transaction: Bytes,
    pub signature: Bytes,
}

/// `arbtimeboost` RPC namespace — sequencer-facing submission API.
#[rpc(server, namespace = "arbtimeboost")]
pub trait ArbTimeboostApi {
    /// Submit a signed express-lane transaction for inclusion in the
    /// current round.
    #[method(name = "sendExpressLaneTransaction")]
    async fn send_express_lane_transaction(&self, msg: ExpressLaneSubmission) -> RpcResult<()>;
}

/// `arbtimeboostauctioneer` RPC namespace — auctioneer-facing
/// resolution API.
#[rpc(server, namespace = "arbtimeboostauctioneer")]
pub trait ArbTimeboostAuctioneerApi {
    /// Submit the winning auction resolution tx. The transaction
    /// encodes round + winner + bids. Only the configured auctioneer
    /// may call this.
    #[method(name = "submitAuctionResolutionTransaction")]
    async fn submit_auction_resolution_transaction(&self, raw_tx: Bytes) -> RpcResult<B256>;
}

/// Configuration for the timeboost namespaces.
#[derive(Debug, Clone, Default)]
pub struct ArbTimeboostConfig {
    /// When false, all methods return "not enabled".
    pub express_lane_enabled: bool,
    /// When false, the auctioneer method returns "not enabled".
    pub auctioneer_enabled: bool,
}

/// Handler for both `arbtimeboost_*` and `arbtimeboostauctioneer_*`.
#[derive(Debug, Clone)]
pub struct ArbTimeboostHandler {
    config: ArbTimeboostConfig,
}

impl ArbTimeboostHandler {
    pub fn new(config: ArbTimeboostConfig) -> Self {
        Self { config }
    }
}

#[async_trait::async_trait]
impl ArbTimeboostApiServer for ArbTimeboostHandler {
    async fn send_express_lane_transaction(&self, _msg: ExpressLaneSubmission) -> RpcResult<()> {
        if !self.config.express_lane_enabled {
            return Err(not_enabled("timeboost express lane"));
        }
        // TODO: hand off to a timeboost-aware tx publisher.
        Err(not_enabled("timeboost express lane publisher"))
    }
}

#[async_trait::async_trait]
impl ArbTimeboostAuctioneerApiServer for ArbTimeboostHandler {
    async fn submit_auction_resolution_transaction(&self, _raw_tx: Bytes) -> RpcResult<B256> {
        if !self.config.auctioneer_enabled {
            return Err(not_enabled("timeboost auctioneer"));
        }
        // TODO: validate caller == configured auctioneer, decode tx,
        // forward to the auction-resolution publisher.
        Err(not_enabled("timeboost auctioneer publisher"))
    }
}
