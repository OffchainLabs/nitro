//! Drives a real `arb-reth` subprocess via the `nitroexecution_digestMessage` RPC.

use std::{
    io::{BufRead, BufReader},
    path::{Path, PathBuf},
    process::{Child, Command, Stdio},
    sync::{
        atomic::{AtomicBool, Ordering},
        Arc,
    },
    thread,
    time::{Duration, Instant},
};

use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::Address;
use base64::{engine::general_purpose::STANDARD as B64, Engine as _};
use rand::{RngCore, SeedableRng};
use rand_chacha::ChaCha20Rng;
use serde::{Deserialize, Serialize};

use super::{BlockInput, RunnerConfig, Workload};
use crate::metrics::{
    clock::Stopwatch, memory::RssMonitor, rolling::build_windows, BlockMetric, HostInfo, RunResult,
    SummaryMetrics,
};

const L1_KIND_L2_MESSAGE: u8 = 3;
const L1_KIND_INIT_MSG: u8 = 11;
const L2_KIND_BATCH: u8 = 3;
const L2_KIND_SIGNED_TX: u8 = 4;
const SEQUENCER_SENDER: Address =
    alloy_primitives::address!("a4b000000000000000000073657175656e636572");

#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct SubprocessConfig {
    pub binary: PathBuf,
    pub genesis: PathBuf,
    pub data_dir: PathBuf,
    pub http_port: u16,
    pub authrpc_port: u16,
    /// Wait this long for the node's HTTP RPC to come up before failing.
    pub startup_timeout: Duration,
    /// Per-request timeout for digestMessage.
    pub request_timeout: Duration,
    /// Persist a JWT secret and emit its path; arb-reth requires authrpc auth.
    pub jwt_secret_path: PathBuf,
    /// Block-buffer flush interval passed to arb-reth.
    pub flush_interval: u64,
    /// Stream the node's stderr to our stderr.
    pub stream_node_logs: bool,
}

impl SubprocessConfig {
    /// Build a config rooted at `data_dir`. Caller is responsible for picking
    /// non-conflicting ports.
    pub fn new(binary: PathBuf, genesis: PathBuf, data_dir: PathBuf) -> Self {
        let jwt_secret_path = data_dir.join("jwt.hex");
        let stream = std::env::var("ARBRETH_BENCH_STREAM_LOGS").is_ok();
        Self {
            binary,
            genesis,
            data_dir,
            http_port: 38545,
            authrpc_port: 38551,
            startup_timeout: Duration::from_secs(60),
            request_timeout: Duration::from_secs(30),
            jwt_secret_path,
            flush_interval: 128,
            stream_node_logs: stream,
        }
    }
}

pub struct SubprocessRunner {
    config: RunnerConfig,
    sub: SubprocessConfig,
    rss: RssMonitor,
}

impl SubprocessRunner {
    pub fn new(config: RunnerConfig, sub: SubprocessConfig) -> Self {
        Self {
            config,
            sub,
            rss: RssMonitor::new(),
        }
    }

    pub fn run(&mut self, workload: Workload) -> eyre::Result<RunResult> {
        std::fs::create_dir_all(&self.sub.data_dir)?;
        write_jwt_secret(&self.sub.jwt_secret_path)?;

        // Inject the workload's funded accounts and deployed contracts into
        // a copy of the genesis file in data_dir; point arb-reth at that copy.
        let custom_genesis = self.sub.data_dir.join("genesis.json");
        build_custom_genesis(&self.sub.genesis, &custom_genesis, &workload)?;
        let mut sub = self.sub.clone();
        sub.genesis = custom_genesis;

        tracing::info!(
            data_dir = %sub.data_dir.display(),
            http_port = sub.http_port,
            "spawning arb-reth subprocess"
        );
        let mut node = NodeProcess::spawn(&sub)?;
        let url = format!("http://127.0.0.1:{}", self.sub.http_port);
        let client = reqwest::blocking::Client::builder()
            .timeout(self.sub.request_timeout)
            .build()?;
        wait_for_ready(&client, &url, self.sub.startup_timeout)?;

        let mut msg_idx: u64 = 0;
        send_init_message(&client, &url, workload.chain_id, &mut msg_idx)?;

        let mut blocks = Vec::with_capacity(workload.blocks.len());
        for block in &workload.blocks {
            let metric = self.execute_one_block(&client, &url, &mut msg_idx, block)?;
            blocks.push(metric);
        }

        node.shutdown();

        let windows = build_windows(&blocks, self.config.rolling_window_blocks);
        let summary = SummaryMetrics::from_blocks(&blocks, &windows);
        Ok(RunResult {
            manifest_name: workload.manifest_name,
            blocks,
            windows,
            summary,
            host: HostInfo::collect(),
        })
    }

    fn execute_one_block(
        &mut self,
        client: &reqwest::blocking::Client,
        url: &str,
        msg_idx: &mut u64,
        block: &BlockInput,
    ) -> eyre::Result<BlockMetric> {
        let l2_msg = encode_l2_batch(&block.txs);
        let body = serde_json::json!({
            "jsonrpc": "2.0",
            "id": *msg_idx,
            "method": "nitroexecution_digestMessage",
            "params": [
                *msg_idx,
                {
                    "message": {
                        "header": {
                            "kind": L1_KIND_L2_MESSAGE,
                            "sender": format!("{SEQUENCER_SENDER:#x}"),
                            "blockNumber": block.block_number,
                            "timestamp": block.timestamp,
                            "requestId": null,
                            "baseFeeL1": format!("{}", block.base_fee),
                        },
                        "l2Msg": B64.encode(&l2_msg),
                    },
                    "delayedMessagesRead": 1u64,
                },
                serde_json::Value::Null,
            ],
        });

        let sw = Stopwatch::start();
        let resp = client.post(url).json(&body).send()?.error_for_status()?;
        let json: serde_json::Value = resp.json()?;
        let (wall, cpu) = sw.elapsed_ns();

        if let Some(err) = json.get("error") {
            return Err(eyre::eyre!("digestMessage error: {err}"));
        }
        let rss = self.rss.current_rss();
        *msg_idx += 1;

        // Pull the actual gas + tx count from the produced block.
        let (gas_used, tx_count, success_count) =
            fetch_block_stats(client, url).unwrap_or((0, block.txs.len(), block.txs.len()));

        Ok(BlockMetric {
            block_number: block.block_number,
            wall_clock_ns: wall,
            cpu_ns: cpu,
            gas_used,
            tx_count,
            success_count,
            rss_bytes: rss,
        })
    }
}

fn fetch_block_stats(
    client: &reqwest::blocking::Client,
    url: &str,
) -> eyre::Result<(u64, usize, usize)> {
    let body = serde_json::json!({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "eth_getBlockByNumber",
        "params": ["latest", false],
    });
    let resp: serde_json::Value = client
        .post(url)
        .json(&body)
        .send()?
        .error_for_status()?
        .json()?;
    let result = resp
        .get("result")
        .ok_or_else(|| eyre::eyre!("rpc error: {:?}", resp.get("error")))?;
    let gas_str = result
        .get("gasUsed")
        .and_then(|v| v.as_str())
        .unwrap_or("0x0");
    let gas_used = u64::from_str_radix(gas_str.trim_start_matches("0x"), 16).unwrap_or(0);
    let tx_count = result
        .get("transactions")
        .and_then(|v| v.as_array())
        .map(|a| a.len())
        .unwrap_or(0);
    Ok((gas_used, tx_count, tx_count))
}

fn build_custom_genesis(base_genesis: &Path, dest: &Path, workload: &Workload) -> eyre::Result<()> {
    let bytes = std::fs::read(base_genesis)
        .map_err(|e| eyre::eyre!("read base genesis {}: {e}", base_genesis.display()))?;
    let mut json: serde_json::Value = serde_json::from_slice(&bytes)?;

    if let Some(config) = json.get_mut("config") {
        if let Some(obj) = config.as_object_mut() {
            obj.insert(
                "chainId".into(),
                serde_json::Value::Number(workload.chain_id.into()),
            );
        }
    }

    let alloc = json
        .get_mut("alloc")
        .and_then(|v| v.as_object_mut())
        .ok_or_else(|| eyre::eyre!("base genesis has no alloc object"))?;

    for (addr, balance) in &workload.funded_accounts {
        let key = format!("{:x}", addr);
        let mut entry = serde_json::Map::new();
        entry.insert(
            "balance".into(),
            serde_json::Value::String(format!("0x{:x}", balance)),
        );
        alloc.insert(key, serde_json::Value::Object(entry));
    }
    for c in &workload.deployed_contracts {
        let key = format!("{:x}", c.address);
        let mut entry = serde_json::Map::new();
        entry.insert(
            "balance".into(),
            serde_json::Value::String(format!("0x{:x}", c.balance)),
        );
        entry.insert(
            "code".into(),
            serde_json::Value::String(format!("0x{}", hex::encode(&c.runtime_code))),
        );
        alloc.insert(key, serde_json::Value::Object(entry));
    }

    if let Some(prewarm) = &workload.prewarm_alloc {
        let bal = format!("0x{:x}", prewarm.balance);
        let mut rng = ChaCha20Rng::seed_from_u64(prewarm.seed);
        let mut bytes = [0u8; 20];
        for _ in 0..prewarm.count {
            rng.fill_bytes(&mut bytes);
            let key = hex::encode(bytes);
            let mut entry = serde_json::Map::new();
            entry.insert("balance".into(), serde_json::Value::String(bal.clone()));
            alloc.insert(key, serde_json::Value::Object(entry));
        }
        tracing::info!(
            count = prewarm.count,
            "injected prewarm alloc into custom genesis"
        );
    }

    std::fs::write(dest, serde_json::to_vec(&json)?)?;
    Ok(())
}

fn encode_l2_batch(txs: &[arb_primitives::ArbTransactionSigned]) -> Vec<u8> {
    let mut out = Vec::new();
    out.push(L2_KIND_BATCH);
    for tx in txs {
        let mut tx_bytes = Vec::new();
        tx_bytes.push(L2_KIND_SIGNED_TX);
        let mut enc = Vec::new();
        tx.encode_2718(&mut enc);
        tx_bytes.extend_from_slice(&enc);
        let len = tx_bytes.len() as u64;
        out.extend_from_slice(&len.to_be_bytes());
        out.extend_from_slice(&tx_bytes);
    }
    out
}

fn write_jwt_secret(path: &Path) -> std::io::Result<()> {
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)?;
    }
    if path.exists() {
        return Ok(());
    }
    let secret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
    std::fs::write(path, secret)
}

fn send_init_message(
    client: &reqwest::blocking::Client,
    url: &str,
    chain_id: u64,
    msg_idx: &mut u64,
) -> eyre::Result<()> {
    let mut payload = vec![0u8; 32];
    let chain_id_bytes = chain_id.to_be_bytes();
    payload[24..].copy_from_slice(&chain_id_bytes);
    payload.push(1u8); // version
    payload.extend_from_slice(&[0u8; 32]); // initial_l1_base_fee = 0
    let body = serde_json::json!({
        "jsonrpc": "2.0",
        "id": *msg_idx,
        "method": "nitroexecution_digestMessage",
        "params": [
            *msg_idx,
            {
                "message": {
                    "header": {
                        "kind": L1_KIND_INIT_MSG,
                        "sender": format!("{:#x}", Address::ZERO),
                        "blockNumber": 0u64,
                        "timestamp": 1_700_000_000u64,
                        "requestId": null,
                        "baseFeeL1": "0",
                    },
                    "l2Msg": B64.encode(&payload),
                },
                "delayedMessagesRead": 0u64,
            },
            serde_json::Value::Null,
        ],
    });
    let _resp: serde_json::Value = client
        .post(url)
        .json(&body)
        .send()?
        .error_for_status()?
        .json()?;
    *msg_idx += 1;
    Ok(())
}

fn wait_for_ready(
    client: &reqwest::blocking::Client,
    url: &str,
    timeout: Duration,
) -> eyre::Result<()> {
    let start = Instant::now();
    let body = serde_json::json!({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "eth_blockNumber",
        "params": [],
    });
    while start.elapsed() < timeout {
        if let Ok(resp) = client.post(url).json(&body).send() {
            if resp.status().is_success() {
                if let Ok(json) = resp.json::<serde_json::Value>() {
                    if json.get("result").is_some() {
                        return Ok(());
                    }
                }
            }
        }
        thread::sleep(Duration::from_millis(250));
    }
    Err(eyre::eyre!(
        "node did not become ready at {url} within {:?}",
        timeout
    ))
}

struct NodeProcess {
    child: Child,
    log_thread: Option<thread::JoinHandle<()>>,
    stop: Arc<AtomicBool>,
}

impl NodeProcess {
    fn spawn(cfg: &SubprocessConfig) -> eyre::Result<Self> {
        let mut cmd = Command::new(&cfg.binary);
        cmd.args([
            "node",
            "--chain",
            cfg.genesis.to_str().expect("genesis path utf8"),
            "--datadir",
            cfg.data_dir.to_str().expect("datadir utf8"),
            "--http",
            "--http.addr",
            "127.0.0.1",
            "--http.port",
            &cfg.http_port.to_string(),
            "--http.api",
            "eth,web3,net",
            "--authrpc.addr",
            "127.0.0.1",
            "--authrpc.port",
            &cfg.authrpc_port.to_string(),
            "--authrpc.jwtsecret",
            cfg.jwt_secret_path.to_str().expect("jwt path utf8"),
            "--disable-discovery",
            "--db.exclusive=true",
            "--db.sync-mode=safe-no-sync",
        ]);
        let log_filter = std::env::var("ARBRETH_BENCH_NODE_LOG").unwrap_or_else(|_| "warn".into());
        cmd.arg(format!("--log.stdout.filter={log_filter}"));
        cmd.env("ARB_FLUSH_INTERVAL", cfg.flush_interval.to_string());
        if std::env::var("ARB_INITIAL_ARBOS_VERSION").is_err() {
            cmd.env("ARB_INITIAL_ARBOS_VERSION", "60");
        }
        cmd.stdout(Stdio::piped());
        cmd.stderr(Stdio::piped());
        let mut child = cmd
            .spawn()
            .map_err(|e| eyre::eyre!("spawn arb-reth ({}): {e}", cfg.binary.display()))?;
        let stop = Arc::new(AtomicBool::new(false));
        let echo = cfg.stream_node_logs;
        let stdout = child.stdout.take().expect("piped stdout");
        let stderr = child.stderr.take().expect("piped stderr");
        let stop_o = stop.clone();
        let stop_e = stop.clone();
        let log_thread = Some(thread::spawn(move || {
            let stderr_jh = thread::spawn(move || {
                let reader = BufReader::new(stderr);
                for line in reader.lines() {
                    if stop_e.load(Ordering::Relaxed) {
                        return;
                    }
                    if let Ok(l) = line {
                        if echo {
                            eprintln!("[arb-reth] {l}");
                        }
                    }
                }
            });
            let reader = BufReader::new(stdout);
            for line in reader.lines() {
                if stop_o.load(Ordering::Relaxed) {
                    break;
                }
                if let Ok(l) = line {
                    if echo {
                        eprintln!("[arb-reth] {l}");
                    }
                }
            }
            let _ = stderr_jh.join();
        }));
        Ok(Self {
            child,
            log_thread,
            stop,
        })
    }

    fn shutdown(&mut self) {
        self.stop.store(true, Ordering::Relaxed);
        let _ = self.child.kill();
        let _ = self.child.wait();
        if let Some(j) = self.log_thread.take() {
            let _ = j.join();
        }
    }
}

impl Drop for NodeProcess {
    fn drop(&mut self) {
        self.shutdown();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Bytes, TxKind, U256};
    use arb_executor_tests::helpers::{alice_key, sign_legacy, ONE_GWEI};

    #[test]
    fn encode_batch_single_tx_roundtrips_to_kind_batch() {
        let tx = sign_legacy(
            421614,
            0,
            ONE_GWEI,
            21_000,
            TxKind::Call(Address::repeat_byte(0x11)),
            U256::from(1u64),
            Bytes::new(),
            alice_key(),
        );
        let bytes = encode_l2_batch(&[tx]);
        assert_eq!(bytes[0], L2_KIND_BATCH);
        // First sub-msg length is bytes 1..9; should be > 0.
        let len = u64::from_be_bytes(bytes[1..9].try_into().unwrap());
        assert!(len > 0);
        // Sub-msg first byte is the L2 kind.
        assert_eq!(bytes[9], L2_KIND_SIGNED_TX);
    }

    #[test]
    fn encode_batch_empty_yields_only_kind_byte() {
        let bytes = encode_l2_batch(&[]);
        assert_eq!(bytes, vec![L2_KIND_BATCH]);
    }

    #[test]
    fn jwt_secret_written() {
        let dir = tempfile::tempdir().unwrap();
        let p = dir.path().join("jwt.hex");
        write_jwt_secret(&p).unwrap();
        assert!(p.exists());
        let s = std::fs::read_to_string(&p).unwrap();
        assert_eq!(s.len(), 64);
    }
}
