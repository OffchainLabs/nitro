// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! JIT Machine Process Manager.
//!
//! This module acts as the driver for the external "JIT" binary running in
//! "continuous" mode. It is the Rust counterpart to the Go implementation of `JitMachine`
//! found in the `validator` package.
//!
//! # Architecture
//! Unlike "Native" mode (which runs logic in-process), this module manages a persistent
//! subprocess (`jit`) to perform validation. The communication protocol handles the
//! exchange of validation inputs (batches, preimages, WASM binaries) and outputs
//! (new state) via a hybrid IPC mechanism:
//!
//! 1. **Handshake (Stdin):** The server opens an ephemeral TCP listener and writes its
//!    address to the subprocess's Standard Input.
//! 2. **Data Transport (TCP):** The subprocess connects back to the provided address.
//!    This TCP stream is then used for data transfer of the `ValidationRequest` and
//!    the resulting `GlobalState`.

use crate::engine::config::JitMachineConfig;
use anyhow::{anyhow, Context, Result};
use arbutil::Bytes32;
use std::net::TcpListener;
use std::{
    env::{self},
    path::{Path, PathBuf},
    process::Stdio,
};
use tokio::io::AsyncWriteExt;
use tokio::{
    process::{Child, ChildStdin, Command},
    sync::Mutex,
};
use tracing::{error, warn};
use validation::transfer::{receive_response, send_validation_input};
use validation::{GoGlobalState, ValidationInput};

#[derive(Debug)]
pub struct JitMachine {
    /// Handler to jit binary stdin. Instead of using Mutex<> for the entire
    /// JitMachine we chose to use a more granular Mutex<> to avoid contention
    pub process_stdin: Mutex<Option<ChildStdin>>,
    /// Handler to jit binary process. Needs a Mutex<> to force quit on server shutdown
    pub process: Mutex<Child>,
    pub wasm_memory_usage_limit: u64,
}

impl JitMachine {
    pub fn new(config: &JitMachineConfig, module_root: Option<Bytes32>) -> Result<Self> {
        let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));
        let root_path: PathBuf = manifest_dir
            .parent()
            .and_then(|p| p.parent())
            .map(|p| p.to_path_buf()) // Convert &Path to PathBuf
            .unwrap_or_else(|| {
                // This runs only if the parents don't exist
                env::current_dir().expect("Failed to get current working directory")
            });

        // TODO: use JitLocator to get jit_path
        let jit_path = root_path.join("target").join("bin").join("jit");
        let mut cmd = Command::new(jit_path);

        // TODO: use JitLocator to get bin_path
        let bin_path = if let Some(module_root) = module_root {
            root_path
                .join("target")
                .join("machines")
                .join(format!("0x{module_root}"))
                .join(&config.prover_bin_path)
        } else {
            root_path
                .join("target")
                .join("machines")
                .join("latest")
                .join("replay.wasm")
        };

        if config.jit_cranelift {
            cmd.arg("--cranelift");
        }

        cmd.arg("--binary")
            .arg(bin_path)
            .arg("continuous")
            .stdin(Stdio::piped()) // We must pipe stdin so we can write to it.
            .stdout(Stdio::inherit()) // Inherit stdout/stderr so logs show up in your main console.
            .stderr(Stdio::inherit());

        let mut child = cmd.spawn().context("failed to spawn jit binary")?;

        let stdin = child
            .stdin
            .take()
            .ok_or_else(|| anyhow!("failed to open stdin to jit process"))?;

        Ok(Self {
            process_stdin: Mutex::new(Some(stdin)),
            process: Mutex::new(child),
            wasm_memory_usage_limit: config.wasm_memory_usage_limit,
        })
    }

    pub async fn is_active(&self) -> bool {
        self.process_stdin.lock().await.is_some()
    }

    pub async fn feed_machine(&self, request: &ValidationInput) -> Result<GoGlobalState> {
        // 1. Create new TCP connection
        // Binding with a port number of 0 will request that the OS assigns a port to this listener.
        let listener = TcpListener::bind("127.0.0.1:0").context("failed to create TCP listener")?;

        let addr = listener.local_addr().context("failed to get local addr")?;

        // 2. Format the address string (Go: "%v\n")
        let address_str = format!("{addr}\n");

        // 3. Send TCP connection via stdin pipe
        {
            let mut locked_process_stdin = self.process_stdin.lock().await;
            if let Some(stdin) = locked_process_stdin.as_mut() {
                stdin
                    .write_all(address_str.as_bytes())
                    .await
                    .context("failed to write address to jit stdin")?;
            } else {
                return Err(anyhow!("JIT machine stdin is not available"));
            }
        }

        // 4. Wait for the child to call us back
        let (mut conn, _) = listener
            .accept()
            .context("failed to open listener connection")?;

        // 5. Send data
        send_validation_input(&mut conn, request)?;

        // 6. Read Response and return new state
        match receive_response(&mut conn)? {
            Ok((new_state, memory_used)) => {
                if memory_used > self.wasm_memory_usage_limit {
                    warn!(
                        "WARN: memory used {} exceeds limit {}",
                        memory_used, self.wasm_memory_usage_limit
                    );
                }
                Ok(new_state)
            }
            Err(err) => {
                error!("Jit Machine Failure message: {err}");
                Err(anyhow!("Jit Machine Failure: {err}"))
            }
        }
    }

    pub async fn complete_machine(&self) -> Result<()> {
        // Close stdin. This sends EOF to the child process, signaling it to stop.
        // We take the Option to ensure it's dropped and cannot be used again.

        let mut locked_process_stdin = self.process_stdin.lock().await;
        if let Some(stdin) = locked_process_stdin.take() {
            drop(stdin);
        }

        let mut locked_process = self.process.lock().await;
        locked_process
            .kill()
            .await
            .context("failed to kill jit process")?;

        Ok(())
    }
}
