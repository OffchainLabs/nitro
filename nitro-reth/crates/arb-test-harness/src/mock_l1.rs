//! Minimal mock L1 RPC server.
//!
//! Serves the handful of methods Nitro and arbreth probe at startup
//! (`eth_chainId`, `eth_blockNumber`, `eth_getBlockByNumber`,
//! `eth_call`). Replaces the prior Python `mock_l1.py` so the entire
//! orchestration is in Rust.
//!
//! Implementation intentionally lives in this skeleton with a
//! [`MockL1::start`] that reserves the handle shape; the axum routes
//! are filled in by Agent A.

use std::net::SocketAddr;

use crate::{error::HarnessError, Result};

pub struct MockL1 {
    pub addr: SocketAddr,
    pub chain_id: u64,
    // shutdown_tx: tokio::sync::oneshot::Sender<()>,
    // join_handle: tokio::task::JoinHandle<()>,
}

impl MockL1 {
    /// Start the server bound to a free port, return the bound address.
    pub fn start(_chain_id: u64) -> Result<Self> {
        Err(HarnessError::NotImplemented {
            what: "MockL1::start (Stage 2 / Agent A)",
        })
    }

    pub fn rpc_url(&self) -> String {
        format!("http://{}", self.addr)
    }

    pub fn shutdown(self) -> Result<()> {
        Ok(())
    }
}
