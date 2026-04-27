use arb_spec_tests::{run_dir, run_execution_dir, runner::fixtures_root};

macro_rules! spec_dir {
    ($name:ident, $dir:literal) => {
        #[test]
        fn $name() {
            run_dir(&fixtures_root().join($dir));
        }
    };
}

spec_dir!(pricing, "pricing");
spec_dir!(state_transitions, "state_transitions");
spec_dir!(retryables, "retryables");
spec_dir!(l1_pricing_dynamics, "l1_pricing_dynamics");
spec_dir!(address_handling, "address_handling");
spec_dir!(merkle, "merkle");
spec_dir!(version_transitions, "version_transitions");

#[test]
fn execution() {
    run_execution_dir(&fixtures_root().join("execution"));
}

#[test]
fn arbos_gates() {
    run_execution_dir(&fixtures_root().join("arbos"));
}

#[test]
fn stylus() {
    let stylus_root = fixtures_root().join("stylus");
    for sub in ["hostio", "subcall", "cache", "contract_limit"] {
        run_execution_dir(&stylus_root.join(sub));
    }
}

#[test]
fn retryables_exec() {
    let retry_root = fixtures_root().join("retryables");
    if !retry_root.exists() {
        return;
    }
    let mut had_exec = false;
    for entry in std::fs::read_dir(&retry_root).expect("read retryables dir") {
        let path = entry.expect("entry").path();
        if path.extension().and_then(|s| s.to_str()) != Some("json") {
            continue;
        }
        let body = std::fs::read_to_string(&path).expect("read fixture");
        if !body.contains("\"messages\"") {
            continue;
        }
        had_exec = true;
        if let Err(e) = arb_spec_tests::runner::run_execution_fixture(&path, None) {
            panic!("{}: {e}", path.display());
        }
    }
    assert!(had_exec, "no execution-shaped fixtures found in {}", retry_root.display());
}
