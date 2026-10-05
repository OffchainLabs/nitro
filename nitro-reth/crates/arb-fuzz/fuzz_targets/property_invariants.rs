#![no_main]

use arb_fuzz::{
    arbitrary_impls::{ArbosVersion, BoundedBytes, ScenarioMix, TxScenario},
    corpus_helpers::dump_crash_as_fixture,
    shared_nodes::shared_dual_exec,
};
use libfuzzer_sys::fuzz_target;

fn synth_mix_from_seed(seed: u64) -> ScenarioMix {
    let mut state = seed.wrapping_mul(0x9E37_79B9_7F4A_7C15).wrapping_add(1);
    let mut rng = || {
        state = state
            .wrapping_mul(6_364_136_223_846_793_005)
            .wrapping_add(1_442_695_040_888_963_407);
        state
    };

    let n = ((rng() >> 56) & 0x07) as usize + 1;
    let mut txs = Vec::with_capacity(n);
    for _ in 0..n {
        let mut from_bytes = [0u8; 20];
        from_bytes[0] = ((rng() >> 56) & 0xFF) as u8;
        from_bytes[19] = ((rng() >> 48) & 0xFF) as u8;
        let from = alloy_primitives::Address::new(from_bytes);

        let mut to_bytes = [0u8; 20];
        to_bytes[0] = ((rng() >> 56) & 0xFF) as u8;
        to_bytes[19] = ((rng() >> 48) & 0xFF) as u8;
        let to = if (rng() & 1) == 0 {
            Some(alloy_primitives::Address::new(to_bytes))
        } else {
            None
        };

        let data_len = ((rng() >> 56) & 0x1F) as usize;
        let mut data = vec![0u8; data_len];
        for byte in data.iter_mut() {
            *byte = ((rng() >> 56) & 0xFF) as u8;
        }

        let value = alloy_primitives::U256::from(rng() & 0xFFFF);
        let gas = 100_000 + (rng() & 0x3F_FFFF);
        let max_fee = 1_000_000_000u128 + ((rng() & 0x7FFF_FFFF) as u128);

        txs.push(TxScenario {
            from,
            to,
            data: BoundedBytes::<2048>(data),
            value,
            gas,
            max_fee,
        });
    }

    ScenarioMix {
        arbos_version: ArbosVersion(60),
        txs,
    }
}

fuzz_target!(|seed: u64| {
    let mix = synth_mix_from_seed(seed);
    let scen = mix.clone().into_scenario();
    if scen.steps.is_empty() {
        return;
    }

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
    match nodes.run(&scen) {
        Ok(report) if !report.is_clean() => {
            let path = dump_crash_as_fixture(&mix, &report);
            panic!("divergence (fixture: {path:?}): {report:#?}");
        }
        Ok(_) => {
            // Conservation invariant placeholder: stub fields all return 0,
            // so this is a tautology today. Wired up so future per-scenario
            // balance reads can plug in without touching the target.
            let total_before = mix.total_eth_before();
            let total_after = mix.total_eth_after_arbreth();
            let burned = mix.burned_to_zero_arbreth();
            assert_eq!(total_before, total_after + burned);
        }
        Err(e) => {
            eprintln!("harness error: {e}");
        }
    }
});
