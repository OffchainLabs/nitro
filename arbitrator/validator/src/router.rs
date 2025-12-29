use crate::spawner_endpoints;
use axum::routing::{get, post};
use axum::Router;

const BASE_NAMESPACE: &str = "/validation";

pub fn create_router() -> Router {
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
}
