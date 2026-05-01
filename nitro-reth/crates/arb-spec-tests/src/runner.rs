use std::{
    net::TcpListener,
    path::{Path, PathBuf},
    process::{Child, Command, Stdio},
    time::{Duration, Instant},
};

use walkdir::WalkDir;

use crate::{
    case::{SpecCase, SpecError},
    execution::ExecutionFixture,
    mode::FixtureMode,
};

pub const RPC_URL_ENV: &str = "ARB_SPEC_RPC_URL";
pub const BINARY_ENV: &str = "ARB_SPEC_BINARY";

pub fn run_fixture(path: &Path) -> Result<(), SpecError> {
    let case = SpecCase::load(path)?;
    case.run().map_err(|e| match e {
        SpecError::Assertion(msg) => SpecError::Assertion(format!("{}: {msg}", path.display())),
        other => other,
    })
}

pub fn run_execution_fixture(path: &Path, rpc_url: Option<&str>) -> Result<(), SpecError> {
    let mut fixture = ExecutionFixture::load(path)?;
    let label = path.display().to_string();
    let mode = FixtureMode::from_env();

    let result = if matches!(mode, FixtureMode::Record) {
        fixture.record_against_nitro()
    } else if fixture.genesis.is_some() {
        let binary = std::env::var(BINARY_ENV).map_err(|_| {
            SpecError::Action(format!(
                "{label}: fixture has inline genesis but {BINARY_ENV} is unset"
            ))
        })?;
        let node = SpawnedNode::start(&fixture, Path::new(&binary))?;
        fixture.run_with_mode(mode, &node.rpc_url)
    } else {
        let url = rpc_url.ok_or_else(|| {
            SpecError::Action(format!(
                "{label}: fixture has no inline genesis and {RPC_URL_ENV} is unset"
            ))
        })?;
        fixture.run_with_mode(mode, url)
    };

    if matches!(mode, FixtureMode::Record) && result.is_ok() {
        let body = serde_json::to_vec_pretty(&fixture)
            .map_err(|e| SpecError::Action(format!("{label}: encode: {e}")))?;
        std::fs::write(path, body)
            .map_err(|e| SpecError::Action(format!("{label}: write: {e}")))?;
    }

    result.map_err(|e| match e {
        SpecError::Assertion(msg) => SpecError::Assertion(format!("{label}: {msg}")),
        other => other,
    })
}

pub fn run_execution_dir(dir: &Path) {
    let rpc_url = std::env::var(RPC_URL_ENV).ok();
    let has_binary = std::env::var(BINARY_ENV).is_ok();
    if rpc_url.is_none() && !has_binary {
        // The dedicated spec-tests workflow opts in to running execution
        // fixtures by setting ARB_SPEC_REQUIRE_BINARY=1; if it's set and
        // we got here the workflow is misconfigured, so panic. The generic
        // workspace test job and local runs without a release binary fall
        // through to the skip notice.
        if std::env::var("ARB_SPEC_REQUIRE_BINARY").is_ok() {
            panic!(
                "execution fixtures under {} need {RPC_URL_ENV} or {BINARY_ENV} set",
                dir.display()
            );
        }
        eprintln!(
            "skipping execution fixtures under {}: set {RPC_URL_ENV} (static node) and/or {BINARY_ENV} (per-fixture genesis)",
            dir.display()
        );
        return;
    }
    assert!(dir.exists(), "fixture dir missing: {}", dir.display());
    let filter = std::env::var("ARB_SPEC_FILTER").ok();
    let include_pending = std::env::var("ARB_SPEC_INCLUDE_PENDING").is_ok();
    let mut failures = Vec::new();
    let mut count = 0;
    for entry in WalkDir::new(dir).into_iter().filter_map(Result::ok) {
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        // `pending_*.json` files reproduce known-unsolved divergences; they
        // would fail the suite if auto-run, so opt-in only.
        let stem = path
            .file_stem()
            .and_then(|s| s.to_str())
            .unwrap_or_default();
        if stem.starts_with("pending_") && !include_pending {
            continue;
        }
        if let Some(f) = &filter {
            if !path.to_string_lossy().contains(f) {
                continue;
            }
        }
        count += 1;
        if let Err(e) = run_execution_fixture(path, rpc_url.as_deref()) {
            failures.push(format!("{}: {e}", path.display()));
        }
    }
    assert!(count > 0, "no fixtures found under {}", dir.display());
    if !failures.is_empty() {
        panic!(
            "{}/{} execution fixtures failed:\n  {}",
            failures.len(),
            count,
            failures.join("\n  ")
        );
    }
}

pub fn run_dir(dir: &Path) {
    assert!(dir.exists(), "fixture dir missing: {}", dir.display());
    let mut failures = Vec::new();
    let mut count = 0;
    for entry in WalkDir::new(dir).into_iter().filter_map(Result::ok) {
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        count += 1;
        if let Err(e) = run_fixture(path) {
            failures.push(format!("{}: {e}", path.display()));
        }
    }
    assert!(count > 0, "no fixtures found under {}", dir.display());
    if !failures.is_empty() {
        panic!(
            "{}/{} fixtures failed:\n  {}",
            failures.len(),
            count,
            failures.join("\n  ")
        );
    }
}

pub fn fixtures_root() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR")).join("fixtures")
}

// ─── Per-fixture arbreth process ────────────────────────────────────

/// Bind to ephemeral port 0, drop the listener, return the port the OS
/// picked. Used so concurrent test processes (nextest runs every #[test]
/// in its own process) can't collide on a fixed port range.
fn pick_free_port() -> Result<u16, SpecError> {
    let listener = TcpListener::bind(("127.0.0.1", 0))
        .map_err(|e| SpecError::Action(format!("bind 127.0.0.1:0: {e}")))?;
    let port = listener
        .local_addr()
        .map_err(|e| SpecError::Action(format!("local_addr: {e}")))?
        .port();
    drop(listener);
    Ok(port)
}

struct SpawnedNode {
    pub rpc_url: String,
    child: Option<Child>,
    workdir: PathBuf,
}

impl SpawnedNode {
    fn start(fixture: &ExecutionFixture, binary: &Path) -> Result<Self, SpecError> {
        let http_port = pick_free_port()?;
        let auth_port = pick_free_port()?;

        let workdir =
            std::env::temp_dir().join(format!("arb-spec-{}-{}", fixture.name, std::process::id(),));
        eprintln!("[arb-spec] spawning arbreth in {}", workdir.display());
        if workdir.exists() {
            let _ = std::fs::remove_dir_all(&workdir);
        }
        std::fs::create_dir_all(&workdir)
            .map_err(|e| SpecError::Action(format!("mkdir {}: {e}", workdir.display())))?;

        let genesis = fixture
            .genesis
            .as_ref()
            .ok_or_else(|| SpecError::Action("internal: spawn called without genesis".into()))?;
        let cache_path = ensure_genesis_cache(genesis)?;
        let chain_path = layer_fixture_alloc(&cache_path, genesis, &workdir)?;

        let jwt_path = workdir.join("jwt.hex");
        std::fs::write(&jwt_path, hex::encode([0u8; 32]))
            .map_err(|e| SpecError::Action(format!("write jwt: {e}")))?;

        let log_path = workdir.join("node.log");
        let log_file = std::fs::File::create(&log_path)
            .map_err(|e| SpecError::Action(format!("create log file: {e}")))?;
        let log_err = log_file
            .try_clone()
            .map_err(|e| SpecError::Action(format!("clone log fd: {e}")))?;
        let child = Command::new(binary)
            .env(
                "RUST_LOG",
                std::env::var("ARB_SPEC_RUST_LOG")
                    .unwrap_or_else(|_| "info,block_producer=debug".to_string()),
            )
            .arg("node")
            .arg(format!("--chain={}", chain_path.display()))
            .arg(format!("--datadir={}", workdir.join("db").display()))
            .arg("--http")
            .arg("--http.addr=127.0.0.1")
            .arg(format!("--http.port={http_port}"))
            .arg("--http.api=eth,web3,net,debug")
            .arg("--authrpc.addr=127.0.0.1")
            .arg(format!("--authrpc.port={auth_port}"))
            .arg(format!("--authrpc.jwtsecret={}", jwt_path.display()))
            .arg("--disable-discovery")
            .arg("--db.exclusive=true")
            .stdout(Stdio::from(log_file))
            .stderr(Stdio::from(log_err))
            .spawn()
            .map_err(|e| SpecError::Action(format!("spawn arbreth: {e}")))?;

        let rpc_url = format!("http://127.0.0.1:{http_port}");
        let timeout_secs: u64 = std::env::var("ARB_SPEC_STARTUP_TIMEOUT")
            .ok()
            .and_then(|s| s.parse().ok())
            .unwrap_or(90);
        let deadline = Instant::now() + Duration::from_secs(timeout_secs);
        loop {
            if Instant::now() > deadline {
                return Err(SpecError::Action(format!(
                    "arbreth at {rpc_url} did not respond within {timeout_secs}s"
                )));
            }
            let probe = ureq::post(&rpc_url)
                .set("Content-Type", "application/json")
                .send_string(r#"{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}"#);
            if probe.is_ok() {
                break;
            }
            std::thread::sleep(Duration::from_millis(200));
        }

        Ok(SpawnedNode {
            rpc_url,
            child: Some(child),
            workdir,
        })
    }
}

impl Drop for SpawnedNode {
    fn drop(&mut self) {
        if let Some(mut child) = self.child.take() {
            let _ = child.kill();
            let _ = child.wait();
        }
        // Keep workdir for debugging if ARB_SPEC_KEEP_WORKDIR is set.
        if std::env::var("ARB_SPEC_KEEP_WORKDIR").is_err() {
            let _ = std::fs::remove_dir_all(&self.workdir);
        } else {
            eprintln!("kept arbreth workdir: {}", self.workdir.display());
        }
    }
}

/// Ensure a `(chainId, arbosVersion)`-keyed genesis file exists under
/// `fixtures/_genesis_cache/` and return its path. On miss, run
/// `arb-test genesis-capture` to produce it.
fn ensure_genesis_cache(genesis: &serde_json::Value) -> Result<PathBuf, SpecError> {
    let chain_id = genesis
        .pointer("/config/chainId")
        .and_then(serde_json::Value::as_u64)
        .ok_or_else(|| SpecError::Action("genesis missing config.chainId".into()))?;
    let arbos_version = genesis
        .pointer("/config/arbitrum/InitialArbOSVersion")
        .and_then(serde_json::Value::as_u64)
        .ok_or_else(|| {
            SpecError::Action("genesis missing config.arbitrum.InitialArbOSVersion".into())
        })?;

    let cache_dir = fixtures_root().join("_genesis_cache");
    let cache_path = cache_dir.join(format!("chain{chain_id}_v{arbos_version}.json"));

    if !cache_path.exists() {
        std::fs::create_dir_all(&cache_dir)
            .map_err(|e| SpecError::Action(format!("mkdir {}: {e}", cache_dir.display())))?;
        eprintln!(
            "[arb-spec] genesis cache miss for chain={chain_id} arbos=v{arbos_version}, capturing from reference node..."
        );
        let cache_str = cache_path
            .to_str()
            .ok_or_else(|| SpecError::Action("genesis cache path not UTF-8".into()))?;
        let status = Command::new("cargo")
            .args([
                "run",
                "--release",
                "-q",
                "-p",
                "arb-test",
                "--",
                "genesis-capture",
                "--chain-id",
                &chain_id.to_string(),
                "--arbos-version",
                &arbos_version.to_string(),
                "--out",
                cache_str,
            ])
            .status()
            .map_err(|e| SpecError::Action(format!("invoke arb-test genesis-capture: {e}")))?;
        if !status.success() {
            return Err(SpecError::Action(format!(
                "arb-test genesis-capture failed for chain={chain_id} arbos=v{arbos_version}"
            )));
        }
        if !cache_path.exists() {
            return Err(SpecError::Action(format!(
                "arb-test genesis-capture exited 0 but did not produce {}",
                cache_path.display()
            )));
        }
    }

    Ok(cache_path)
}

/// Layer fixture-specific overrides onto the shared cache JSON: per-address
/// alloc entries plus any `config.arbitrum.*` fields the fixture sets
/// (notably `InitialChainOwner`, which determines the chain owner stored
/// at block 0). When neither is present the cache path is returned
/// unchanged. The cache file itself is never mutated.
fn layer_fixture_alloc(
    cache_path: &Path,
    fixture_genesis: &serde_json::Value,
    workdir: &Path,
) -> Result<PathBuf, SpecError> {
    let fixture_alloc = fixture_genesis
        .get("alloc")
        .and_then(|v| v.as_object())
        .filter(|m| !m.is_empty());
    let fixture_arbitrum = fixture_genesis
        .pointer("/config/arbitrum")
        .and_then(|v| v.as_object())
        .filter(|m| !m.is_empty());
    if fixture_alloc.is_none() && fixture_arbitrum.is_none() {
        return Ok(cache_path.to_path_buf());
    }
    let cache_bytes = std::fs::read(cache_path)
        .map_err(|e| SpecError::Action(format!("read cache {}: {e}", cache_path.display())))?;
    let mut cached: serde_json::Value = serde_json::from_slice(&cache_bytes)
        .map_err(|e| SpecError::Action(format!("parse cache {}: {e}", cache_path.display())))?;
    let cached_obj = cached.as_object_mut().ok_or_else(|| {
        SpecError::Action(format!("cache {} not a JSON object", cache_path.display()))
    })?;
    if let Some(alloc) = fixture_alloc {
        let merged_alloc = merge_alloc(
            cached_obj.get("alloc").unwrap_or(&serde_json::Value::Null),
            alloc,
        );
        cached_obj.insert("alloc".to_string(), serde_json::Value::Object(merged_alloc));
    }
    if let Some(arbitrum) = fixture_arbitrum {
        merge_arbitrum_config(cached_obj, arbitrum);
    }
    let out_path = workdir.join("chain.json");
    let merged_bytes = serde_json::to_vec_pretty(cached_obj)
        .map_err(|e| SpecError::Action(format!("encode merged genesis: {e}")))?;
    std::fs::write(&out_path, merged_bytes)
        .map_err(|e| SpecError::Action(format!("write {}: {e}", out_path.display())))?;
    Ok(out_path)
}

/// Overlay fixture-supplied `config.arbitrum.*` keys onto the cache's
/// existing arbitrum block. Fixture values win on collision.
fn merge_arbitrum_config(
    cache_obj: &mut serde_json::Map<String, serde_json::Value>,
    fixture_arbitrum: &serde_json::Map<String, serde_json::Value>,
) {
    let config = cache_obj
        .entry("config")
        .or_insert_with(|| serde_json::Value::Object(serde_json::Map::new()));
    let Some(config_obj) = config.as_object_mut() else {
        return;
    };
    let arbitrum = config_obj
        .entry("arbitrum")
        .or_insert_with(|| serde_json::Value::Object(serde_json::Map::new()));
    let Some(arbitrum_obj) = arbitrum.as_object_mut() else {
        return;
    };
    for (k, v) in fixture_arbitrum {
        arbitrum_obj.insert(k.clone(), v.clone());
    }
}

/// Replace cache entries with fixture entries on a per-address basis.
/// Match keys case-insensitively (chainspec addresses are
/// canonicalized lowercase by the cache producer) so a differently-cased
/// fixture spelling can't sneak in as a separate entry.
fn merge_alloc(
    cached: &serde_json::Value,
    fixture_alloc: &serde_json::Map<String, serde_json::Value>,
) -> serde_json::Map<String, serde_json::Value> {
    let mut out = serde_json::Map::new();
    let mut canonical_for: std::collections::HashMap<String, String> = Default::default();
    if let Some(existing) = cached.get("alloc").and_then(|v| v.as_object()) {
        for (k, v) in existing {
            canonical_for.insert(k.to_lowercase(), k.clone());
            out.insert(k.clone(), v.clone());
        }
    }
    for (k, v) in fixture_alloc {
        let lk = k.to_lowercase();
        if let Some(existing) = canonical_for.get(&lk) {
            out.insert(existing.clone(), v.clone());
        } else {
            out.insert(k.clone(), v.clone());
            canonical_for.insert(lk, k.clone());
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn merge_keeps_system_addresses_and_adds_fixture_entry() {
        let cache = json!({
            "config": { "chainId": 412346 },
            "alloc": {
                "0xa4b05fffffffffffffffffffffffffffffffffff": { "balance": "1", "code": "0x" },
                "0x0000000000000000000000000000000000000064": { "balance": "0", "code": "0x60" }
            }
        });
        let fixture = json!({
            "config": { "chainId": 412346 },
            "alloc": {
                "0x26E554a8acF9003b83495c7f45F06edCB803d4e3": { "balance": "999" }
            }
        });
        let fixture_alloc = fixture
            .get("alloc")
            .and_then(|v| v.as_object())
            .expect("alloc object");
        let merged = merge_alloc(&cache, fixture_alloc);

        let arbos_addr = merged
            .get("0xa4b05fffffffffffffffffffffffffffffffffff")
            .expect("arbos system address present");
        assert_eq!(
            arbos_addr.get("balance").and_then(|v| v.as_str()),
            Some("1")
        );

        let user = merged
            .get("0x26E554a8acF9003b83495c7f45F06edCB803d4e3")
            .expect("fixture alloc entry present");
        assert_eq!(user.get("balance").and_then(|v| v.as_str()), Some("999"));

        let precompile = merged
            .get("0x0000000000000000000000000000000000000064")
            .expect("precompile slot retained");
        assert_eq!(
            precompile.get("code").and_then(|v| v.as_str()),
            Some("0x60")
        );
    }

    #[test]
    fn merge_overwrites_when_fixture_targets_existing_address() {
        let cache = json!({
            "alloc": {
                "0xa4b05fffffffffffffffffffffffffffffffffff": { "balance": "1" }
            }
        });
        let fixture = json!({
            "alloc": {
                "0xA4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF": { "balance": "42" }
            }
        });
        let fixture_alloc = fixture.get("alloc").and_then(|v| v.as_object()).unwrap();
        let merged = merge_alloc(&cache, fixture_alloc);
        // We expect a single canonical entry under the cache's spelling.
        assert_eq!(merged.len(), 1);
        let entry = merged
            .get("0xa4b05fffffffffffffffffffffffffffffffffff")
            .expect("canonical lower entry");
        assert_eq!(entry.get("balance").and_then(|v| v.as_str()), Some("42"));
    }

    #[test]
    fn layer_skips_when_fixture_alloc_empty() {
        let tmp = std::env::temp_dir().join(format!("arb-spec-merge-test-{}", std::process::id()));
        std::fs::create_dir_all(&tmp).unwrap();
        let cache_path = tmp.join("cache.json");
        std::fs::write(&cache_path, r#"{"alloc":{"0xaa":{"balance":"1"}}}"#).unwrap();
        let workdir = tmp.join("work");
        std::fs::create_dir_all(&workdir).unwrap();
        let no_alloc = serde_json::json!({ "alloc": {} });
        let returned = layer_fixture_alloc(&cache_path, &no_alloc, &workdir).unwrap();
        assert_eq!(returned, cache_path);
        let _ = std::fs::remove_dir_all(&tmp);
    }
}
