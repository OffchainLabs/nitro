// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{spawner_endpoints, ServerState};
use axum::routing::{get, post};
use axum::Router;
use std::sync::Arc;
use tower_http::trace::TraceLayer;

pub fn create_router() -> Router<Arc<ServerState>> {
    let router = Router::new()
        // Standard JSON-RPC 2.0 dispatch endpoint (used by go-ethereum's rpc.Client)
        .route("/", post(spawner_endpoints::jsonrpc_dispatch))
        // Path-based endpoints (used by direct HTTP callers)
        .route("/validate", post(spawner_endpoints::validate))
        .layer(TraceLayer::new_for_http());
    #[cfg(test)]
    let router = router.route("/test", get(|| async { "OK" }));
    router
}
