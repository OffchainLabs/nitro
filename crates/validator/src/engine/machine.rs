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

use anyhow::{anyhow, Context, Result};
use arbutil::Bytes32;
use std::{
    collections::HashMap,
    env::{self},
    path::{Path, PathBuf},
    process::Stdio,
};
use tokio::{
    io::{AsyncRead, AsyncReadExt, AsyncWriteExt},
    net::{TcpListener, TcpStream},
    process::{Child, ChildStdin, Command},
};
use tracing::{error, warn};

use crate::{
    engine::{config::JitMachineConfig, execution::ValidationRequest},
    spawner_endpoints::{local_target, GlobalState},
};

const SUCCESS_BYTE: u8 = 0x0;
const FAILURE_BYTE: u8 = 0x1;
const ANOTHER_BYTE: u8 = 0x3;
const READY_BYTE: u8 = 0x4;

async fn write_exact(conn: &mut TcpStream, data: &[u8]) -> Result<()> {
    conn.write_all(data).await.map_err(|e| anyhow!(e))
}

async fn write_u8(conn: &mut TcpStream, data: u8) -> Result<()> {
    write_exact(conn, &[data]).await
}

async fn write_u32(conn: &mut TcpStream, data: u32) -> Result<()> {
    write_exact(conn, &data.to_be_bytes()).await
}

async fn write_u64(conn: &mut TcpStream, data: u64) -> Result<()> {
    write_exact(conn, &data.to_be_bytes()).await
}

async fn write_bytes(conn: &mut TcpStream, data: &[u8]) -> Result<()> {
    write_u64(conn, data.len() as u64).await?;
    write_exact(conn, data).await
}

async fn read_bytes32<R: AsyncRead + Unpin>(reader: &mut R) -> Result<[u8; 32]> {
    let mut buf = [0u8; 32];
    reader.read_exact(&mut buf).await?;
    Ok(buf)
}

async fn read_bytes_with_len<R: AsyncRead + Unpin>(reader: &mut R) -> Result<Vec<u8>> {
    let len = reader.read_u64().await?;
    let mut buf = vec![0u8; len as usize];
    reader.read_exact(&mut buf).await?;
    Ok(buf)
}

#[derive(Debug)]
pub struct JitMachine {
    pub process_stdin: Option<ChildStdin>,
    pub process: Child,
    pub wasm_memory_usage_limit: u64,
    is_active: bool,
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
            process_stdin: Some(stdin),
            process: child,
            wasm_memory_usage_limit: config.wasm_memory_usage_limit,
            is_active: true,
        })
    }

    pub fn is_active(&self) -> bool {
        self.is_active && self.process_stdin.is_some()
    }

    pub async fn feed_machine(&mut self, request: &ValidationRequest) -> Result<GlobalState> {
        // 1. Create new TCP connection
        // Binding with a port number of 0 will request that the OS assigns a port to this listener.
        let listener = TcpListener::bind("127.0.0.1:0")
            .await
            .context("failed to create TCP listener")?;

        let mut state = GlobalState::default();

        let addr = listener.local_addr().context("failed to get local addr")?;

        // 2. Format the address string (Go: "%v\n")
        let address_str = format!("{addr}\n");

        // 3. Send TCP connection via stdin pipe
        if let Some(stdin) = &mut self.process_stdin {
            stdin
                .write_all(address_str.as_bytes())
                .await
                .context("failed to write address to jit stdin")?;
        } else {
            return Err(anyhow!("JIT machine stdin is not available"));
        }

        // 4. Wait for the child to call us back
        let (mut conn, _) = listener
            .accept()
            .await
            .context("failed to open listener connection")?;

        // 5. Send Global State
        // TODO: add timeout for reads and writes
        write_u64(&mut conn, request.start_state.batch).await?;
        write_u64(&mut conn, request.start_state.pos_in_batch).await?;
        write_exact(&mut conn, &request.start_state.block_hash.0).await?;
        write_exact(&mut conn, &request.start_state.send_root.0).await?;

        // 6. Send batch info
        for batch in request.batch_info.iter() {
            write_u8(&mut conn, ANOTHER_BYTE).await?;
            write_u64(&mut conn, batch.number).await?;
            write_bytes(&mut conn, &batch.data).await?;
        }
        write_u8(&mut conn, SUCCESS_BYTE).await?;

        // 7. Send Delayed Inbox
        if request.has_delayed_msg {
            write_u8(&mut conn, ANOTHER_BYTE).await?;
            write_u64(&mut conn, request.delayed_msg_number).await?;
            write_bytes(&mut conn, &request.delayed_msg).await?;
        }
        write_u8(&mut conn, SUCCESS_BYTE).await?;

        // 8. Send Known Preimages
        write_u32(&mut conn, request.preimages.len() as u32).await?;

        for (ty, preimages) in request.preimages.iter() {
            write_u8(&mut conn, *ty as u8).await?;
            write_u32(&mut conn, preimages.len() as u32).await?;
            for (hash, preimage) in preimages {
                write_exact(&mut conn, &hash.0).await?;
                write_bytes(&mut conn, preimage).await?;
            }
        }

        // 9. Send User Wasms
        let local_target = local_target();
        let local_user_wasm = request.user_wasms.get(local_target);

        // if there are user wasms, but only for wrong architecture - error
        if local_user_wasm.is_none_or(|m| m.is_empty()) {
            for (arch, wasms) in &request.user_wasms {
                if !wasms.is_empty() {
                    return Err(anyhow!(
                        "bad stylus arch. got {arch}, expected {local_target}",
                    ));
                }
            }
        }

        let empty_map = HashMap::new();
        let local_user_wasm = local_user_wasm.unwrap_or(&empty_map);
        write_u32(&mut conn, local_user_wasm.len() as u32).await?;
        for (module_hash, program) in local_user_wasm {
            write_exact(&mut conn, &module_hash.0).await?;
            write_bytes(&mut conn, program).await?;
        }

        // 10. Signal that we are done sending global state
        write_u8(&mut conn, READY_BYTE).await?;

        // 11. Read Response and return new state
        let mut kind_buf = [0u8; 1];
        conn.read_exact(&mut kind_buf).await?;

        match kind_buf[0] {
            FAILURE_BYTE => {
                let msg_bytes = read_bytes_with_len(&mut conn).await?;
                let msg = String::from_utf8_lossy(&msg_bytes);
                error!("Jit Machine Failure message: {msg}");
                Err(anyhow!("Jit Machine Failure: {msg}"))
            }
            SUCCESS_BYTE => {
                // We write the values to socket in BigEndian so we can use
                // read_u64() directly from AsyncReadExt which handles
                // BigEndian by default
                state.batch = conn.read_u64().await?;
                state.pos_in_batch = conn.read_u64().await?;
                state.block_hash.0 = read_bytes32(&mut conn).await?;
                state.send_root.0 = read_bytes32(&mut conn).await?;

                let memory_used = conn.read_u64().await?;
                if memory_used > self.wasm_memory_usage_limit {
                    warn!(
                        "WARN: memory used {} exceeds limit {}",
                        memory_used, self.wasm_memory_usage_limit
                    );
                }

                Ok(state)
            }
            _ => Err(anyhow!("inter-process communication failure: unknown byte")),
        }
    }

    pub async fn complete_machine(&mut self) -> Result<()> {
        // Close stdin. This sends EOF to the child process, signaling it to stop.
        // We take the Option to ensure it's dropped and cannot be used again.
        if let Some(stdin) = self.process_stdin.take() {
            drop(stdin);
        }

        self.process
            .kill()
            .await
            .context("failed to kill jit process")?;

        self.is_active = false;

        Ok(())
    }
}
