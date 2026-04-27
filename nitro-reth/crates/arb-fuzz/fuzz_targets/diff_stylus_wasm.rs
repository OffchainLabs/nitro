#![no_main]

use arb_fuzz::arbitrary_impls::stylus::{smith_wasm, StylusFuzzInput};
use arb_fuzz::corpus_helpers::dump_crash_as_fixture;
use arb_fuzz::shared_nodes::shared_dual_exec;
use arb_test_harness::scenario::{Scenario, ScenarioSetup};
use libfuzzer_sys::fuzz_target;
use serde::Serialize;

#[derive(Debug, Serialize)]
struct StylusCrashInput {
    seed_input: StylusFuzzInput,
    wasm_hex: String,
}

fuzz_target!(|input: StylusFuzzInput| {
    let wasm = match smith_wasm(input.wasm_seed) {
        Ok(b) => b,
        Err(_) => return,
    };

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");

    let scen = Scenario {
        name: "fuzz_stylus_wasm".into(),
        description: format!(
            "fuzz-generated stylus program (seed={}, wasm_len={}, gas={})",
            input.wasm_seed,
            wasm.len(),
            input.gas_budget
        ),
        setup: ScenarioSetup {
            l2_chain_id: 412_346,
            arbos_version: 60,
            genesis: None,
        },
        steps: Vec::new(),
    };

    match nodes.run(&scen) {
        Ok(report) if !report.is_clean() => {
            let crash = StylusCrashInput {
                seed_input: input,
                wasm_hex: hex::encode(&wasm),
            };
            let path = dump_crash_as_fixture(&crash, &report);
            panic!("divergence (fixture: {path:?}): {report:#?}");
        }
        Ok(_) => {}
        Err(e) => {
            eprintln!("harness error: {e}");
        }
    }
});
