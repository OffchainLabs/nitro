use std::{
    net::SocketAddr,
    sync::{
        atomic::{AtomicU64, Ordering},
        Arc,
    },
};

use axum::{extract::State, http::StatusCode, response::IntoResponse, routing::post, Json, Router};
use serde_json::{json, Value};
use tokio::sync::oneshot;

use crate::{error::HarnessError, Result};

struct Inner {
    runtime: tokio::runtime::Runtime,
    shutdown_tx: Option<oneshot::Sender<()>>,
    join_handle: Option<tokio::task::JoinHandle<()>>,
}

pub struct MockL1 {
    pub addr: SocketAddr,
    pub chain_id: u64,
    block_number: Arc<AtomicU64>,
    inner: Option<Inner>,
}

#[derive(Clone)]
struct AppState {
    chain_id: u64,
    block_number: Arc<AtomicU64>,
}

impl MockL1 {
    pub fn start(chain_id: u64) -> Result<Self> {
        let runtime = tokio::runtime::Builder::new_multi_thread()
            .enable_all()
            .worker_threads(2)
            .thread_name("mock-l1")
            .build()
            .map_err(HarnessError::Io)?;

        let block_number = Arc::new(AtomicU64::new(1));
        let state = AppState {
            chain_id,
            block_number: block_number.clone(),
        };

        let listener = runtime
            .block_on(async {
                tokio::net::TcpListener::bind(SocketAddr::from(([127, 0, 0, 1], 0))).await
            })
            .map_err(HarnessError::Io)?;
        let addr = listener.local_addr().map_err(HarnessError::Io)?;

        let app = Router::new().route("/", post(handle)).with_state(state);
        let (shutdown_tx, shutdown_rx) = oneshot::channel::<()>();

        let join_handle = runtime.spawn(async move {
            let _ = axum::serve(listener, app)
                .with_graceful_shutdown(async move {
                    let _ = shutdown_rx.await;
                })
                .await;
        });

        Ok(Self {
            addr,
            chain_id,
            block_number,
            inner: Some(Inner {
                runtime,
                shutdown_tx: Some(shutdown_tx),
                join_handle: Some(join_handle),
            }),
        })
    }

    pub fn rpc_url(&self) -> String {
        format!("http://{}", self.addr)
    }

    pub fn advance_block(&self) -> u64 {
        self.block_number.fetch_add(1, Ordering::SeqCst) + 1
    }

    pub fn shutdown(mut self) -> Result<()> {
        self.shutdown_inner()
    }

    fn shutdown_inner(&mut self) -> Result<()> {
        let Some(mut inner) = self.inner.take() else {
            return Ok(());
        };
        if let Some(tx) = inner.shutdown_tx.take() {
            let _ = tx.send(());
        }
        if let Some(handle) = inner.join_handle.take() {
            let _ = inner.runtime.block_on(handle);
        }
        Ok(())
    }
}

impl Drop for MockL1 {
    fn drop(&mut self) {
        let _ = self.shutdown_inner();
    }
}

async fn handle(State(state): State<AppState>, Json(req): Json<Value>) -> impl IntoResponse {
    let id = req.get("id").cloned().unwrap_or_else(|| json!(1));
    let method = req
        .get("method")
        .and_then(|v| v.as_str())
        .unwrap_or("")
        .to_string();
    let params = req
        .get("params")
        .cloned()
        .unwrap_or_else(|| Value::Array(vec![]));

    let result = match method.as_str() {
        "eth_chainId" => Ok(Value::String(format!("0x{:x}", state.chain_id))),
        "net_version" => Ok(Value::String(state.chain_id.to_string())),
        "eth_blockNumber" => Ok(Value::String(format!(
            "0x{:x}",
            state.block_number.load(Ordering::SeqCst)
        ))),
        "eth_getBlockByNumber" | "eth_getBlockByHash" => {
            let n = state.block_number.load(Ordering::SeqCst);
            Ok(stub_block(n))
        }
        "eth_call" => Ok(Value::String("0x".to_string())),
        "eth_getTransactionByHash" | "eth_getTransactionReceipt" => Ok(Value::Null),
        "eth_getLogs" => Ok(Value::Array(vec![])),
        "eth_syncing" => Ok(Value::Bool(false)),
        "eth_gasPrice" => Ok(Value::String("0x0".to_string())),
        "web3_clientVersion" => Ok(Value::String("arb-test-harness/mock-l1".to_string())),
        _ => Err(format!("method {method} not supported by mock L1")),
    };

    match result {
        Ok(v) => (
            StatusCode::OK,
            Json(json!({
                "jsonrpc": "2.0",
                "id": id,
                "result": v,
            })),
        ),
        Err(msg) => {
            let _ = params;
            (
                StatusCode::OK,
                Json(json!({
                    "jsonrpc": "2.0",
                    "id": id,
                    "error": {"code": -32601, "message": msg},
                })),
            )
        }
    }
}

fn stub_block(number: u64) -> Value {
    let zero = "0x0000000000000000000000000000000000000000000000000000000000000000";
    let zero_addr = "0x0000000000000000000000000000000000000000";
    json!({
        "number": format!("0x{:x}", number),
        "hash": zero,
        "parentHash": zero,
        "nonce": "0x0000000000000000",
        "sha3Uncles": zero,
        "logsBloom": format!("0x{}", "0".repeat(512)),
        "transactionsRoot": zero,
        "stateRoot": zero,
        "receiptsRoot": zero,
        "miner": zero_addr,
        "difficulty": "0x0",
        "totalDifficulty": "0x0",
        "extraData": "0x",
        "size": "0x0",
        "gasLimit": "0x1c9c380",
        "gasUsed": "0x0",
        "timestamp": "0x0",
        "transactions": [],
        "uncles": [],
        "baseFeePerGas": "0x1",
    })
}
