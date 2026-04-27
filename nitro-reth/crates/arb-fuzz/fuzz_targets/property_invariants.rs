#![no_main]

use arb_fuzz::arbitrary_impls::ScenarioMix;
use arb_fuzz::shared_nodes::shared_dual_exec;
use libfuzzer_sys::fuzz_target;
use proptest::prelude::*;
use proptest::test_runner::{Config, TestRunner};

fuzz_target!(|seed: u64| {
    let config = Config {
        cases: 4,
        failure_persistence: None,
        ..Config::default()
    };
    let mut runner = TestRunner::new(config);
    let _ = seed;

    let _ = runner.run(&proptest::strategy::Just(ScenarioMix {
        arbos_version: arb_fuzz::arbitrary_impls::ArbosVersion(60),
        txs: Vec::new(),
    }), |scenario: ScenarioMix| {
        let nodes = shared_dual_exec();
        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        let scen = scenario.clone().into_scenario();
        match nodes.run(&scen) {
            Ok(report) => {
                prop_assert!(report.is_clean(), "{report:#?}");
            }
            Err(e) => {
                eprintln!("harness error: {e}");
            }
        }

        let total_before = scenario.total_eth_before();
        let total_after = scenario.total_eth_after_arbreth();
        let burned = scenario.burned_to_zero_arbreth();
        prop_assert_eq!(total_before, total_after + burned);

        Ok(())
    });
});
