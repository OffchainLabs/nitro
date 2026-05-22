//! Differential test: Stylus contract sub-calls a 7702-delegated EOA.
//!
//! This is the exact failure pattern of Sepolia tx 0x2bd2b083 at block
//! 213,061,213. We construct it deterministically and compare arbreth vs
//! Nitro at ArbOS v50/v51/v60 — the versions our node will be running
//! once the upgrade activates (~block 216M on Sepolia).
//!
//! Scenario:
//!   1. Fund caller EOA.
//!   2. Fund victim EOA.
//!   3. Deploy log-emitter (EVM bytecode that emits LOG1 on any call).
//!   4. SetCodeTx from victim authorising log-emitter as its delegate. => victim's code becomes
//!      `0xef 0x01 0x00 || log_emitter_addr`.
//!   5. Deploy SolCaller (Stylus contract with `forward(addr, bytes)`).
//!   6. activateProgram(SolCaller).
//!   7. caller -> SolCaller.forward(victim, "") Stylus call_contract hostio targets victim (the
//!      7702 EOA). stylus_call_trampoline must:
//!         - detect the 7702 designator
//!         - load log-emitter's code
//!         - execute it -> emit LOG1
//!         - Pre-fix arbreth: traps with all gas consumed (`0xef` invalid opcode).
//!         - Post-fix arbreth: matches Nitro byte-for-byte.
//!
//! Run command:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.0-rc.10-b1cf6db \
//!     ARB_FUZZ_ARBOS_VERSION=50 \
//!     cargo test -p arb-fuzz --test stylus_calls_7702 --release \
//!     -- --ignored --nocapture
//!
//! Re-run with ARB_FUZZ_ARBOS_VERSION=51 and =60 for the full sweep.

use alloy_primitives::{b256, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{interop::wrap_init_code, message_step},
    shared_nodes::{fuzz_arbos_version, next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, AuthorizationItem, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const INVOKE_GAS_CAP: u64 = 30_000_000;
const DEPLOY_GAS_CAP: u64 = 150_000_000;

const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
const ARBWASM_ADDR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);

// ── Fixed signing keys ──────────────────────────────────────────────────
//
// Two distinct EOAs so the SetCodeTx authority is independent from the
// transactor. Hard-coded so CREATE addresses are deterministic.
fn caller_signing_key() -> B256 {
    b256!("c701e4ad26b3a9d63b9f0f0bb3b1d2d6e2c8f6d4a3b1f0e9c7a6d5e4f3b2a1c0")
}
fn victim_signing_key() -> B256 {
    b256!("a90237e8d4c5b2a1f9e8d7c6b5a4938271605f4e3d2c1b0a9988776655443322")
}

fn selector(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

/// CREATE address (sender, nonce).
fn create_address(sender: Address, nonce: u64) -> Address {
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
            trimmed.to_vec()
        } else {
            let mut v = vec![0x80 + trimmed.len() as u8];
            v.extend_from_slice(trimmed);
            v
        }
    };
    let mut rlp = Vec::with_capacity(2 + 20 + nonce_rlp.len());
    rlp.push(0xc0 + (21 + nonce_rlp.len() - 1) as u8);
    rlp.push(0x94);
    rlp.extend_from_slice(sender.as_slice());
    rlp.extend_from_slice(&nonce_rlp);
    let h = keccak256(&rlp);
    Address::from_slice(&h[12..])
}

/// 12-byte log-emitter runtime: on any call, emit `LOG1(data="", topic=marker)`,
/// then RETURN with empty output. Lets us assert via receipt logs that the
/// delegate code (not the EOA stub) actually executed.
fn log_emitter_runtime() -> Vec<u8> {
    // PUSH1 0xaa  (topic)
    // PUSH1 0x00  (len)
    // PUSH1 0x00  (offset)
    // LOG1
    // PUSH1 0x00  (size)
    // PUSH1 0x00  (offset)
    // RETURN
    vec![
        0x60, 0xaa, 0x60, 0x00, 0x60, 0x00, 0xa1, 0x60, 0x00, 0x60, 0x00, 0xf3,
    ]
}

fn signed_with_key(
    signing_key: B256,
    nonce: u64,
    to: Option<Address>,
    data: Bytes,
    value: U256,
    gas: u64,
    auths: Vec<AuthorizationItem>,
) -> SignedL2TxBuilder {
    let kind = if auths.is_empty() {
        L2TxKind::Eip1559
    } else {
        L2TxKind::Eip7702
    };
    SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
        nonce,
        to,
        value,
        data,
        gas_limit: gas,
        gas_price: 0,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: auths,
        kind,
        signing_key,
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

#[test]
#[ignore]
fn stylus_calls_7702_delegated_eoa_matches_canon() {
    let caller_key = caller_signing_key();
    let victim_key = victim_signing_key();
    let caller = derive_address(caller_key);
    let victim = derive_address(victim_key);

    let mut steps: Vec<ScenarioStep> = Vec::new();

    // 1. Fund caller.
    let idx = next_msg_idx();
    let fund_caller = DepositBuilder {
        from: caller,
        to: caller,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund caller");
    steps.push(message_step(idx, fund_caller, idx));

    // 2. Fund victim (small amount — just enough to pay for the SetCodeTx).
    let idx = next_msg_idx();
    let fund_victim = DepositBuilder {
        from: victim,
        to: victim,
        amount: U256::from(10u128).pow(U256::from(18u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund victim");
    steps.push(message_step(idx, fund_victim, idx));

    // 3. caller deploys log-emitter (EVM contract).
    let log_emitter_nonce = 0u64;
    let log_emitter_addr = create_address(caller, log_emitter_nonce);
    let deploy_emitter = signed_with_key(
        caller_key,
        log_emitter_nonce,
        None,
        Bytes::from(wrap_init_code(&log_emitter_runtime())),
        U256::ZERO,
        2_000_000,
        Vec::new(),
    )
    .build()
    .expect("deploy log-emitter");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_emitter, idx));

    // 4. SetCodeTx from victim authorising log-emitter as delegate. EIP-7702 requires `to` to be
    //    set; we send it to log_emitter_addr with empty calldata (a no-op call). The interesting
    //    part is the authorization list, which installs the delegation on `victim`.
    let auth = AuthorizationItem {
        chain_id: FUZZ_L2_CHAIN_ID,
        address: log_emitter_addr,
        nonce: 0,
        signing_key: victim_key,
    };
    let set_code = signed_with_key(
        victim_key,
        0,
        Some(log_emitter_addr),
        Bytes::new(),
        U256::ZERO,
        INVOKE_GAS_CAP,
        vec![auth],
    )
    .build()
    .expect("set-code tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, set_code, idx));

    // 5. caller deploys SolCaller (Stylus initcode).
    let sol_caller_nonce = 1u64;
    let sol_caller_addr = create_address(caller, sol_caller_nonce);
    let sol_caller_initcode_hex = include_str!("../prebuilt/sol_caller.hex").trim();
    let sol_caller_initcode: Vec<u8> = (0..sol_caller_initcode_hex.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&sol_caller_initcode_hex[i..i + 2], 16).expect("hex"))
        .collect();
    let deploy_stylus = signed_with_key(
        caller_key,
        sol_caller_nonce,
        None,
        Bytes::from(sol_caller_initcode),
        U256::ZERO,
        DEPLOY_GAS_CAP,
        Vec::new(),
    )
    .build()
    .expect("deploy stylus");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_stylus, idx));

    // 6. activateProgram(sol_caller_addr).
    let mut activate_data = Vec::with_capacity(4 + 32);
    activate_data.extend_from_slice(&selector("activateProgram(address)"));
    let mut padded = [0u8; 32];
    padded[12..].copy_from_slice(sol_caller_addr.as_slice());
    activate_data.extend_from_slice(&padded);
    let activate = signed_with_key(
        caller_key,
        2,
        Some(ARBWASM_ADDR),
        Bytes::from(activate_data),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
        Vec::new(),
    )
    .build()
    .expect("activate");
    let idx = next_msg_idx();
    steps.push(message_step(idx, activate, idx));

    // 7. caller -> SolCaller.forward(victim, "") SolCaller's `forward(address target, bytes data)
    //    -> bytes` sub-CALLs `target` from inside Stylus. With `target = victim` (a 7702-delegated
    //    EOA), the trampoline must follow the delegation and execute log-emitter's code.
    let mut forward_data = Vec::with_capacity(4 + 32 + 32 + 32);
    forward_data.extend_from_slice(&selector("forward(address,bytes)"));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(victim.as_slice());
    forward_data.extend_from_slice(&pad); // target
    forward_data.extend_from_slice(&{
        let mut b = [0u8; 32];
        b[31] = 0x40; // offset to bytes
        b
    });
    forward_data.extend_from_slice(&[0u8; 32]); // bytes length = 0 (empty calldata)
    let forward = signed_with_key(
        caller_key,
        3,
        Some(sol_caller_addr),
        Bytes::from(forward_data),
        U256::ZERO,
        INVOKE_GAS_CAP,
        Vec::new(),
    )
    .build()
    .expect("forward call");
    let idx = next_msg_idx();
    steps.push(message_step(idx, forward, idx));

    let scen = Scenario {
        name: "stylus_calls_7702_delegated_eoa".into(),
        description: "Stylus -> 7702 EOA -> delegate (log-emitter)".into(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
    let report = nodes.run(&scen).expect("run stylus_calls_7702 scenario");

    if !report.is_clean() {
        let payload = serde_json::json!({
            "arbos_version": fuzz_arbos_version(),
            "block_diffs": format!("{:#?}", report.block_diffs),
            "tx_diffs": format!("{:#?}", report.tx_diffs),
            "state_diffs": format!("{:#?}", report.state_diffs),
            "log_diffs": format!("{:#?}", report.log_diffs),
        });
        let path = std::path::PathBuf::from(format!(
            "/tmp/stylus_calls_7702_v{}.json",
            fuzz_arbos_version()
        ));
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!(
            "arbreth diverged from Nitro on stylus_calls_7702 at ArbOS v{}; see {}",
            fuzz_arbos_version(),
            path.display()
        );
    }
}
