use std::{
    path::PathBuf,
    sync::{
        atomic::{AtomicU64, Ordering},
        Mutex, OnceLock,
    },
};

use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    mock_l1::MockL1,
    node::{arbreth::ArbrethProcess, nitro_docker::NitroDocker, NodeStartCtx},
};

/// L2 chain id used for all fuzz scenarios.
pub const FUZZ_L2_CHAIN_ID: u64 = 412_346;
/// L1 chain id served by the embedded mock.
pub const FUZZ_L1_CHAIN_ID: u64 = 11_155_111;
/// ArbOS version baked into the shared genesis. Default v60; override per
/// process via `ARB_FUZZ_ARBOS_VERSION=32|50|60`.
pub const FUZZ_ARBOS_VERSION: u64 = 60;

pub fn fuzz_arbos_version() -> u64 {
    std::env::var("ARB_FUZZ_ARBOS_VERSION")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(FUZZ_ARBOS_VERSION)
}

/// Path to the captured Nitro genesis for `(chain_id, arbos_version)`. The
/// fuzz harness loads this so both nodes start from byte-identical state and
/// genesis state_root matches without any post-hoc filter.
pub fn captured_genesis_path(chain_id: u64, arbos_version: u64) -> PathBuf {
    let manifest = env!("CARGO_MANIFEST_DIR");
    PathBuf::from(manifest)
        .parent()
        .map(std::path::Path::to_path_buf)
        .unwrap_or_else(|| PathBuf::from("crates"))
        .join("arb-spec-tests")
        .join("fixtures")
        .join("_genesis_cache")
        .join(format!("chain{chain_id}_v{arbos_version}.json"))
}

fn load_captured_or_build(chain_id: u64, arbos_version: u64) -> serde_json::Value {
    let path = captured_genesis_path(chain_id, arbos_version);
    if let Ok(bytes) = std::fs::read(&path) {
        if let Ok(value) = serde_json::from_slice::<serde_json::Value>(&bytes) {
            return value;
        }
    }
    GenesisBuilder::new(chain_id, arbos_version)
        .build()
        .expect("genesis build")
}

static NODES: OnceLock<Mutex<DualExec<NitroDocker, ArbrethProcess>>> = OnceLock::new();

/// Process-wide monotonic L1 message index. Every scenario builder must
/// allocate idx values via [`next_msg_idx`] so the shared dual-exec inbox
/// stays in lockstep across fuzz iterations. Resetting per-iteration (the
/// historical pattern in `DiffTxScenario`, `ScenarioMix`, etc.) silently
/// wedges Nitro's `digestMessage` after iter 0.
static GLOBAL_MSG_IDX: AtomicU64 = AtomicU64::new(1);

/// Allocate the next inbox idx and return it.
pub fn next_msg_idx() -> u64 {
    GLOBAL_MSG_IDX.fetch_add(1, Ordering::SeqCst)
}

/// Peek the next idx without consuming it.
pub fn peek_msg_idx() -> u64 {
    GLOBAL_MSG_IDX.load(Ordering::SeqCst)
}

/// Reset the counter for tests that need a clean slate (e.g. ones that
/// spawn their own non-shared dual-exec).
pub fn reset_msg_idx() {
    GLOBAL_MSG_IDX.store(1, Ordering::SeqCst);
}

/// Construct a `NodeStartCtx` pointing at the supplied mock L1.
pub fn default_ctx(mock_rpc: String) -> NodeStartCtx {
    let genesis = load_captured_or_build(FUZZ_L2_CHAIN_ID, fuzz_arbos_version());
    NodeStartCtx {
        binary: None,
        l2_chain_id: FUZZ_L2_CHAIN_ID,
        l1_chain_id: FUZZ_L1_CHAIN_ID,
        mock_l1_rpc: mock_rpc,
        genesis,
        jwt_hex: String::new(),
        workdir: std::path::PathBuf::new(),
        http_port: 0,
        authrpc_port: 0,
    }
}

/// Process-wide shared `DualExec`. The first call spawns a mock L1, a Docker
/// Nitro reference container, and a local arbreth process. State accumulates
/// across fuzz iterations — that broadens coverage but means crash repro may
/// require restarting the fuzzer.
pub fn shared_dual_exec() -> &'static Mutex<DualExec<NitroDocker, ArbrethProcess>> {
    NODES.get_or_init(|| {
        let mock = MockL1::start(FUZZ_L1_CHAIN_ID).expect("mock l1 start");
        let ctx = default_ctx(mock.rpc_url());
        let nitro = NitroDocker::start(&ctx).expect("nitro docker start");
        let arbreth = ArbrethProcess::start(&ctx).expect("arbreth start");
        // Keep the mock alive for the lifetime of the test process.
        std::mem::forget(mock);
        Mutex::new(DualExec::new(nitro, arbreth))
    })
}
