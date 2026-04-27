#![no_main]

use arb_fuzz::arbitrary_impls::PrecompileScenario;
use arb_fuzz::corpus_helpers::dump_crash_as_fixture;
use arb_fuzz::shared_nodes::shared_dual_exec;
use libfuzzer_sys::fuzz_target;

fuzz_target!(|scenario: PrecompileScenario| {
    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
    let scen = scenario.clone().into_scenario();
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
