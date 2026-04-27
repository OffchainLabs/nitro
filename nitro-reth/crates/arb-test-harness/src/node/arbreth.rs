use std::{
    collections::BTreeMap,
    io::Write,
    path::PathBuf,
    process::{Child, Command, Stdio},
    sync::atomic::{AtomicU16, Ordering},
    time::{Duration, Instant},
};

use alloy_primitives::{Address, Bytes, B256, U256};
use serde_json::{json, Value};

use super::common::{
    arb_receipt_fields, block_from_json, free_tcp_port, json_to_b256, json_to_bytes,
    json_to_u256, json_to_u64, parse_b256, receipt_from_json, tail, tx_request_to_json,
};
use crate::{
    error::HarnessError,
    messaging::L1Message,
    node::{
        ArbReceiptFields, Block, BlockId, ExecutionNode, NodeKind, NodeStartCtx, TxReceipt,
        TxRequest,
    },
    rpc::JsonRpcClient,
    Result,
};

const ARB_BINARY_ENV: &str = "ARB_SPEC_BINARY";
const STARTUP_TIMEOUT: Duration = Duration::from_secs(30);

static NEXT_PORT: AtomicU16 = AtomicU16::new(38545);

pub struct ArbrethProcess {
    rpc_url: String,
    rpc: JsonRpcClient,
    workdir: PathBuf,
    child: Option<Child>,
}

impl ArbrethProcess {
    pub fn start(ctx: &NodeStartCtx) -> Result<Self> {
        let binary = match &ctx.binary {
            Some(b) => b.clone(),
            None => std::env::var(ARB_BINARY_ENV).map_err(|_| HarnessError::MissingEnv {
                name: ARB_BINARY_ENV,
            })?,
        };

        let workdir = if ctx.workdir.as_os_str().is_empty() {
            std::env::temp_dir().join(format!(
                "arb-harness-arbreth-{}-{}",
                std::process::id(),
                NEXT_PORT.fetch_add(0, Ordering::SeqCst)
            ))
        } else {
            ctx.workdir.clone()
        };
        if workdir.exists() {
            let _ = std::fs::remove_dir_all(&workdir);
        }
        std::fs::create_dir_all(&workdir).map_err(HarnessError::Io)?;

        let chain_path = workdir.join("chain.json");
        let chain_bytes = serde_json::to_vec_pretty(&ctx.genesis)?;
        std::fs::File::create(&chain_path)
            .and_then(|mut f| f.write_all(&chain_bytes))
            .map_err(HarnessError::Io)?;

        let jwt_path = workdir.join("jwt.hex");
        let jwt_hex = if ctx.jwt_hex.is_empty() {
            hex::encode([0u8; 32])
        } else {
            ctx.jwt_hex.clone()
        };
        std::fs::write(&jwt_path, &jwt_hex).map_err(HarnessError::Io)?;

        let http_port = if ctx.http_port == 0 {
            free_tcp_port(&NEXT_PORT)?
        } else {
            ctx.http_port
        };
        let auth_port = if ctx.authrpc_port == 0 {
            free_tcp_port(&NEXT_PORT)?
        } else {
            ctx.authrpc_port
        };

        let stdout_path = workdir.join("stdout.log");
        let stderr_path = workdir.join("stderr.log");
        let stdout_file = std::fs::File::create(&stdout_path).map_err(HarnessError::Io)?;
        let stderr_file = std::fs::File::create(&stderr_path).map_err(HarnessError::Io)?;

        let datadir = workdir.join("db");

        let child = Command::new(&binary)
            .env(
                "RUST_LOG",
                std::env::var("ARB_HARNESS_RUST_LOG")
                    .unwrap_or_else(|_| "info,block_producer=warn".to_string()),
            )
            .arg("node")
            .arg(format!("--chain={}", chain_path.display()))
            .arg(format!("--datadir={}", datadir.display()))
            .arg("--http")
            .arg("--http.addr=127.0.0.1")
            .arg(format!("--http.port={http_port}"))
            .arg("--http.api=eth,web3,net,debug")
            .arg("--authrpc.addr=127.0.0.1")
            .arg(format!("--authrpc.port={auth_port}"))
            .arg(format!("--authrpc.jwtsecret={}", jwt_path.display()))
            .arg("--disable-discovery")
            .arg("--db.exclusive=true")
            .stdout(Stdio::from(stdout_file))
            .stderr(Stdio::from(stderr_file))
            .spawn()
            .map_err(|e| HarnessError::Rpc(format!("spawn arbreth at {binary}: {e}")))?;

        let rpc_url = format!("http://127.0.0.1:{http_port}");
        let rpc = JsonRpcClient::new(rpc_url.clone()).with_timeout(Duration::from_secs(60));

        let deadline = Instant::now() + STARTUP_TIMEOUT;
        if let Err(e) = rpc.call_with_retry("eth_chainId", json!([]), deadline) {
            let stderr_tail = std::fs::read_to_string(&stderr_path).unwrap_or_default();
            return Err(HarnessError::Rpc(format!(
                "arbreth at {rpc_url} did not respond within {:?}: {e}; stderr_tail:\n{}",
                STARTUP_TIMEOUT,
                tail(&stderr_tail, 4096)
            )));
        }

        Ok(Self {
            rpc_url,
            rpc,
            workdir,
            child: Some(child),
        })
    }
}

impl ExecutionNode for ArbrethProcess {
    fn kind(&self) -> NodeKind {
        NodeKind::Arbreth
    }

    fn rpc_url(&self) -> &str {
        &self.rpc_url
    }

    fn submit_message(
        &mut self,
        idx: u64,
        msg: &L1Message,
        delayed_messages_read: u64,
    ) -> Result<()> {
        let params = json!([
            idx,
            {
                "message": {
                    "header": &msg.header,
                    "l2Msg": &msg.l2_msg,
                },
                "delayedMessagesRead": delayed_messages_read,
            },
            Value::Null,
        ]);
        self.rpc.call("nitroexecution_digestMessage", params)?;
        Ok(())
    }

    fn block(&self, id: BlockId) -> Result<Block> {
        let v = self
            .rpc
            .call("eth_getBlockByNumber", json!([id.to_rpc(), false]))?;
        block_from_json(&v)
    }

    fn receipt(&self, tx: B256) -> Result<TxReceipt> {
        let v = self
            .rpc
            .call("eth_getTransactionReceipt", json!([format!("{tx:#x}")]))?;
        receipt_from_json(&v)
    }

    fn arb_receipt(&self, tx: B256) -> Result<ArbReceiptFields> {
        let v = self
            .rpc
            .call("eth_getTransactionReceipt", json!([format!("{tx:#x}")]))?;
        Ok(arb_receipt_fields(&v))
    }

    fn storage(&self, addr: Address, slot: B256, at: BlockId) -> Result<B256> {
        let v = self.rpc.call(
            "eth_getStorageAt",
            json!([format!("{addr:#x}"), format!("{slot:#x}"), at.to_rpc()]),
        )?;
        json_to_b256(&v)
    }

    fn balance(&self, addr: Address, at: BlockId) -> Result<U256> {
        let v = self.rpc.call(
            "eth_getBalance",
            json!([format!("{addr:#x}"), at.to_rpc()]),
        )?;
        json_to_u256(&v)
    }

    fn nonce(&self, addr: Address, at: BlockId) -> Result<u64> {
        let v = self.rpc.call(
            "eth_getTransactionCount",
            json!([format!("{addr:#x}"), at.to_rpc()]),
        )?;
        json_to_u64(&v)
    }

    fn code(&self, addr: Address, at: BlockId) -> Result<Bytes> {
        let v = self.rpc.call(
            "eth_getCode",
            json!([format!("{addr:#x}"), at.to_rpc()]),
        )?;
        json_to_bytes(&v)
    }

    fn eth_call(&self, tx: TxRequest, at: BlockId) -> Result<Bytes> {
        let v = self.rpc.call(
            "eth_call",
            json!([tx_request_to_json(&tx), at.to_rpc()]),
        )?;
        json_to_bytes(&v)
    }

    fn debug_storage_range(
        &self,
        addr: Address,
        at: BlockId,
    ) -> Result<BTreeMap<B256, B256>> {
        let block = self.block(at.clone())?;
        let v = self.rpc.call(
            "debug_storageRangeAt",
            json!([
                format!("{:#x}", block.hash),
                0,
                format!("{addr:#x}"),
                format!("{:#x}", B256::ZERO),
                u32::MAX,
            ]),
        )?;
        let mut out = BTreeMap::new();
        if let Some(map) = v.get("storage").and_then(|s| s.as_object()) {
            for entry in map.values() {
                let key = entry.get("key").and_then(|x| x.as_str());
                let val = entry.get("value").and_then(|x| x.as_str());
                if let (Some(k), Some(v)) = (key, val) {
                    let k = parse_b256(k)?;
                    let v = parse_b256(v)?;
                    out.insert(k, v);
                }
            }
        }
        Ok(out)
    }

    fn shutdown(self: Box<Self>) -> Result<()> {
        let _ = self;
        Ok(())
    }
}

impl Drop for ArbrethProcess {
    fn drop(&mut self) {
        if let Some(mut child) = self.child.take() {
            let _ = child.kill();
            let _ = child.wait();
        }
        if std::env::var("ARB_HARNESS_KEEP_WORKDIR").is_err() {
            let _ = std::fs::remove_dir_all(&self.workdir);
        }
    }
}
