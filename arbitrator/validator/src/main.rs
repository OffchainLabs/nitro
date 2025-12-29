use anyhow::Result;
use axum::{
    routing::{get, post},
    Router,
};
use clap::Parser;
use tokio::net::TcpListener;
use tracing::info;

mod config;
mod logging;
mod spawner_endpoints;

const BASE_NAMESPACE: &str = "/validation";

#[tokio::main]
async fn main() -> Result<()> {
    let config = config::ServerConfig::parse();
    logging::init(config.logging_format)?;
    info!("Starting validator server with config: {:#?}", config);

    let app = Router::new()
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
        );

    let listener = TcpListener::bind(format!("{}:{}", config.host, config.port)).await?;
    axum::serve(listener, app).await.map_err(Into::into)
}
