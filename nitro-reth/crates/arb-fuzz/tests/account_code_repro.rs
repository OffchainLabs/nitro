//! Pins the Stylus account_code repeat-read gas charge: a program that reads a
//! large data contract's code via account_code in many small chunks is charged
//! the account-touch cost only on the first read of an address. Compares
//! arbreth vs Nitro per-block gas.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     cargo test -p arb-fuzz --test account_code_repro --release -- --ignored --nocapture

use alloy_primitives::{Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    scaffolding::{
        activate_program, deploy_solidity, eoa_create_addr, fund_interop_eoa, signed,
        DEPLOY_GAS_CAP, INVOKE_GAS_CAP,
    },
    shared_nodes::{fuzz_arbos_version, next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::MessageBuilder,
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const WAT_ACCOUNT_CODE_LOOP: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../arb-spec-tests/fixtures/_wat/hostio_account_code_loop.wat"
));

/// Wrap raw Stylus WASM in the activation prefix + a minimal CODECOPY deployer.
fn build_init_code(wasm: &[u8]) -> Vec<u8> {
    let mut body = Vec::with_capacity(3 + wasm.len());
    body.extend_from_slice(&[0xEF, 0xF0, 0x00]);
    body.extend_from_slice(wasm);
    let size = body.len();
    let size_hi = ((size >> 8) & 0xFF) as u8;
    let size_lo = (size & 0xFF) as u8;
    let mut out = Vec::with_capacity(14 + size);
    out.extend_from_slice(&[
        0x61, size_hi, size_lo, 0x60, 0x0c, 0x60, 0x00, 0x39, 0x61, size_hi, size_lo, 0x60, 0x00,
        0xF3,
    ]);
    out.extend_from_slice(&body);
    out
}

#[test]
#[ignore]
fn account_code_loop_gas_parity() {
    let wasm = wat::parse_bytes(WAT_ACCOUNT_CODE_LOOP.as_bytes())
        .expect("wat compile")
        .into_owned();

    let mut steps: Vec<ScenarioStep> = Vec::new();
    fund_interop_eoa(&mut steps);

    // Data contract: 24,513-byte blob, read by account_code in chunks.
    let data_runtime = vec![0u8; 24_513];
    let data_addr = deploy_solidity(&mut steps, 0, &data_runtime);

    // Stylus account_code-loop program.
    let init = build_init_code(&wasm);
    let deploy = signed(1, None, Bytes::from(init), U256::ZERO, DEPLOY_GAS_CAP)
        .build()
        .expect("deploy stylus");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy, idx));
    let stylus_addr = eoa_create_addr(1);
    activate_program(&mut steps, 2, stylus_addr);

    // Invoke: calldata = target data-contract address (20 bytes).
    let invoke = signed(
        3,
        Some(stylus_addr),
        Bytes::from(data_addr.as_slice().to_vec()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("invoke");
    let idx = next_msg_idx();
    steps.push(message_step(idx, invoke, idx));

    let scen = Scenario {
        name: "account_code_loop".into(),
        description: "Stylus account_code chunked reads of a 24KB data contract".into(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex");
    let report = nodes.run(&scen).expect("run");

    eprintln!("data_addr={data_addr} stylus_addr={stylus_addr}");
    eprintln!(
        "block_diffs={} tx_diffs={} state_diffs={} log_diffs={}",
        report.block_diffs.len(),
        report.tx_diffs.len(),
        report.state_diffs.len(),
        report.log_diffs.len()
    );
    for d in &report.block_diffs {
        eprintln!(
            "  BLOCK {} {} left(nitro)={} right(arbreth)={}",
            d.number, d.field, d.left, d.right
        );
    }
    for d in &report.tx_diffs {
        eprintln!(
            "  TX {:?} {} left(nitro)={} right(arbreth)={}",
            d.tx_hash, d.field, d.left, d.right
        );
    }
    assert!(
        report.is_clean(),
        "account_code loop diverged (see diffs above)"
    );
}
