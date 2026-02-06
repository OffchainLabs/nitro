// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use anyhow::Result;
use std::future::Future;
use std::sync::Arc;
use tokio::net::TcpListener;
use tokio::signal;
use tracing::info;

use crate::config::ServerState;
use crate::router::create_router;

pub async fn run_server(listener: TcpListener, state: Arc<ServerState>) -> Result<()> {
    run_server_internal(listener, state, shutdown_signal()).await
}

async fn run_server_internal<F>(
    listener: TcpListener,
    state: Arc<ServerState>,
    shutdown: F,
) -> Result<()>
where
    F: Future<Output = ()> + Send + 'static,
{
    axum::serve(listener, create_router().with_state(state.clone()))
        .with_graceful_shutdown(shutdown)
        .await?;

    info!("Shutdown signal received. Running cleanup...");

    state.jit_manager.complete_machines().await?;

    Ok(())
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
    use anyhow::Result;
    use clap::Parser;
    use std::{net::SocketAddr, sync::Arc};
    use tokio::{
        net::TcpListener,
        sync::oneshot::{self, Sender},
        task::JoinHandle,
    };

    use crate::{
        config::{ServerConfig, ServerState},
        engine::config::ModuleRoot,
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
        println!("Test server listening on {}", addr);

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
            .get(format!("http://{}/validation_capacity", test_config.addr))
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
        let machine_arc = {
            let machines = test_config.state.jit_manager.machines.read().await;
            machines.get(&module_root).cloned()
        };
        assert!(machine_arc.is_none());

        // 9. Verify same request from above fails expectadly
        let resp = client
            .get(format!("http://{}/validation_capacity", test_config.addr))
            .send()
            .await;

        assert!(resp.is_err(), "server should not be up");

        Ok(())
    }

    #[tokio::test]
    async fn test_server_lifecycle_native_mode() -> Result<()> {
        // 1. Setup Config and State. Use dummy module root is okay.
        let config = ServerConfig::try_parse_from([
            "server",
            "--module-root",
            "0x0000000000000000000000000000000000000000000000000000000000000000",
        ])?;
        let test_config = spinup_server(&config).await?;

        let module_root = config.get_module_root()?;

        verify_and_shutdown_server(test_config, module_root)
            .await?;

        Ok(())
    }

    #[tokio::test]
    async fn test_server_lifecycle_continuous_mode() -> Result<()> {
        // 1. Setup Config and State. Use dummy module root is okay.
        let config = ServerConfig::try_parse_from([
            "server",
            "--module-root",
            "0x0000000000000000000000000000000000000000000000000000000000000000",
            "--mode",
            "continuous",
        ])?;
        let module_root = config.get_module_root()?;

        let test_config = spinup_server(&config).await?;

        // Check that jit machine is active
        let machine_arc = {
            let machines = test_config.state.jit_manager.machines.read().await;
            machines.get(&module_root).cloned()
        };
        assert!(machine_arc.is_some());
        drop(machine_arc);

        verify_and_shutdown_server(test_config, module_root)
            .await?;

        Ok(())
    }
}
