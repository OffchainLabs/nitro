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
    let subs = ["hostio", "subcall", "cache", "contract_limit", "regression"];
    let mut panics: Vec<String> = Vec::new();
    for sub in subs {
        let dir = stylus_root.join(sub);
        let r = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| run_execution_dir(&dir)));
        if let Err(payload) = r {
            let msg = payload
                .downcast_ref::<String>()
                .cloned()
                .or_else(|| payload.downcast_ref::<&str>().map(|s| s.to_string()))
                .unwrap_or_else(|| format!("<panic in stylus/{sub} (non-string payload)>"));
            panics.push(msg);
        }
    }
    if !panics.is_empty() {
        panic!("{}/{} stylus sub-dirs failed:\n{}", panics.len(), subs.len(), panics.join("\n"));
    }
}

#[test]
fn retryables_exec() {
    let retry_root = fixtures_root().join("retryables");
    if !retry_root.exists() {
        return;
    }
    let mut had_exec = false;
    let mut count = 0;
    let mut failures: Vec<String> = Vec::new();
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
        count += 1;
        if let Err(e) = arb_spec_tests::runner::run_execution_fixture(&path, None) {
            failures.push(format!("{}: {e}", path.display()));
        }
    }
    assert!(had_exec, "no execution-shaped fixtures found in {}", retry_root.display());
    if !failures.is_empty() {
        panic!(
            "{}/{} execution fixtures failed:\n  {}",
            failures.len(),
            count,
            failures.join("\n  ")
        );
    }
}
