// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use std::{future::Future, sync::Arc};

use anyhow::Result;
use axum::{Router, routing::post};
use tokio::{net::TcpListener, signal};
use tower_http::trace::TraceLayer;
use tracing::info;

use crate::{config::ServerState, jwt, spawner_endpoints};

pub async fn run_server(listener: TcpListener, state: Arc<ServerState>) -> Result<()> {
    run_server_internal(listener, state, shutdown_signal()).await
}

async fn run_server_internal(
    listener: TcpListener,
    state: Arc<ServerState>,
    shutdown: impl Future<Output = ()> + Send + 'static,
) -> Result<()> {
    let shutdown_state = state.clone();
    axum::serve(listener, create_router(state))
        .with_graceful_shutdown(shutdown)
        .await?;

    info!("Shutdown signal received. Running cleanup...");

    shutdown_state.shutdown().await
}

fn create_router(state: Arc<ServerState>) -> Router {
    let router = Router::new()
        .route("/", post(spawner_endpoints::jsonrpc_dispatch))
        .route_layer(axum::middleware::from_fn_with_state(
            state.clone(),
            jwt::auth_middleware,
        ))
        .layer(TraceLayer::new_for_http())
        .with_state(state);

    // Test-only health-check endpoint. Added after route_layer so it is NOT
    // behind JWT auth — this is intentional: tests use it to verify the server
    // is up without needing a valid token.
    #[cfg(test)]
    let router = router.route("/test", axum::routing::get(|| async { "OK" }));

    router
}

// Listens for Ctrl+C or SIGTERM
pub(crate) async fn shutdown_signal() {
    // Handles Ctrl+C signal
    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    // Handles SIGTERM signal used by Kubernetes/Docker shutdown
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    // Handles SIGQUIT signal
    let quit = async {
        signal::unix::signal(signal::unix::SignalKind::quit())
            .expect("failed to install SIGQUIT handler")
            .recv()
            .await;
    };

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
        _ = quit => {},
    }
}

#[cfg(test)]
mod tests {
    use std::{net::SocketAddr, sync::Arc};

    use anyhow::Result;
    use clap::Parser;
    use tokio::{
        net::TcpListener,
        sync::oneshot::{self, Sender},
        task::JoinHandle,
    };

    use crate::{
        config::{ExecutionMode, ServerConfig, ServerState},
        engine::ModuleRoot,
        server::run_server_internal,
    };

    struct TestServerConfig {
        sender: Sender<()>,
        server_handle: JoinHandle<Result<()>>,
        addr: SocketAddr,
        state: Arc<ServerState>,
    }

    async fn spinup_server(config: &ServerConfig) -> Result<TestServerConfig> {
        let state = Arc::new(ServerState::new(config, 4)?);
        // 2. Bind to random free port
        let listener = TcpListener::bind("127.0.0.1:0").await?;
        let addr = listener.local_addr()?;
        println!("Test server listening on {addr}");

        // 3. Create a channel to simulate Ctrl+C
        let (tx, rx) = oneshot::channel();

        // 4. Spawn the server in the background
        let state_for_server = state.clone();
        let server_handle = tokio::spawn(async move {
            // We map the oneshot channel error to () because the server expects Future<Output=()>
            let shutdown_signal = async {
                let _ = rx.await;
            };
            run_server_internal(listener, state_for_server, shutdown_signal).await
        });

        Ok(TestServerConfig {
            sender: tx,
            server_handle,
            addr,
            state,
        })
    }

    async fn verify_and_shutdown_server(
        test_config: TestServerConfig,
        module_root: ModuleRoot,
    ) -> Result<()> {
        // 5. Make a real request here to prove the server is up
        let client = reqwest::Client::new();
        let resp = client
            .get(format!("http://{}/test", test_config.addr))
            .send()
            .await;

        assert!(
            resp.is_ok(),
            "Failed to connect to validation_capacity endpoint"
        );
        assert_eq!(resp?.status(), 200);

        // 6. Trigger Shutdown
        println!("Sending shutdown signal...");
        let _ = test_config.sender.send(());

        // 7. Wait for the server to finish (this ensures cleanup ran)
        let result = test_config.server_handle.await?;
        assert!(result.is_ok(), "Server should exit successfully");

        // 8. Verify jit_manager Cleanup
        if let ExecutionMode::Continuous { jit_manager } = &test_config.state.execution {
            let machines = jit_manager.machines.read().await;
            assert!(machines.get(&module_root).is_none());
        }

        // 9. Verify same request from above fails expectedly
        let resp = client
            .get(format!("http://{}/test", test_config.addr))
            .send()
            .await;

        assert!(resp.is_err(), "server should not be up");

        Ok(())
    }

    async fn test_server_lifecycle(additional_args: Option<Vec<&'static str>>) -> Result<()> {
        let mut args = vec!["server"];
        if let Some(extra) = additional_args {
            args = [&args[..], &extra[..]].concat();
        }

        let config = ServerConfig::try_parse_from(args)?;
        let test_config = spinup_server(&config).await?;

        let module_root = test_config
            .state
            .locator
            .latest_wasm_module_root()
            .module_root;
        verify_and_shutdown_server(test_config, module_root).await
    }

    #[tokio::test]
    async fn test_server_lifecycle_native_mode() -> Result<()> {
        test_server_lifecycle(Some(vec!["--mode", "native"])).await
    }

    #[tokio::test]
    async fn test_server_lifecycle_continuous_mode() -> Result<()> {
        test_server_lifecycle(Some(vec!["--mode", "continuous"])).await
    }
}
