#![no_main]

use arb_fuzz::{
    arbitrary_impls::DiffTxScenario, corpus_helpers::dump_crash_as_fixture,
    shared_nodes::shared_dual_exec,
};
use libfuzzer_sys::fuzz_target;

fuzz_target!(|scenario: DiffTxScenario| {
    let scen = scenario.clone().into_scenario();
    if scen.steps.is_empty() {
        return;
    }
    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
    match nodes.run(&scen) {
        Ok(report) if !report.is_clean() => {
            let path = dump_crash_as_fixture(&scenario, &report);
            panic!("divergence (fixture: {path:?}): {report:#?}");
        }
        Ok(_) => {}
        Err(e) => {
            eprintln!("harness error: {e}");
        }
    }
});
