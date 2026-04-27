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
