// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{spawner_endpoints, ServerState};
use axum::routing::{get, post};
use axum::Router;
use std::sync::Arc;
use tower_http::trace::TraceLayer;

const BASE_NAMESPACE: &str = "/validation";

pub fn create_router() -> Router<Arc<ServerState>> {
    Router::new()
        .route(
            &format!("{BASE_NAMESPACE}_capacity"),
            get(spawner_endpoints::capacity),
        )
        .route(
            &format!("{BASE_NAMESPACE}_name"),
            get(spawner_endpoints::name),
        )
        .route(
            &format!("{BASE_NAMESPACE}_stylusArchs"),
            get(spawner_endpoints::stylus_archs),
        )
        .route(
            &format!("{BASE_NAMESPACE}_validate"),
            post(spawner_endpoints::validate),
        )
        .route(
            &format!("{BASE_NAMESPACE}_wasmModuleRoots"),
            get(spawner_endpoints::wasm_module_roots),
        )
        .layer(TraceLayer::new_for_http())
}
