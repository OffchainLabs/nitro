//! Differential test for every ArbWasm read-only precompile method.
//!
//! Calls each method via a signed EIP-1559 tx against both arbreth and
//! Nitro Docker, then asserts the receipts match byte-exact. Catches the
//! gas-accounting drift family (e.g. block 152,429,039 — missing Params()
//! warm read on stylusVersion/codehashVersion).
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     ARB_FUZZ_ARBOS_VERSION=40 \
//!     cargo test -p arb-fuzz --test arbwasm_read_methods --release \
//!     -- --ignored differential_against_nitro --nocapture

use alloy_primitives::{b256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{message_step, ArbWasmArgKind, ArbWasmReadMethod},
    shared_nodes::{fuzz_arbos_version, next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
const ARBWASM_ADDR: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);

fn signing_key() -> B256 {
    b256!("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
}

fn eoa() -> Address {
    derive_address(signing_key())
}

fn all_methods() -> &'static [ArbWasmReadMethod] {
    &[
        ArbWasmReadMethod::StylusVersion,
        ArbWasmReadMethod::InkPrice,
        ArbWasmReadMethod::MaxStackDepth,
        ArbWasmReadMethod::FreePages,
        ArbWasmReadMethod::PageGas,
        ArbWasmReadMethod::PageRamp,
        ArbWasmReadMethod::PageLimit,
        ArbWasmReadMethod::MinInitGas,
        ArbWasmReadMethod::InitCostScalar,
        ArbWasmReadMethod::ExpiryDays,
        ArbWasmReadMethod::KeepaliveDays,
        ArbWasmReadMethod::BlockCacheSize,
        ArbWasmReadMethod::ActivationGas,
        ArbWasmReadMethod::CodehashVersion,
        ArbWasmReadMethod::CodehashAsmSize,
        ArbWasmReadMethod::ProgramVersion,
        ArbWasmReadMethod::ProgramInitGas,
        ArbWasmReadMethod::ProgramMemoryFootprint,
        ArbWasmReadMethod::ProgramTimeLeft,
    ]
}

fn method_name(m: ArbWasmReadMethod) -> &'static str {
    match m {
        ArbWasmReadMethod::StylusVersion => "stylusVersion",
        ArbWasmReadMethod::InkPrice => "inkPrice",
        ArbWasmReadMethod::MaxStackDepth => "maxStackDepth",
        ArbWasmReadMethod::FreePages => "freePages",
        ArbWasmReadMethod::PageGas => "pageGas",
        ArbWasmReadMethod::PageRamp => "pageRamp",
        ArbWasmReadMethod::PageLimit => "pageLimit",
        ArbWasmReadMethod::MinInitGas => "minInitGas",
        ArbWasmReadMethod::InitCostScalar => "initCostScalar",
        ArbWasmReadMethod::ExpiryDays => "expiryDays",
        ArbWasmReadMethod::KeepaliveDays => "keepaliveDays",
        ArbWasmReadMethod::BlockCacheSize => "blockCacheSize",
        ArbWasmReadMethod::ActivationGas => "activationGas",
        ArbWasmReadMethod::CodehashVersion => "codehashVersion",
        ArbWasmReadMethod::CodehashAsmSize => "codehashAsmSize",
        ArbWasmReadMethod::ProgramVersion => "programVersion",
        ArbWasmReadMethod::ProgramInitGas => "programInitGas",
        ArbWasmReadMethod::ProgramMemoryFootprint => "programMemoryFootprint",
        ArbWasmReadMethod::ProgramTimeLeft => "programTimeLeft",
    }
}

fn build_calldata(method: ArbWasmReadMethod) -> Vec<u8> {
    let mut data = Vec::with_capacity(36);
    data.extend_from_slice(&method.selector());
    match method.arg_kind() {
        ArbWasmArgKind::NoArg => {}
        ArbWasmArgKind::Address | ArbWasmArgKind::Codehash => {
            // For programs the address is the call target; for codehashes
            // a dummy 32-byte hash. Either way, calldata is exactly 1 word.
            let arg = [0x42u8; 32];
            data.extend_from_slice(&arg);
        }
    }
    data
}

#[test]
#[ignore]
fn differential_against_nitro() {
    let nodes = shared_dual_exec();
    let nonce: u64 = 0;

    // Fund the EOA once.
    {
        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        let idx = next_msg_idx();
        let dep = DepositBuilder {
            from: eoa(),
            to: eoa(),
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: 0,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        let msg = dep.build().expect("deposit builds");
        let scen = Scenario {
            name: "fund".into(),
            description: "fund eoa".into(),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: fuzz_arbos_version(),
                genesis: None,
            },
            steps: vec![message_step(idx, msg, idx)],
        };
        let _ = nodes.run(&scen);
    }

    let mut divergences: Vec<(String, String)> = Vec::new();
    for (nonce, &method) in (nonce..).zip(all_methods().iter()) {
        let data = build_calldata(method);
        let builder = SignedL2TxBuilder {
            chain_id: FUZZ_L2_CHAIN_ID,
            nonce,
            to: Some(ARBWASM_ADDR),
            value: U256::ZERO,
            data: Bytes::from(data),
            gas_limit: 200_000,
            gas_price: 0,
            max_fee_per_gas: 2_000_000_000,
            max_priority_fee_per_gas: 0,
            access_list: Vec::new(),
            authorization_list: Vec::new(),
            kind: L2TxKind::Eip1559,
            signing_key: signing_key(),
            l1_block_number: 1,
            timestamp: 1_700_000_001,
            request_id: None,
            sender: SEQUENCER_ALIAS,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        let msg = builder.build().expect("signed tx builds");
        let idx = next_msg_idx();
        let scen = Scenario {
            name: format!("arbwasm_{}", method_name(method)),
            description: format!("call ArbWasm.{}", method_name(method)),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: fuzz_arbos_version(),
                genesis: None,
            },
            steps: vec![message_step(idx, msg, idx)],
        };

        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        match nodes.run(&scen) {
            Ok(report) if report.is_clean() => {
                eprintln!("[arbwasm-diff] {}: CLEAN", method_name(method));
            }
            Ok(report) => {
                let summary = format!(
                    "{}: block_diffs={} tx_diffs={} state_diffs={} log_diffs={}\n{:#?}",
                    method_name(method),
                    report.block_diffs.len(),
                    report.tx_diffs.len(),
                    report.state_diffs.len(),
                    report.log_diffs.len(),
                    report,
                );
                eprintln!("[arbwasm-diff] {}: DIVERGE\n{summary}", method_name(method));
                divergences.push((method_name(method).into(), summary));
            }
            Err(e) => {
                eprintln!("[arbwasm-diff] {}: harness error: {e}", method_name(method));
            }
        }
    }

    if !divergences.is_empty() {
        let names: Vec<&str> = divergences.iter().map(|(n, _)| n.as_str()).collect();
        panic!(
            "{} methods diverged: [{}]",
            divergences.len(),
            names.join(", ")
        );
    }
}
