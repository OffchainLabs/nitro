use std::sync::{Mutex, OnceLock};

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

static NODES: OnceLock<Mutex<DualExec<NitroDocker, ArbrethProcess>>> = OnceLock::new();

/// Construct a `NodeStartCtx` pointing at the supplied mock L1.
pub fn default_ctx(mock_rpc: String) -> NodeStartCtx {
    let genesis = GenesisBuilder::new(FUZZ_L2_CHAIN_ID, fuzz_arbos_version())
        .build()
        .expect("genesis build");
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
