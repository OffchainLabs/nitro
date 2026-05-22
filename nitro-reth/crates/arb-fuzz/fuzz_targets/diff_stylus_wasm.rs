#![no_main]

use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::{
        message_step,
        stylus::{smith_wasm, StylusFuzzInput},
    },
    corpus_helpers::dump_crash_as_fixture,
    shared_nodes::{shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{ContractTxBuilder, DepositBuilder, MessageBuilder},
    scenario::{Scenario, ScenarioSetup},
};
use libfuzzer_sys::fuzz_target;
use serde::Serialize;

const FUZZ_ARBOS_VERSION: u64 = 60;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const FUZZ_GAS_CAP: u64 = 8_000_000;

#[derive(Debug, Serialize)]
struct StylusCrashInput {
    seed_input: StylusFuzzInput,
    wasm_hex: String,
}

fn deployer() -> Address {
    let mut b = [0u8; 20];
    b[0] = 0xab;
    b[19] = 0xcd;
    Address::new(b)
}

fn caller() -> Address {
    let mut b = [0u8; 20];
    b[0] = 0xfe;
    b[19] = 0xed;
    Address::new(b)
}

/// EVM init code that copies its codecopy region and returns it. The body
/// shipped after the init prologue is the raw WASM with the Stylus
/// discriminant prefix; whether the deployer accepts it is exactly what the
/// fuzzer wants to differentially test.
fn build_init_code(wasm: &[u8]) -> Vec<u8> {
    // Body to be returned: 0xEF 0xF0 0x00 || wasm
    let mut body = Vec::with_capacity(3 + wasm.len());
    body.extend_from_slice(&[0xEF, 0xF0, 0x00]);
    body.extend_from_slice(wasm);

    // Init prologue: PUSH2 size; PUSH1 0x0c; PUSH1 0x00; CODECOPY; PUSH2 size; PUSH1 0x00; RETURN
    // Layout uses a fixed prologue of 12 bytes so the body offset is constant.
    let size = body.len();
    let size_hi = ((size >> 8) & 0xFF) as u8;
    let size_lo = (size & 0xFF) as u8;
    let mut out = Vec::with_capacity(12 + size);
    out.extend_from_slice(&[
        0x61, size_hi, size_lo, // PUSH2 size
        0x60, 0x0c, // PUSH1 0x0c (offset)
        0x60, 0x00, // PUSH1 0x00 (dest)
        0x39, // CODECOPY
        0x61, size_hi, size_lo, // PUSH2 size
        0x60, 0x00, // PUSH1 0x00
        0xF3, // RETURN
    ]);
    out.extend_from_slice(&body);
    out
}

fuzz_target!(|input: StylusFuzzInput| {
    let wasm = match smith_wasm(input.wasm_seed) {
        Ok(b) if !b.is_empty() => b,
        _ => return,
    };

    let mut steps = Vec::new();
    let mut idx: u64 = 1;
    let mut delayed: u64 = 0;

    // Fund the deployer + caller so they can pay gas.
    for addr in [deployer(), caller()] {
        let dep = DepositBuilder {
            from: addr,
            to: addr,
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: idx,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Ok(msg) = dep.build() {
            delayed += 1;
            steps.push(message_step(idx, msg, delayed));
            idx += 1;
        }
    }

    // CREATE the contract carrying the WASM.
    let init_code = build_init_code(&wasm);
    let create = ContractTxBuilder {
        from: deployer(),
        gas_limit: input.gas_budget.clamp(500_000, FUZZ_GAS_CAP),
        max_fee_per_gas: U256::from(2_000_000_000u64),
        to: Address::ZERO,
        value: U256::ZERO,
        data: Bytes::from(init_code),
        l1_block_number: 1,
        timestamp: 1_700_000_001,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    };
    if let Ok(msg) = create.build() {
        delayed += 1;
        steps.push(message_step(idx, msg, delayed));
        idx += 1;
    }

    // Invoke whatever address the create produced. We don't know the address
    // a priori, so target the conventional CREATE address derived from the
    // deployer + nonce 0 and let the harness diff failures uniformly. The
    // important property is that both nodes either agree on success or
    // agree on the same failure mode.
    let probe_addr = create_address(deployer(), 0);
    let invoke = ContractTxBuilder {
        from: caller(),
        gas_limit: input.gas_budget.clamp(200_000, FUZZ_GAS_CAP),
        max_fee_per_gas: U256::from(2_000_000_000u64),
        to: probe_addr,
        value: U256::ZERO,
        data: Bytes::from(input.calldata.0.clone()),
        l1_block_number: 1,
        timestamp: 1_700_000_002,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    };
    if let Ok(msg) = invoke.build() {
        delayed += 1;
        steps.push(message_step(idx, msg, delayed));
    }

    if steps.is_empty() {
        return;
    }

    let scen = Scenario {
        name: "fuzz_stylus_wasm".into(),
        description: format!(
            "fuzz-generated stylus program (seed={}, wasm_len={}, gas={})",
            input.wasm_seed,
            wasm.len(),
            input.gas_budget
        ),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: FUZZ_ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
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

/// Manual CREATE address derivation (`keccak256(rlp([sender, nonce]))[12..]`)
/// for nonce=0; matches the Ethereum/Arbitrum convention.
fn create_address(sender: Address, nonce: u64) -> Address {
    use alloy_primitives::keccak256;
    // RLP-encode [sender, nonce]. For nonce=0 the encoding is 0x80; otherwise
    // it is the integer's big-endian byte sequence with appropriate prefix.
    let nonce_rlp = if nonce == 0 {
        vec![0x80u8]
    } else {
        let bytes = nonce.to_be_bytes();
        let trimmed: &[u8] = bytes
            .iter()
            .position(|b| *b != 0)
            .map(|i| &bytes[i..])
            .unwrap_or(&bytes[..0]);
        if trimmed.len() == 1 && trimmed[0] < 0x80 {
            vec![trimmed[0]]
        } else {
            let mut v = vec![0x80 + trimmed.len() as u8];
            v.extend_from_slice(trimmed);
            v
        }
    };
    let mut payload = Vec::new();
    payload.push(0x80 + 20); // sender length prefix
    payload.extend_from_slice(sender.as_slice());
    payload.extend_from_slice(&nonce_rlp);
    let mut rlp = vec![0xC0 + payload.len() as u8];
    rlp.extend_from_slice(&payload);
    let hash = keccak256(&rlp);
    Address::from_slice(&hash.as_slice()[12..])
}
