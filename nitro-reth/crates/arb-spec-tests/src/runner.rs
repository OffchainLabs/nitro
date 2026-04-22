use std::{
    io::Write,
    path::{Path, PathBuf},
    process::{Child, Command, Stdio},
    sync::atomic::{AtomicU16, Ordering},
    time::{Duration, Instant},
};

use walkdir::WalkDir;

use crate::{
    case::{SpecCase, SpecError},
    execution::ExecutionFixture,
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
    let fixture = ExecutionFixture::load(path)?;
    let label = path.display().to_string();

    let result = if fixture.genesis.is_some() {
        let binary = std::env::var(BINARY_ENV).map_err(|_| {
            SpecError::Action(format!(
                "{label}: fixture has inline genesis but {BINARY_ENV} is unset"
            ))
        })?;
        let node = SpawnedNode::start(&fixture, Path::new(&binary))?;
        fixture.run(&node.rpc_url)
    } else {
        let url = rpc_url.ok_or_else(|| {
            SpecError::Action(format!(
                "{label}: fixture has no inline genesis and {RPC_URL_ENV} is unset"
            ))
        })?;
        fixture.run(url)
    };

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
    let mut failures = Vec::new();
    let mut count = 0;
    for entry in WalkDir::new(dir).into_iter().filter_map(Result::ok) {
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
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

static NEXT_PORT: AtomicU16 = AtomicU16::new(28545);

struct SpawnedNode {
    pub rpc_url: String,
    child: Option<Child>,
    workdir: PathBuf,
}

impl SpawnedNode {
    fn start(fixture: &ExecutionFixture, binary: &Path) -> Result<Self, SpecError> {
        let port_pair = NEXT_PORT.fetch_add(2, Ordering::SeqCst);
        let http_port = port_pair;
        let auth_port = port_pair + 1;

        let workdir =
            std::env::temp_dir().join(format!("arb-spec-{}-{}", fixture.name, std::process::id(),));
        eprintln!("[arb-spec] spawning arbreth in {}", workdir.display());
        if workdir.exists() {
            let _ = std::fs::remove_dir_all(&workdir);
        }
        std::fs::create_dir_all(&workdir)
            .map_err(|e| SpecError::Action(format!("mkdir {}: {e}", workdir.display())))?;

        let chain_path = workdir.join("chain.json");
        let genesis = fixture
            .genesis
            .as_ref()
            .ok_or_else(|| SpecError::Action("internal: spawn called without genesis".into()))?;
        let chain_json = serde_json::to_vec_pretty(genesis)
            .map_err(|e| SpecError::Action(format!("serialize genesis: {e}")))?;
        std::fs::File::create(&chain_path)
            .and_then(|mut f| f.write_all(&chain_json))
            .map_err(|e| SpecError::Action(format!("write chain.json: {e}")))?;

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
        let deadline = Instant::now() + Duration::from_secs(30);
        loop {
            if Instant::now() > deadline {
                return Err(SpecError::Action(format!(
                    "arbreth at {rpc_url} did not respond within 30s"
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
