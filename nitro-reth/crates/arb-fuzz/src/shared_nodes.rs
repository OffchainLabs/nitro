use std::sync::{Mutex, OnceLock};

use arb_test_harness::node::{arbreth::ArbrethProcess, nitro_local::NitroProcess};
use arb_test_harness::DualExec;

static NODES: OnceLock<Mutex<DualExec<NitroProcess, ArbrethProcess>>> = OnceLock::new();

/// Process-wide shared `DualExec`; first call spawns both nodes.
pub fn shared_dual_exec() -> &'static Mutex<DualExec<NitroProcess, ArbrethProcess>> {
    NODES.get_or_init(|| {
        unimplemented!("shared_dual_exec: node spawn not wired");
    })
}
