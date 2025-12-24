//! Endpoints related to the `ValidationSpawner` Go interface and used by the nitro's validation
//! client.

use axum::response::IntoResponse;

pub async fn capacity() -> impl IntoResponse {
    "1" // TODO: Figure out max number of workers (optionally, make it configurable)
}

pub async fn name() -> impl IntoResponse {
    "Rust JIT validator"
}

pub async fn stylus_archs() -> impl IntoResponse {
    if cfg!(target_os = "linux") {
        if cfg!(target_arch = "aarch64") {
            return "arm64";
        } else if cfg!(target_arch = "x86_64") {
            return "amd64";
        }
    }
    "host"
}

pub async fn validate()  -> impl IntoResponse {}

pub async fn wasm_module_roots() -> impl IntoResponse {
    "[]" // TODO: Figure this out from local replay.wasm
}
