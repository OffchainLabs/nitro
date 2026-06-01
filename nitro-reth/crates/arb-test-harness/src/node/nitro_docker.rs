use std::{
    collections::BTreeMap,
    process::{Command, Stdio},
    time::{Duration, Instant},
};

use alloy_primitives::{Address, Bytes, B256, U256};
use serde_json::{json, Value};

use arb_node::genesis::INITIAL_ARBOS_VERSION;

use crate::{
    error::HarnessError,
    messaging::L1Message,
    node::{
        common::{
            arb_receipt_fields, block_from_json, receipt_from_json, tail, tx_request_to_json,
        },
        ArbReceiptFields, Block, BlockId, ExecutionNode, NodeKind, NodeStartCtx, TxReceipt,
        TxRequest,
    },
    rpc::JsonRpcClient,
    Result,
};

const DEFAULT_IMAGE: &str = "offchainlabs/nitro-node:v3.10.1-d7f07be";
const STARTUP_TIMEOUT: Duration = Duration::from_secs(60);

pub struct NitroDocker {
    rpc_url: String,
    rpc: JsonRpcClient,
    container_id: String,
}

impl NitroDocker {
    pub fn start(ctx: &NodeStartCtx) -> Result<Self> {
        crate::node::reaper::reap_orphan_containers();
        let image = std::env::var("NITRO_REF_IMAGE").unwrap_or_else(|_| DEFAULT_IMAGE.to_string());
        let seq = CONTAINER_SEQ.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
        let name = format!("arb-harness-nitro-{}-{}", std::process::id(), seq);

        let _ = Command::new("docker")
            .args(["rm", "-f", &name])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();

        let parent_chain_url = ctx.mock_l1_rpc.replace("127.0.0.1", "host.docker.internal");

        let chain_config = build_chain_config(ctx);
        // Seed Nitro's genesis from the fixture's alloc by writing a
        // geth-compatible genesis file and mounting it into the container.
        // Without this, --init.empty=true makes Nitro start from a chain-config-
        // only state, dropping fixture-defined accounts and rejecting any tx
        // whose sender lacks balance or has a fresh nonce. The conversion
        // matches the layout Nitro's init.go parses (serializedChainConfig
        // string + standard geth genesis fields + alloc).
        let seed_genesis = ctx.genesis.get("alloc").is_some();
        let genesis_host_dir = if seed_genesis {
            Some(write_nitro_genesis_file(ctx, &chain_config)?)
        } else {
            None
        };
        let chain_info_json = render_chain_info_json(ctx, &chain_config, seed_genesis);

        let owner_label = format!(
            "{}={}",
            crate::node::reaper::OWNER_PID_LABEL,
            std::process::id()
        );
        let mut cmd = Command::new("docker");
        cmd.args([
            "run",
            "-d",
            "--name",
            &name,
            "--label",
            &owner_label,
            "-p",
            "127.0.0.1::8547",
            "--user",
            "root",
        ]);
        if let Some(dir) = &genesis_host_dir {
            // Mount the directory holding genesis.json read-only into the container.
            cmd.arg("-v");
            cmd.arg(format!("{}:/arb-harness-genesis:ro", dir.display()));
        }
        cmd.args(["--entrypoint", "/usr/local/bin/nitro", &image]);
        if seed_genesis {
            cmd.arg("--init.genesis-json-file=/arb-harness-genesis/genesis.json");
        } else {
            cmd.arg("--init.empty=true");
        }
        cmd.args([
            "--init.validate-genesis-assertion=false",
            "--persistent.global-config=/tmp/nitro-data",
            "--node.parent-chain-reader.enable=false",
            "--node.dangerous.no-l1-listener=true",
            "--node.dangerous.disable-blob-reader=true",
            "--node.staker.enable=false",
            "--execution.forwarding-target=null",
            "--node.sequencer=false",
            "--node.batch-poster.enable=false",
            "--node.feed.input.url=",
            "--http.addr=0.0.0.0",
            "--http.port=8547",
            "--http.api=net,web3,eth,arb,debug,nitroexecution",
            "--http.vhosts=*",
            "--execution.rpc-server.enable=true",
            "--execution.rpc-server.public=true",
            "--execution.rpc-server.authenticated=false",
            "--log-level=WARN",
        ]);
        cmd.arg(format!("--chain.id={}", ctx.l2_chain_id));
        cmd.arg(format!("--chain.info-json={chain_info_json}"));
        cmd.arg(format!("--parent-chain.connection.url={parent_chain_url}"));
        cmd.arg(format!(
            "--parent-chain.blob-client.beacon-url={parent_chain_url}"
        ));

        let output = cmd
            .output()
            .map_err(|e| HarnessError::Rpc(format!("docker run nitro: {e}")))?;
        if !output.status.success() {
            return Err(HarnessError::Rpc(format!(
                "docker run nitro failed: {}",
                String::from_utf8_lossy(&output.stderr)
            )));
        }
        let container_id = String::from_utf8_lossy(&output.stdout).trim().to_string();

        let host_port = resolve_published_port(&name)?;
        let rpc_url = format!("http://127.0.0.1:{host_port}");
        let rpc = JsonRpcClient::new(rpc_url.clone()).with_timeout(Duration::from_secs(60));

        let deadline = Instant::now() + STARTUP_TIMEOUT;
        if let Err(e) = rpc.call_with_retry("eth_chainId", json!([]), deadline) {
            let logs = Command::new("docker")
                .args(["logs", "--tail=80", &name])
                .output()
                .map(|o| String::from_utf8_lossy(&o.stderr).into_owned())
                .unwrap_or_default();
            let _ = Command::new("docker").args(["rm", "-f", &name]).status();
            return Err(HarnessError::Rpc(format!(
                "nitro docker {rpc_url} did not respond within {:?}: {e}; logs:\n{}",
                STARTUP_TIMEOUT,
                tail(&logs, 4096)
            )));
        }

        Ok(Self {
            rpc_url,
            rpc,
            container_id,
        })
    }

    fn stop(&self) {
        let _ = Command::new("docker")
            .args(["rm", "-f", &self.container_id])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();
    }
}

impl Drop for NitroDocker {
    fn drop(&mut self) {
        self.stop();
    }
}

static CONTAINER_SEQ: std::sync::atomic::AtomicU32 = std::sync::atomic::AtomicU32::new(0);

fn resolve_published_port(container_name: &str) -> Result<u16> {
    let out = Command::new("docker")
        .args(["port", container_name, "8547"])
        .output()
        .map_err(|e| HarnessError::Rpc(format!("docker port: {e}")))?;
    if !out.status.success() {
        return Err(HarnessError::Rpc(format!(
            "docker port {container_name} 8547 failed: {}",
            String::from_utf8_lossy(&out.stderr)
        )));
    }
    let mapping = String::from_utf8_lossy(&out.stdout);
    for line in mapping.lines() {
        if let Some((_, port)) = line.rsplit_once(':') {
            if let Ok(p) = port.trim().parse::<u16>() {
                return Ok(p);
            }
        }
    }
    Err(HarnessError::Rpc(format!(
        "could not resolve published port from: {mapping}"
    )))
}

fn build_chain_config(ctx: &NodeStartCtx) -> Value {
    let mut chain_config = ctx
        .genesis
        .get("config")
        .cloned()
        .unwrap_or_else(|| json!({}));
    if let Some(map) = chain_config.as_object_mut() {
        let defaults = [
            ("chainId", json!(ctx.l2_chain_id)),
            ("homesteadBlock", json!(0)),
            ("daoForkSupport", json!(true)),
            ("eip150Block", json!(0)),
            ("eip155Block", json!(0)),
            ("eip158Block", json!(0)),
            ("byzantiumBlock", json!(0)),
            ("constantinopleBlock", json!(0)),
            ("petersburgBlock", json!(0)),
            ("istanbulBlock", json!(0)),
            ("muirGlacierBlock", json!(0)),
            ("berlinBlock", json!(0)),
            ("londonBlock", json!(0)),
            (
                "depositContractAddress",
                json!("0x0000000000000000000000000000000000000000"),
            ),
            ("clique", json!({"period": 0, "epoch": 0})),
        ];
        for (key, value) in defaults {
            map.entry(key.to_string()).or_insert(value);
        }
        if let Some(arb) = map
            .entry("arbitrum".to_string())
            .or_insert(json!({}))
            .as_object_mut()
        {
            let arb_defaults = [
                ("EnableArbOS", json!(true)),
                ("AllowDebugPrecompiles", json!(true)),
                ("DataAvailabilityCommittee", json!(false)),
                ("InitialArbOSVersion", json!(INITIAL_ARBOS_VERSION)),
                (
                    "InitialChainOwner",
                    json!("0x0000000000000000000000000000000000000000"),
                ),
                ("GenesisBlockNum", json!(0u64)),
            ];
            for (key, value) in arb_defaults {
                arb.entry(key.to_string()).or_insert(value);
            }
        }
    }
    chain_config
}

fn render_chain_info_json(ctx: &NodeStartCtx, chain_config: &Value, seed_genesis: bool) -> String {
    let entry = json!([{
        "chain-name": format!("arbreth-test-{}", ctx.l2_chain_id),
        "parent-chain-id": ctx.l1_chain_id,
        "parent-chain-is-arbitrum": false,
        // When seed_genesis is set, nitro loads init.genesis-json-file
        // and its state takes precedence over chain-config-only init.
        // has-genesis-state stays false so nitro reads our JSON path.
        "has-genesis-state": false,
        "chain-config": chain_config,
        "rollup": {
            "bridge": "0x0000000000000000000000000000000000000000",
            "inbox": "0x0000000000000000000000000000000000000000",
            "rollup": "0x0000000000000000000000000000000000000000",
            "sequencer-inbox": "0x0000000000000000000000000000000000000000",
            "validator-utils": "0x0000000000000000000000000000000000000000",
            "validator-wallet-creator": "0x0000000000000000000000000000000000000000",
            "deployed-at": 0,
        },
    }]);
    let _ = seed_genesis; // reserved for future use if the flag has to flip
    serde_json::to_string(&entry).unwrap_or_default()
}

/// Materialise a Nitro-compatible genesis file from the fixture's `genesis`
/// section and return the host directory containing `genesis.json` (which the
/// caller bind-mounts into the container). Nitro's `init.go::OpenInitDb`
/// expects:
///   - `serializedChainConfig`: stringified JSON of the chain config
///   - standard geth genesis fields (`alloc`, `gasLimit`, etc.)
fn write_nitro_genesis_file(
    ctx: &NodeStartCtx,
    chain_config: &Value,
) -> Result<std::path::PathBuf> {
    let dir = std::env::temp_dir().join(format!(
        "arb-harness-nitro-genesis-{}-{}",
        std::process::id(),
        CONTAINER_SEQ.load(std::sync::atomic::Ordering::SeqCst),
    ));
    std::fs::create_dir_all(&dir).map_err(HarnessError::Io)?;

    let alloc = ctx
        .genesis
        .get("alloc")
        .cloned()
        .unwrap_or_else(|| json!({}));

    // Pull through any geth fields the fixture supplied; default the rest.
    let take =
        |key: &str, default: Value| -> Value { ctx.genesis.get(key).cloned().unwrap_or(default) };

    let serialized_chain_config = serde_json::to_string(chain_config).unwrap_or_default();

    let genesis = json!({
        "serializedChainConfig": serialized_chain_config,
        "alloc": alloc,
        "nonce": take("nonce", json!("0x0")),
        "timestamp": take("timestamp", json!("0x0")),
        "extraData": take("extraData", json!("0x")),
        "gasLimit": take("gasLimit", json!("0x4000000000000")),
        "difficulty": take("difficulty", json!("0x1")),
        "mixHash": take("mixHash", json!("0x0000000000000000000000000000000000000000000000000000000000000000")),
        "coinbase": take("coinbase", json!("0x0000000000000000000000000000000000000000")),
        "number": take("number", json!("0x0")),
        "gasUsed": take("gasUsed", json!("0x0")),
        "parentHash": take("parentHash", json!("0x0000000000000000000000000000000000000000000000000000000000000000")),
        "baseFeePerGas": take("baseFeePerGas", json!("0x5f5e100")),
    });

    let path = dir.join("genesis.json");
    std::fs::write(
        &path,
        serde_json::to_vec_pretty(&genesis).unwrap_or_default(),
    )
    .map_err(HarnessError::Io)?;
    Ok(dir)
}

impl ExecutionNode for NitroDocker {
    fn kind(&self) -> NodeKind {
        NodeKind::NitroDocker
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
                "message": { "header": &msg.header, "l2Msg": &msg.l2_msg },
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
        crate::node::common::json_to_b256(&v)
    }

    fn balance(&self, addr: Address, at: BlockId) -> Result<U256> {
        let v = self
            .rpc
            .call("eth_getBalance", json!([format!("{addr:#x}"), at.to_rpc()]))?;
        crate::node::common::json_to_u256(&v)
    }

    fn nonce(&self, addr: Address, at: BlockId) -> Result<u64> {
        let v = self.rpc.call(
            "eth_getTransactionCount",
            json!([format!("{addr:#x}"), at.to_rpc()]),
        )?;
        crate::node::common::json_to_u64(&v)
    }

    fn code(&self, addr: Address, at: BlockId) -> Result<Bytes> {
        let v = self
            .rpc
            .call("eth_getCode", json!([format!("{addr:#x}"), at.to_rpc()]))?;
        crate::node::common::json_to_bytes(&v)
    }

    fn eth_call(&self, tx: TxRequest, at: BlockId) -> Result<Bytes> {
        let v = self
            .rpc
            .call("eth_call", json!([tx_request_to_json(&tx), at.to_rpc()]))?;
        crate::node::common::json_to_bytes(&v)
    }

    fn debug_storage_range(&self, addr: Address, at: BlockId) -> Result<BTreeMap<B256, B256>> {
        let block = self.block(at.clone())?;
        let mut out = BTreeMap::new();
        let mut start = B256::ZERO;
        loop {
            let v = self.rpc.call(
                "debug_storageRangeAt",
                json!([
                    format!("{:#x}", block.hash),
                    0,
                    format!("{addr:#x}"),
                    format!("{:#x}", start),
                    1024,
                ]),
            )?;
            let storage = v.get("storage").and_then(|s| s.as_object());
            if let Some(map) = storage {
                for entry in map.values() {
                    let key = entry.get("key").and_then(|x| x.as_str());
                    let val = entry.get("value").and_then(|x| x.as_str());
                    if let (Some(k), Some(vv)) = (key, val) {
                        let k = crate::node::common::parse_b256(k)?;
                        let vv = crate::node::common::parse_b256(vv)?;
                        out.insert(k, vv);
                    }
                }
            }
            let next = v.get("nextKey").and_then(|x| x.as_str());
            match next {
                Some(s) if !s.is_empty() && s != "0x0" && s != "null" => {
                    start = crate::node::common::parse_b256(s)?;
                }
                _ => break,
            }
        }
        Ok(out)
    }

    fn shutdown(self: Box<Self>) -> Result<()> {
        self.stop();
        Ok(())
    }
}
