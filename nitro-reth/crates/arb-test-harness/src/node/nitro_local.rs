use std::{
    collections::BTreeMap,
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

const NITRO_BINARY_ENV: &str = "NITRO_REF_BINARY";
const STARTUP_TIMEOUT: Duration = Duration::from_secs(30);

static NEXT_PORT: AtomicU16 = AtomicU16::new(48545);

pub struct NitroProcess {
    rpc_url: String,
    rpc: JsonRpcClient,
    workdir: PathBuf,
    child: Option<Child>,
}

impl NitroProcess {
    pub fn start(ctx: &NodeStartCtx) -> Result<Self> {
        let binary = match &ctx.binary {
            Some(b) => b.clone(),
            None => std::env::var(NITRO_BINARY_ENV).map_err(|_| HarnessError::MissingEnv {
                name: NITRO_BINARY_ENV,
            })?,
        };

        let workdir = if ctx.workdir.as_os_str().is_empty() {
            std::env::temp_dir().join(format!(
                "arb-harness-nitro-{}-{}",
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

        let datadir = workdir.join("data");
        std::fs::create_dir_all(&datadir).map_err(HarnessError::Io)?;
        let chain_info_path = workdir.join("chain-info.json");
        let chain_info = render_chain_info(ctx)?;
        std::fs::write(&chain_info_path, serde_json::to_vec_pretty(&chain_info)?)
            .map_err(HarnessError::Io)?;

        let http_port = if ctx.http_port == 0 {
            free_tcp_port(&NEXT_PORT)?
        } else {
            ctx.http_port
        };
        let ws_port = free_tcp_port(&NEXT_PORT)?;

        let stdout_path = workdir.join("stdout.log");
        let stderr_path = workdir.join("stderr.log");
        let stdout_file = std::fs::File::create(&stdout_path).map_err(HarnessError::Io)?;
        let stderr_file = std::fs::File::create(&stderr_path).map_err(HarnessError::Io)?;

        let mut cmd = Command::new(&binary);
        cmd.args([
            "--init.empty=true",
            "--init.validate-genesis-assertion=false",
            "--node.parent-chain-reader.enable=false",
            "--node.dangerous.no-l1-listener=true",
            "--node.dangerous.disable-blob-reader=true",
            "--node.staker.enable=false",
            "--execution.forwarding-target=null",
            "--node.sequencer=false",
            "--node.batch-poster.enable=false",
            "--node.feed.input.url=",
            "--execution.rpc-server.enable=true",
            "--execution.rpc-server.public=true",
            "--execution.rpc-server.authenticated=false",
            "--http",
            "--http.addr=127.0.0.1",
            "--http.api=eth,net,web3,debug,arb,nitroexecution",
            "--http.vhosts=*",
        ]);
        cmd.arg(format!("--http.port={http_port}"));
        cmd.arg(format!("--ws.port={ws_port}"));
        cmd.arg(format!("--chain.id={}", ctx.l2_chain_id));
        cmd.arg(format!(
            "--chain.info-files={}",
            chain_info_path.display()
        ));
        cmd.arg(format!(
            "--persistent.global-config={}",
            datadir.display()
        ));
        cmd.arg(format!(
            "--parent-chain.connection.url={}",
            ctx.mock_l1_rpc
        ));
        cmd.arg(format!(
            "--parent-chain.blob-client.beacon-url={}",
            ctx.mock_l1_rpc
        ));
        cmd.arg("--log-level=WARN");
        cmd.stdout(Stdio::from(stdout_file));
        cmd.stderr(Stdio::from(stderr_file));

        let child = cmd.spawn().map_err(|e| {
            HarnessError::Rpc(format!("spawn nitro at {binary}: {e}"))
        })?;

        let rpc_url = format!("http://127.0.0.1:{http_port}");
        let rpc = JsonRpcClient::new(rpc_url.clone()).with_timeout(Duration::from_secs(60));

        let deadline = Instant::now() + STARTUP_TIMEOUT;
        if let Err(e) = rpc.call_with_retry("eth_chainId", json!([]), deadline) {
            let stderr_tail = std::fs::read_to_string(&stderr_path).unwrap_or_default();
            return Err(HarnessError::Rpc(format!(
                "nitro at {rpc_url} did not respond within {:?}: {e}; stderr_tail:\n{}",
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

impl ExecutionNode for NitroProcess {
    fn kind(&self) -> NodeKind {
        NodeKind::NitroLocal
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

impl Drop for NitroProcess {
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

fn render_chain_info(ctx: &NodeStartCtx) -> Result<Value> {
    let chain_id = ctx.l2_chain_id;
    let parent_chain_id = ctx.l1_chain_id;

    let mut config = json!({
        "chainId": chain_id,
        "homesteadBlock": 0,
        "daoForkBlock": null,
        "daoForkSupport": true,
        "eip150Block": 0,
        "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "eip155Block": 0,
        "eip158Block": 0,
        "byzantiumBlock": 0,
        "constantinopleBlock": 0,
        "petersburgBlock": 0,
        "istanbulBlock": 0,
        "muirGlacierBlock": 0,
        "berlinBlock": 0,
        "londonBlock": 0,
        "clique": { "period": 0, "epoch": 0 },
        "arbitrum": {
            "EnableArbOS": true,
            "AllowDebugPrecompiles": true,
            "DataAvailabilityCommittee": false,
            "InitialArbOSVersion": 10,
            "InitialChainOwner": "0x71B61c2E250AFa05dFc36304D6c91501bE0965D8",
            "GenesisBlockNum": 0,
        }
    });

    if let Some(provided) = ctx.genesis.get("config") {
        if let Some(provided_arb) = provided.get("arbitrum") {
            if let Some(target_arb) = config.get_mut("arbitrum") {
                if let (Some(t), Some(p)) = (target_arb.as_object_mut(), provided_arb.as_object()) {
                    for (k, v) in p {
                        t.insert(k.clone(), v.clone());
                    }
                }
            }
        }
    }

    Ok(json!([{
        "chain-id": chain_id,
        "parent-chain-id": parent_chain_id,
        "parent-chain-is-arbitrum": false,
        "chain-name": format!("test-chain-{chain_id}"),
        "sequencer-url": "",
        "feed-url": "",
        "feed-signed": false,
        "chain-config": config,
        "rollup": {
            "bridge": "0x0000000000000000000000000000000000000000",
            "inbox": "0x0000000000000000000000000000000000000000",
            "sequencer-inbox": "0x0000000000000000000000000000000000000000",
            "rollup": "0x0000000000000000000000000000000000000000",
            "validator-utils": "0x0000000000000000000000000000000000000000",
            "validator-wallet-creator": "0x0000000000000000000000000000000000000000",
            "deployed-at": 0,
        },
    }]))
}
