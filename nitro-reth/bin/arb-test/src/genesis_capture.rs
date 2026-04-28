use std::{
    collections::BTreeMap,
    process::{Command, Stdio},
    str::FromStr,
    time::{Duration, Instant},
};

use alloy_primitives::{Address, B256, U256};
use anyhow::{anyhow, bail, Context, Result};
use serde_json::{json, Map, Value};

use arb_test_harness::rpc::JsonRpcClient;

const DEFAULT_IMAGE: &str = "offchainlabs/nitro-node:v3.10.0-rc.2-746bda2";
const STARTUP_TIMEOUT: Duration = Duration::from_secs(60);
const RPC_TIMEOUT: Duration = Duration::from_secs(60);

/// Empty trie root (`keccak256(rlp(""))`).
const EMPTY_STORAGE_HASH: B256 = B256::new([
    0x56, 0xe8, 0x1f, 0x17, 0x1b, 0xcc, 0x55, 0xa6, 0xff, 0x83, 0x45, 0xe6, 0x92, 0xc0, 0xf8, 0x6e,
    0x5b, 0x48, 0xe0, 0x1b, 0x99, 0x6c, 0xad, 0xc0, 0x01, 0x62, 0x2f, 0xb5, 0xe3, 0x63, 0xb4, 0x21,
]);

/// Capture a reference node's full block-0 state and emit a geth-format genesis JSON.
#[derive(Debug, clap::Args)]
pub struct GenesisCaptureArgs {
    /// L2 chain id to bake into the chain config.
    #[arg(long)]
    pub chain_id: u64,

    /// ArbOS version to initialize (e.g. 10, 32, 50, 60).
    #[arg(long)]
    pub arbos_version: u64,

    /// Output path for the generated genesis JSON.
    #[arg(long)]
    pub out: std::path::PathBuf,

    /// Optional: address that owns the chain post-init (defaults to zero).
    #[arg(long, default_value = "0x0000000000000000000000000000000000000000")]
    pub initial_chain_owner: String,

    /// Whether `AllowDebugPrecompiles` should be set in the chain config.
    #[arg(long, default_value_t = true)]
    pub allow_debug_precompiles: bool,

    /// Override the docker image. Defaults to a pinned reference release.
    #[arg(long)]
    pub nitro_image: Option<String>,
}

pub fn run(cli: GenesisCaptureArgs) -> Result<()> {
    let chain_owner = Address::from_str(cli.initial_chain_owner.trim_start_matches("0x"))
        .context("invalid --initial-chain-owner")?;
    let image = cli
        .nitro_image
        .clone()
        .or_else(|| std::env::var("NITRO_REF_IMAGE").ok())
        .unwrap_or_else(|| DEFAULT_IMAGE.to_string());

    let runner = NitroRunner::start(
        &image,
        cli.chain_id,
        cli.arbos_version,
        chain_owner,
        cli.allow_debug_precompiles,
    )?;

    let result = (|| -> Result<()> {
        let alloc = capture_alloc(&runner.rpc, chain_owner)?;
        let entry_count = alloc.len();
        let genesis = build_genesis_json(
            cli.chain_id,
            cli.arbos_version,
            chain_owner,
            cli.allow_debug_precompiles,
            alloc,
        );
        let body = serde_json::to_vec_pretty(&genesis)
            .context("serialize genesis json")?;
        std::fs::write(&cli.out, body)
            .with_context(|| format!("write {}", cli.out.display()))?;
        eprintln!(
            "wrote {} ({} alloc entries)",
            cli.out.display(),
            entry_count
        );
        Ok(())
    })();

    drop(runner);
    result
}

struct NitroRunner {
    rpc: JsonRpcClient,
    container_name: String,
}

impl NitroRunner {
    fn start(
        image: &str,
        chain_id: u64,
        arbos_version: u64,
        chain_owner: Address,
        allow_debug_precompiles: bool,
    ) -> Result<Self> {
        let name = format!(
            "arb-genesis-capture-{}-{}",
            std::process::id(),
            now_nanos()
        );
        let _ = Command::new("docker")
            .args(["rm", "-f", &name])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();

        let chain_info_json = render_chain_info_json(
            chain_id,
            arbos_version,
            chain_owner,
            allow_debug_precompiles,
        );

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
            "--entrypoint",
            "/usr/local/bin/nitro",
            image,
            "--init.empty=true",
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
            "--execution.caching.enable-preimages=true",
            "--http.addr=0.0.0.0",
            "--http.port=8547",
            "--http.api=net,web3,eth,arb,debug,nitroexecution",
            "--http.vhosts=*",
            "--execution.rpc-server.enable=true",
            "--execution.rpc-server.public=true",
            "--execution.rpc-server.authenticated=false",
            "--log-level=WARN",
        ]);
        cmd.arg(format!("--chain.id={chain_id}"));
        cmd.arg(format!("--chain.info-json={chain_info_json}"));

        let output = cmd
            .output()
            .context("invoke docker run for nitro container")?;
        if !output.status.success() {
            bail!(
                "docker run failed: {}",
                String::from_utf8_lossy(&output.stderr).trim()
            );
        }

        let host_port = resolve_published_port(&name)
            .with_context(|| format!("resolve published RPC port for {name}"))?;
        let rpc_url = format!("http://127.0.0.1:{host_port}");
        let rpc = JsonRpcClient::new(rpc_url.clone()).with_timeout(RPC_TIMEOUT);

        let deadline = Instant::now() + STARTUP_TIMEOUT;
        if let Err(e) = rpc.call_with_retry("eth_chainId", json!([]), deadline) {
            let logs = Command::new("docker")
                .args(["logs", "--tail=120", &name])
                .output()
                .map(|o| String::from_utf8_lossy(&o.stderr).into_owned())
                .unwrap_or_default();
            let _ = Command::new("docker").args(["rm", "-f", &name]).status();
            bail!(
                "nitro at {rpc_url} did not respond within {:?}: {e}\nlogs (tail):\n{}",
                STARTUP_TIMEOUT,
                logs
            );
        }

        Ok(Self {
            rpc,
            container_name: name,
        })
    }
}

impl Drop for NitroRunner {
    fn drop(&mut self) {
        let _ = Command::new("docker")
            .args(["rm", "-f", &self.container_name])
            .stdout(Stdio::null())
            .stderr(Stdio::null())
            .status();
    }
}

fn now_nanos() -> u128 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0)
}

fn resolve_published_port(container_name: &str) -> Result<u16> {
    let deadline = Instant::now() + Duration::from_secs(10);
    loop {
        let out = Command::new("docker")
            .args(["port", container_name, "8547"])
            .output()
            .context("docker port")?;
        if out.status.success() {
            let mapping = String::from_utf8_lossy(&out.stdout);
            for line in mapping.lines() {
                if let Some((_, port)) = line.rsplit_once(':') {
                    if let Ok(p) = port.trim().parse::<u16>() {
                        return Ok(p);
                    }
                }
            }
        }
        if Instant::now() >= deadline {
            return Err(anyhow!(
                "could not resolve published port for {container_name}"
            ));
        }
        std::thread::sleep(Duration::from_millis(200));
    }
}

fn render_chain_info_json(
    chain_id: u64,
    arbos_version: u64,
    chain_owner: Address,
    allow_debug_precompiles: bool,
) -> String {
    let chain_config = json!({
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
            "AllowDebugPrecompiles": allow_debug_precompiles,
            "DataAvailabilityCommittee": false,
            "InitialArbOSVersion": arbos_version,
            "InitialChainOwner": format!("{chain_owner:#x}"),
            "GenesisBlockNum": 0u64,
        },
    });

    let entry = json!([{
        "chain-id": chain_id,
        "parent-chain-id": 1u64,
        "parent-chain-is-arbitrum": false,
        "chain-name": format!("arb-genesis-capture-{chain_id}"),
        "sequencer-url": "",
        "feed-url": "",
        "feed-signed": false,
        "chain-config": chain_config,
        "rollup": {
            "bridge": "0x0000000000000000000000000000000000000000",
            "inbox": "0x0000000000000000000000000000000000000000",
            "sequencer-inbox": "0x0000000000000000000000000000000000000000",
            "rollup": "0x0000000000000000000000000000000000000000",
            "validator-utils": "0x0000000000000000000000000000000000000000",
            "validator-wallet-creator": "0x0000000000000000000000000000000000000000",
            "deployed-at": 0,
        },
    }]);

    serde_json::to_string(&entry).unwrap_or_default()
}

fn capture_alloc(
    rpc: &JsonRpcClient,
    chain_owner: Address,
) -> Result<BTreeMap<String, Value>> {
    if let Some(alloc) = try_debug_dump_block(rpc)? {
        if !alloc.is_empty() {
            return Ok(alloc);
        }
        eprintln!("debug_dumpBlock returned no accounts; falling back to manual enumeration");
    }
    enumerate_alloc(rpc, chain_owner)
}

fn try_debug_dump_block(rpc: &JsonRpcClient) -> Result<Option<BTreeMap<String, Value>>> {
    let v = match rpc.call("debug_dumpBlock", json!(["0x0"])) {
        Ok(v) => v,
        Err(e) => {
            let msg = e.to_string().to_lowercase();
            if msg.contains("not found")
                || msg.contains("not available")
                || msg.contains("does not exist")
                || msg.contains("not supported")
                || msg.contains("method handler crashed")
            {
                return Ok(None);
            }
            return Err(anyhow!("debug_dumpBlock: {e}"));
        }
    };

    let accounts = match v.get("accounts").and_then(Value::as_object) {
        Some(a) => a,
        None => return Ok(None),
    };

    let mut alloc = BTreeMap::new();
    for (addr_key, raw) in accounts {
        let addr = match parse_dump_address(addr_key) {
            Ok(a) => a,
            Err(_) => continue,
        };
        let entry = parse_dump_account(raw)
            .with_context(|| format!("parse dump account for {addr_key}"))?;
        if entry.is_empty() {
            continue;
        }
        alloc.insert(format!("{addr:#x}"), Value::Object(entry));
    }

    Ok(Some(alloc))
}

fn parse_dump_address(key: &str) -> Result<Address> {
    let s = key.trim_start_matches("0x");
    if s.len() != 40 {
        return Err(anyhow!("expected 40-char address, got {} chars", s.len()));
    }
    Address::from_str(s).map_err(|e| anyhow!("{e}"))
}

fn parse_dump_account(raw: &Value) -> Result<Map<String, Value>> {
    let mut entry = Map::new();
    let balance = raw
        .get("balance")
        .and_then(Value::as_str)
        .ok_or_else(|| anyhow!("account missing balance"))?;
    let balance_u256 = parse_decimal_or_hex_u256(balance)
        .with_context(|| format!("parse balance {balance}"))?;
    entry.insert("balance".into(), Value::String(format!("{balance_u256:#x}")));

    let nonce_opt = raw.get("nonce").and_then(|n| {
        n.as_u64().or_else(|| {
            n.as_str()
                .and_then(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).ok())
        })
    });
    if let Some(n) = nonce_opt {
        if n > 0 {
            entry.insert("nonce".into(), Value::String(format!("{n:#x}")));
        }
    }

    if let Some(code) = raw.get("code").and_then(Value::as_str) {
        let trimmed = code.trim_start_matches("0x");
        if !trimmed.is_empty() && trimmed != "0" {
            entry.insert("code".into(), Value::String(format!("0x{trimmed}")));
        }
    }

    if let Some(storage) = raw.get("storage").and_then(Value::as_object) {
        let mut sm = Map::new();
        for (slot_key, slot_val) in storage {
            let val_str = slot_val
                .as_str()
                .ok_or_else(|| anyhow!("storage value not string"))?
                .trim_start_matches("0x");
            if val_str.trim_start_matches('0').is_empty() {
                continue;
            }
            let slot = pad_b256_hex(slot_key);
            let val = pad_b256_hex(val_str);
            sm.insert(slot, Value::String(val));
        }
        if !sm.is_empty() {
            entry.insert("storage".into(), Value::Object(sm));
        }
    }

    Ok(entry)
}

fn pad_b256_hex(s: &str) -> String {
    let s = s.trim_start_matches("0x");
    if s.len() >= 64 {
        format!("0x{}", &s[s.len() - 64..])
    } else {
        format!("0x{:0>64}", s)
    }
}

fn parse_decimal_or_hex_u256(s: &str) -> Result<U256> {
    let s = s.trim();
    if let Some(rest) = s.strip_prefix("0x").or_else(|| s.strip_prefix("0X")) {
        return U256::from_str_radix(rest, 16).map_err(|e| anyhow!("hex u256: {e}"));
    }
    U256::from_str_radix(s, 10).map_err(|e| anyhow!("decimal u256: {e}"))
}

fn enumerate_alloc(
    rpc: &JsonRpcClient,
    chain_owner: Address,
) -> Result<BTreeMap<String, Value>> {
    let block_zero_hash = block_zero_hash(rpc)?;

    let mut targets: Vec<Address> = Vec::new();
    for byte in 0x64u8..=0x74u8 {
        targets.push(precompile_address(byte));
    }
    targets.push(precompile_address(0xff));
    targets.push(parse_addr_unchecked("0x00000000000000000000000000000000000a4b05"));
    targets.push(parse_addr_unchecked("0xa4b05fffffffffffffffffffffffffffffffffff"));
    if !chain_owner.is_zero() {
        targets.push(chain_owner);
    }

    let mut alloc = BTreeMap::new();
    for addr in targets {
        if let Some(entry) = capture_account(rpc, addr, &block_zero_hash)? {
            alloc.insert(format!("{addr:#x}"), Value::Object(entry));
        }
    }
    Ok(alloc)
}

fn block_zero_hash(rpc: &JsonRpcClient) -> Result<String> {
    let v = rpc
        .call("eth_getBlockByNumber", json!(["0x0", false]))
        .map_err(|e| anyhow!("eth_getBlockByNumber: {e}"))?;
    let hash = v
        .get("hash")
        .and_then(Value::as_str)
        .ok_or_else(|| anyhow!("block 0 has no hash"))?;
    Ok(hash.to_string())
}

fn capture_account(
    rpc: &JsonRpcClient,
    addr: Address,
    block_hash: &str,
) -> Result<Option<Map<String, Value>>> {
    let bal = rpc
        .call("eth_getBalance", json!([format!("{addr:#x}"), "0x0"]))
        .map_err(|e| anyhow!("eth_getBalance({addr}): {e}"))?;
    let balance_u256 = parse_decimal_or_hex_u256(bal.as_str().unwrap_or("0x0"))?;

    let nonce_v = rpc
        .call(
            "eth_getTransactionCount",
            json!([format!("{addr:#x}"), "0x0"]),
        )
        .map_err(|e| anyhow!("eth_getTransactionCount({addr}): {e}"))?;
    let nonce = nonce_v
        .as_str()
        .and_then(|s| u64::from_str_radix(s.trim_start_matches("0x"), 16).ok())
        .unwrap_or(0);

    let code_v = rpc
        .call("eth_getCode", json!([format!("{addr:#x}"), "0x0"]))
        .map_err(|e| anyhow!("eth_getCode({addr}): {e}"))?;
    let code = code_v.as_str().unwrap_or("0x");
    let code_trimmed = code.trim_start_matches("0x");
    let has_code = !code_trimmed.is_empty() && code_trimmed != "0";

    let storage_hash = account_storage_hash(rpc, addr).unwrap_or(B256::ZERO);
    let storage = if storage_hash != EMPTY_STORAGE_HASH && storage_hash != B256::ZERO {
        enumerate_storage(rpc, addr, block_hash).unwrap_or_default()
    } else {
        BTreeMap::new()
    };

    if balance_u256.is_zero() && nonce == 0 && !has_code && storage.is_empty() {
        return Ok(None);
    }

    let mut entry = Map::new();
    entry.insert("balance".into(), Value::String(format!("{balance_u256:#x}")));
    if nonce > 0 {
        entry.insert("nonce".into(), Value::String(format!("{nonce:#x}")));
    }
    if has_code {
        entry.insert("code".into(), Value::String(format!("0x{code_trimmed}")));
    }
    if !storage.is_empty() {
        let mut sm = Map::new();
        for (slot, val) in storage {
            sm.insert(format!("{slot:#x}"), Value::String(format!("{val:#x}")));
        }
        entry.insert("storage".into(), Value::Object(sm));
    }
    Ok(Some(entry))
}

fn account_storage_hash(rpc: &JsonRpcClient, addr: Address) -> Result<B256> {
    let v = rpc
        .call("eth_getProof", json!([format!("{addr:#x}"), [], "0x0"]))
        .map_err(|e| anyhow!("eth_getProof({addr}): {e}"))?;
    let s = v
        .get("storageHash")
        .and_then(Value::as_str)
        .ok_or_else(|| anyhow!("getProof missing storageHash"))?;
    B256::from_str(s).map_err(|e| anyhow!("parse storageHash {s}: {e}"))
}

fn enumerate_storage(
    rpc: &JsonRpcClient,
    addr: Address,
    block_hash: &str,
) -> Result<BTreeMap<B256, B256>> {
    let mut out = BTreeMap::new();
    let mut start = B256::ZERO;
    loop {
        let v = match rpc.call(
            "debug_storageRangeAt",
            json!([
                block_hash,
                0,
                format!("{addr:#x}"),
                format!("{start:#x}"),
                4096
            ]),
        ) {
            Ok(v) => v,
            Err(e) => {
                eprintln!(
                    "warn: debug_storageRangeAt({addr:#x}) failed: {e}; \
                     skipping storage enumeration"
                );
                return Ok(out);
            }
        };

        let storage = match v.get("storage").and_then(|s| s.as_object()) {
            Some(s) => s,
            None => break,
        };
        if storage.is_empty() {
            break;
        }
        for entry in storage.values() {
            let key = entry.get("key").and_then(Value::as_str);
            let val = entry.get("value").and_then(Value::as_str);
            if let (Some(k), Some(v)) = (key, val) {
                let k = match B256::from_str(k) {
                    Ok(b) => b,
                    Err(_) => continue,
                };
                let v = match B256::from_str(v) {
                    Ok(b) => b,
                    Err(_) => continue,
                };
                if v != B256::ZERO {
                    out.insert(k, v);
                }
            }
        }
        let next = v.get("nextKey").and_then(Value::as_str);
        match next {
            Some(s) if s != "null" => match B256::from_str(s) {
                Ok(b) if b != B256::ZERO => start = b,
                _ => break,
            },
            _ => break,
        }
    }
    Ok(out)
}

fn precompile_address(byte: u8) -> Address {
    let mut bytes = [0u8; 20];
    bytes[19] = byte;
    Address::new(bytes)
}

fn parse_addr_unchecked(s: &str) -> Address {
    Address::from_str(s.trim_start_matches("0x")).expect("static address literal valid")
}

fn build_genesis_json(
    chain_id: u64,
    arbos_version: u64,
    chain_owner: Address,
    allow_debug_precompiles: bool,
    alloc: BTreeMap<String, Value>,
) -> Value {
    let mix_hash = arbos_mix_hash(arbos_version);

    json!({
        "config": {
            "chainId": chain_id,
            "homesteadBlock": 0,
            "daoForkSupport": true,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "muirGlacierBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "depositContractAddress": "0x0000000000000000000000000000000000000000",
            "clique": { "period": 0, "epoch": 0 },
            "arbitrum": {
                "EnableArbOS": true,
                "AllowDebugPrecompiles": allow_debug_precompiles,
                "DataAvailabilityCommittee": false,
                "InitialArbOSVersion": arbos_version,
                "InitialChainOwner": format!("{chain_owner:#x}"),
                "GenesisBlockNum": 0u64,
                "SkipGenesisInjection": true,
            }
        },
        "alloc": alloc,
        "nonce": "0x1",
        "timestamp": "0x0",
        "gasLimit": "0x4000000000000",
        "difficulty": "0x1",
        "mixHash": mix_hash,
        "coinbase": "0x0000000000000000000000000000000000000000",
        "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000",
        "baseFeePerGas": "0x5f5e100",
    })
}

fn arbos_mix_hash(arbos_version: u64) -> String {
    let mut bytes = [0u8; 32];
    bytes[23] = arbos_version as u8;
    format!("0x{}", hex::encode(bytes))
}
