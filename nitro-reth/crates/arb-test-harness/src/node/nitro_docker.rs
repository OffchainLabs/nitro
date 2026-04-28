use std::{
    collections::BTreeMap,
    path::PathBuf,
    process::{Command, Stdio},
    str::FromStr,
    time::{Duration, Instant},
};

use alloy_genesis::GenesisAccount;
use alloy_primitives::{hex, Address, Bytes, B256, U256};
use revm::database::{EmptyDB, State, StateBuilder};
use revm_database::states::bundle_state::BundleRetention;
use serde_json::{json, Map, Value};

use arb_node::genesis::{initialize_arbos_state, ArbOSInit, INITIAL_ARBOS_VERSION};
use arbos::arbos_types::ParsedInitMessage;

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

const DEFAULT_IMAGE: &str = "offchainlabs/nitro-node:v3.10.0-rc.2-746bda2";
const STARTUP_TIMEOUT: Duration = Duration::from_secs(60);

pub struct NitroDocker {
    rpc_url: String,
    rpc: JsonRpcClient,
    container_id: String,
    genesis_path: Option<PathBuf>,
}

impl NitroDocker {
    pub fn start(ctx: &NodeStartCtx) -> Result<Self> {
        let image =
            std::env::var("NITRO_REF_IMAGE").unwrap_or_else(|_| DEFAULT_IMAGE.to_string());
        let seq = CONTAINER_SEQ.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
        let name = format!("arb-harness-nitro-{}-{}", std::process::id(), seq);

        let _ = Command::new("docker")
            .args(["rm", "-f", &name])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();

        let parent_chain_url = ctx.mock_l1_rpc.replace("127.0.0.1", "host.docker.internal");

        let chain_config = build_chain_config(ctx);
        let chain_info_json = render_chain_info_json(ctx, &chain_config);
        let genesis_path = write_genesis_json_file(ctx, &chain_config, seq)?;
        let genesis_path_str = genesis_path
            .to_str()
            .ok_or_else(|| HarnessError::Rpc("genesis path is not valid UTF-8".into()))?
            .to_string();
        let genesis_mount = format!("{genesis_path_str}:/tmp/genesis.json:ro");

        let mut cmd = Command::new("docker");
        cmd.args([
            "run",
            "-d",
            "--name",
            &name,
            "-p",
            "127.0.0.1::8547",
            "--user",
            "root",
            "-v",
            &genesis_mount,
            "--entrypoint",
            "/usr/local/bin/nitro",
            &image,
            "--init.genesis-json-file=/tmp/genesis.json",
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

        let output = cmd.output().map_err(|e| {
            HarnessError::Rpc(format!("docker run nitro: {e}"))
        })?;
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
            genesis_path: Some(genesis_path),
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
        if let Some(path) = self.genesis_path.take() {
            if std::env::var("ARB_HARNESS_KEEP_WORKDIR").is_err() {
                let _ = std::fs::remove_file(path);
            }
        }
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

fn render_chain_info_json(ctx: &NodeStartCtx, chain_config: &Value) -> String {
    let entry = json!([{
        "chain-name": format!("arbreth-test-{}", ctx.l2_chain_id),
        "parent-chain-id": ctx.l1_chain_id,
        "parent-chain-is-arbitrum": false,
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
    serde_json::to_string(&entry).unwrap_or_default()
}

/// Compute the genesis alloc by running ArbOS init in a scratch state, and
/// write a geth-format genesis JSON file mountable into the Nitro container.
/// The same file can be passed to arbreth via `--chain=<path>` and will produce
/// an identical block-0 stateRoot.
fn write_genesis_json_file(
    ctx: &NodeStartCtx,
    chain_config: &Value,
    seq: u32,
) -> Result<PathBuf> {
    let chain_id = ctx.l2_chain_id;

    let arbos_version = chain_config
        .pointer("/arbitrum/InitialArbOSVersion")
        .and_then(Value::as_u64)
        .unwrap_or(INITIAL_ARBOS_VERSION);

    let chain_owner = chain_config
        .pointer("/arbitrum/InitialChainOwner")
        .and_then(Value::as_str)
        .and_then(|s| Address::from_str(s.trim_start_matches("0x")).ok())
        .unwrap_or(Address::ZERO);

    let arbos_init = ArbOSInit {
        native_token_supply_management_enabled: chain_config
            .pointer("/arbitrum/ArbOSInit/nativeTokenSupplyManagementEnabled")
            .or_else(|| chain_config.pointer("/arbitrum/nativeTokenSupplyManagementEnabled"))
            .and_then(Value::as_bool)
            .unwrap_or(false),
        transaction_filtering_enabled: chain_config
            .pointer("/arbitrum/ArbOSInit/transactionFilteringEnabled")
            .or_else(|| chain_config.pointer("/arbitrum/transactionFilteringEnabled"))
            .and_then(Value::as_bool)
            .unwrap_or(false),
    };

    // Match `arb_node::chainspec::compute_arbos_alloc` exactly: empty
    // serialized chain config and zero initial L1 base fee. The alloc we
    // emit then equals what arbreth's chainspec parser injects when given
    // the fixture's inline genesis.
    let init_msg = ParsedInitMessage {
        chain_id: U256::from(chain_id),
        initial_l1_base_fee: U256::ZERO,
        serialized_chain_config: Vec::new(),
    };

    let mut state: State<EmptyDB> = StateBuilder::new()
        .with_database(EmptyDB::default())
        .with_bundle_update()
        .build();

    initialize_arbos_state(
        &mut state,
        &init_msg,
        chain_id,
        arbos_version,
        chain_owner,
        arbos_init,
    )
    .map_err(|e| HarnessError::Rpc(format!("initialize_arbos_state: {e}")))?;

    state.merge_transitions(BundleRetention::PlainState);
    let bundle = state.take_bundle();

    // Build the alloc, retaining slots that were written to zero. Nitro's
    // `InitializeArbosInDatabase` runs its own ArbOS init before applying
    // the alloc overlay, so any slot it would set to a non-zero value
    // (e.g. `pricePerUnit` from `DefaultInitialL1BaseFee`) needs an
    // explicit zero entry here to be reset.
    let mut alloc: BTreeMap<String, Value> = BTreeMap::new();
    for (addr, account) in bundle.state.iter() {
        let info = match &account.info {
            Some(i) => i,
            None => continue,
        };
        let mut storage = BTreeMap::new();
        for (slot, slot_value) in account.storage.iter() {
            storage.insert(
                B256::from(slot.to_be_bytes::<32>()),
                B256::from(slot_value.present_value.to_be_bytes::<32>()),
            );
        }
        let code = match &info.code {
            Some(c) if !c.original_bytes().is_empty() => Some(c.original_bytes()),
            _ => None,
        };
        let entry = GenesisAccount {
            balance: info.balance,
            nonce: Some(info.nonce),
            code,
            storage: if storage.is_empty() {
                None
            } else {
                Some(storage)
            },
            private_key: None,
        };
        let key = format!("{:#x}", addr);
        alloc.insert(key, account_to_geth_json(&entry));
    }

    // Re-serialize the chain config so Nitro can read it via `gen.GetConfig()`.
    // Nitro requires a valid `serializedChainConfig` field to deserialize the
    // chain config when the parent-chain reader is disabled (it constructs a
    // fake init message from this).
    let serialized_chain_config_bytes = serde_json::to_vec(chain_config)
        .map_err(|e| HarnessError::Rpc(format!("encode chain config: {e}")))?;
    let serialized_chain_config_str = String::from_utf8(serialized_chain_config_bytes)
        .map_err(|e| HarnessError::Rpc(format!("chain config not utf-8: {e}")))?;

    let genesis = json!({
        "config": chain_config,
        "serializedChainConfig": serialized_chain_config_str,
        "nonce": "0x0",
        "timestamp": "0x0",
        "extraData": "0x",
        "gasLimit": "0x4000000000000",
        "difficulty": "0x1",
        "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "coinbase": "0x0000000000000000000000000000000000000000",
        "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "alloc": alloc,
    });

    let path = std::env::temp_dir().join(format!(
        "arb-harness-nitro-genesis-{}-{}.json",
        std::process::id(),
        seq
    ));
    let body = serde_json::to_vec_pretty(&genesis)
        .map_err(|e| HarnessError::Rpc(format!("encode genesis: {e}")))?;
    std::fs::write(&path, body).map_err(HarnessError::Io)?;

    // Geth's GenesisAlloc unmarshaller needs world-readable content because
    // Nitro runs inside the container under root and we bind-mount this file.
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let _ = std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o644));
    }

    Ok(path)
}

/// Encode a `GenesisAccount` as the geth `core.Genesis` `alloc` entry shape:
/// keys are decimal-or-hex JSON-friendly strings, balance is hex, nonce is
/// hex, code is hex bytes, storage values are 32-byte hex.
fn account_to_geth_json(account: &GenesisAccount) -> Value {
    let mut entry = Map::new();
    entry.insert("balance".into(), Value::String(format!("{:#x}", account.balance)));
    if let Some(nonce) = account.nonce {
        if nonce > 0 {
            entry.insert("nonce".into(), Value::String(format!("{nonce:#x}")));
        }
    }
    if let Some(code) = &account.code {
        if !code.is_empty() {
            entry.insert(
                "code".into(),
                Value::String(format!("0x{}", hex::encode(code))),
            );
        }
    }
    if let Some(storage) = &account.storage {
        if !storage.is_empty() {
            let mut sm = Map::new();
            for (slot, val) in storage {
                sm.insert(format!("{:#x}", slot), Value::String(format!("{:#x}", val)));
            }
            entry.insert("storage".into(), Value::Object(sm));
        }
    }
    Value::Object(entry)
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

    fn debug_storage_range(
        &self,
        _addr: Address,
        _at: BlockId,
    ) -> Result<BTreeMap<B256, B256>> {
        Err(HarnessError::NotImplemented {
            what: "NitroDocker::debug_storage_range",
        })
    }

    fn shutdown(self: Box<Self>) -> Result<()> {
        self.stop();
        Ok(())
    }
}
