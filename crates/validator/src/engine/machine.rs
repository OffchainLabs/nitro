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

use crate::engine::config::{JitManagerConfig, ModuleRoot, REPLAY_WASM};
use crate::engine::machine_locator::MachineLocator;
use anyhow::{anyhow, Context, Result};
use std::collections::HashMap;
use std::net::TcpListener;
use std::thread::sleep;
use std::time::Duration;
use std::{
    env::{self},
    path::{Path, PathBuf},
    process::Stdio,
    sync::{
        atomic::{AtomicBool, Ordering},
        Arc,
    },
};
use tokio::io::AsyncWriteExt;
use tokio::sync::RwLock;
use tokio::{
    process::{Child, ChildStdin, Command},
    sync::Mutex,
};
use tracing::{debug, error, info, warn};
use validation::transfer::{receive_response, send_validation_input};
use validation::{GoGlobalState, ValidationInput};

#[derive(Debug)]
pub struct JitMachine {
    /// Handler to jit binary stdin. Instead of using Mutex<> for the entire
    /// JitMachine we chose to use a more granular Mutex<> to avoid contention
    pub process_stdin: Mutex<ChildStdin>,
    /// Handler to jit binary process. Needs a Mutex<> to force quit on server shutdown.
    pub process: Mutex<Child>,
}

impl JitMachine {
    pub async fn ensure_alive(&self) -> Result<()> {
        let mut child = self.process.lock().await;
        ensure_process_is_alive(&mut child)
    }

    pub async fn feed_machine(
        &self,
        wasm_memory_usage_limit: u64,
        request: &ValidationInput,
    ) -> Result<GoGlobalState> {
        // 0. Ensure process is alive
        self.ensure_alive().await?;

        // 1. Create new TCP connection
        // Binding with a port number of 0 will request that the OS assigns a port to this listener.
        let listener = TcpListener::bind("127.0.0.1:0").context("failed to create TCP listener")?;

        let addr = listener.local_addr().context("failed to get local addr")?;

        // 2. Format the address string (Go: "%v\n")
        let address_str = format!("{addr}\n");

        // 3. Send TCP connection via stdin pipe
        self.process_stdin
            .lock()
            .await
            .write_all(address_str.as_bytes())
            .await
            .context("failed to write address to jit stdin")?;

        // 4. Wait for the child to call us back
        let (mut conn, _) = listener
            .accept()
            .context("failed to open listener connection")?;

        // 5. Send data
        send_validation_input(&mut conn, request)?;

        // 6. Read Response and return new state
        match receive_response(&mut conn)? {
            Ok((new_state, memory_used)) => {
                if memory_used > wasm_memory_usage_limit {
                    warn!(
                        "WARN: memory used {memory_used} exceeds limit {wasm_memory_usage_limit}",
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
        drop(self.process_stdin.lock().await);

        let mut locked_process = self.process.lock().await;
        locked_process
            .kill()
            .await
            .context("failed to kill jit process")
    }
}

#[derive(Debug)]
pub struct JitProcessManager {
    pub wasm_memory_usage_limit: u64,
    // Using Arc<JitMachine> allows us to clone the Arc and drop the HashMap lock
    // immediately, avoiding contention during long-running I/O operations.
    pub machines: RwLock<HashMap<ModuleRoot, Arc<JitMachine>>>,
    // Signals that the server is shutting down. When true, new requests are rejected.
    shutting_down: AtomicBool,
}

impl JitProcessManager {
    pub fn new_empty(config: &JitManagerConfig) -> Self {
        Self {
            wasm_memory_usage_limit: config.wasm_memory_usage_limit,
            machines: RwLock::new(HashMap::new()),
            shutting_down: AtomicBool::new(false),
        }
    }

    pub fn new(config: &JitManagerConfig, locator: &MachineLocator) -> Result<Self> {
        let machines: HashMap<ModuleRoot, Arc<JitMachine>> = locator
            .module_roots()
            .iter()
            .cloned()
            .map(|root_meta| {
                let root_path = root_meta.path.join(REPLAY_WASM);
                let sub_machine = create_jit_machine(config.jit_cranelift, &root_path)?;
                Ok::<(ModuleRoot, Arc<JitMachine>), anyhow::Error>((
                    root_meta.module_root,
                    Arc::new(sub_machine),
                ))
            })
            .collect::<Result<_, _>>()?;

        Ok(Self {
            wasm_memory_usage_limit: config.wasm_memory_usage_limit,
            machines: RwLock::new(machines),
            shutting_down: AtomicBool::new(false),
        })
    }

    pub async fn feed_machine_with_root(
        &self,
        request: &ValidationInput,
        module_root: ModuleRoot,
    ) -> Result<GoGlobalState> {
        // Reject new operations if we're shutting down
        if self.shutting_down.load(Ordering::Acquire) {
            return Err(anyhow!("Server is shutting down"));
        }

        let machine = {
            let machines = self.machines.read().await;
            match machines.get(&module_root) {
                // Clone the Arc while holding the read lock, then drop the lock immediately.
                // This allows other threads to access the HashMap while we perform I/O operations.
                Some(machine) => machine.clone(),
                None => return Err(anyhow!("Trying to feed machine when no machine for module root {module_root} is available/running"))
            }
        };

        machine
            .feed_machine(self.wasm_memory_usage_limit, request)
            .await
    }

    pub async fn complete_machines(&self) -> Result<()> {
        // Signal that we're shutting down to reject new requests
        self.shutting_down.store(true, Ordering::Release);

        // It's okay and expected to hold the write lock while shutting down since we don't
        // allow any other requests to be performed on the server
        let mut locked_machines = self.machines.write().await;

        // Iterate over all machines: for each one, complete it and remove it from the map
        // while holding the write lock. This ensures no other thread can access machines
        // during shutdown.
        for (module_root, machine) in locked_machines.drain() {
            info!("Completing machine with module root {module_root}");
            machine.complete_machine().await?;
        }

        Ok(())
    }
}

fn create_jit_machine(jit_cranelift: bool, prover_bin_path: &PathBuf) -> Result<JitMachine> {
    let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));
    let root_path: PathBuf = manifest_dir
        .parent()
        .and_then(|p| p.parent())
        .map(|p| p.to_path_buf()) // Convert &Path to PathBuf
        .unwrap_or_else(|| {
            // This runs only if the parents don't exist
            env::current_dir().expect("Failed to get current working directory")
        });

    // TODO: use helper to get jit_path (NIT-4347)
    let jit_path = root_path.join("target").join("bin").join("jit");
    let mut cmd = Command::new(jit_path);

    if jit_cranelift {
        cmd.arg("--cranelift");
    }

    cmd.arg("--binary")
        .arg(prover_bin_path)
        .arg("continuous")
        .stdin(Stdio::piped()) // We must pipe stdin so we can write to it.
        .stdout(Stdio::inherit()) // Inherit stdout/stderr so logs show up in your main console.
        .stderr(Stdio::inherit());

    debug!("Executing JIT command: {:?}", cmd);

    let mut child = cmd.spawn().context("failed to spawn jit binary")?;

    // Wait briefly for the OS to allocate resources and for the child process to start up. Then,
    // check if the child process has already exited, which would indicate a startup failure.
    debug!("Waiting for JIT process to come up");
    sleep(Duration::from_secs(2));
    ensure_process_is_alive(&mut child)?;

    let stdin = child
        .stdin
        .take()
        .ok_or_else(|| anyhow!("failed to open stdin to jit process"))?;

    Ok(JitMachine {
        process_stdin: Mutex::new(stdin),
        process: Mutex::new(child),
    })
}

fn ensure_process_is_alive(p: &mut Child) -> Result<()> {
    match p.try_wait() {
        Ok(Some(status)) => Err(anyhow!("JIT process has exited with status: {status}")),
        Ok(None) => Ok(()),
        Err(err) => Err(anyhow!("failed to check jit process status: {err}")),
    }
}
