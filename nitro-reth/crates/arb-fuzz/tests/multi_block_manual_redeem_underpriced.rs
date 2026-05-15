//! Multi-block manual-redeem parity test for an underpriced retryable.
//!
//! Companion to `underpriced_l1_retryable.rs`. The submitted retryable has
//! `max_fee_per_gas = 1 wei`, far below the L2 basefee. Auto-redeem is
//! suppressed (the single-block test covers that). This test exercises the
//! *next* block: an EOA-signed call to `ArbRetryableTx.redeem(bytes32)`
//! that schedules a manual retry tx.
//!
//! Architectural observation worth recording: the retry tx constructed by
//! the `redeem` precompile takes `gas_fee_cap = currentBaseFee` (see
//! `arbos/retryables/retryable.rs::make_tx` callers in both
//! `arb-evm/src/build.rs` and `arbos/precompiles/ArbRetryableTx.go`). So
//! the manual-redeem path is structurally immune to "user-supplied
//! underpriced fee cap" bugs: by the time the retry tx is built, the
//! original `max_fee_per_gas = 1` has been replaced. This test confirms
//! parity in that exact path — both nodes must agree on the resulting
//! tx count, gas, and balance changes.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     cargo test -p arb-fuzz --test multi_block_manual_redeem_underpriced \
//!     --release -- --ignored --nocapture

use alloy_primitives::{b256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    shared_nodes::{fuzz_arbos_version, next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        retryable::{apply_l1_to_l2_alias, RetryableSubmitBuilder},
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
/// `ArbRetryableTx` precompile address.
const ARB_RETRYABLE_TX: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6e,
]);
/// 4-byte selector for `redeem(bytes32)`.
const REDEEM_SELECTOR: [u8; 4] = [0xed, 0xa1, 0x12, 0x2c];

fn redeemer_key() -> B256 {
    b256!("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
}

fn redeemer_addr() -> Address {
    derive_address(redeemer_key())
}

#[test]
#[ignore]
fn multi_block_manual_redeem_underpriced_matches_canon() {
    let nodes = shared_dual_exec();

    // L1 sender that posted the retryable. Its alias receives the deposit
    // half on Nitro; we mirror that funding on arbreth so retry execution
    // semantics aren't perturbed by missing funds anywhere.
    let l1_sender = Address::new([
        0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
        0x11, 0x22, 0x33, 0x44, 0x55,
    ]);
    let aliased = apply_l1_to_l2_alias(l1_sender);
    let target = Address::new([
        0xee, 0x16, 0x9c, 0x35, 0xdc, 0xf9, 0xdb, 0xda, 0x9f, 0x95, 0xd5, 0xbe, 0x92, 0x97, 0x12,
        0x29, 0xd8, 0xa2, 0x98, 0x57,
    ]);
    let fee_refund = Address::repeat_byte(0xfe);
    let value_refund = Address::repeat_byte(0xed);

    // Deterministic ticket id so the retryable can be referenced by the
    // later redeem tx without scanning logs.
    let ticket_id: B256 =
        b256!("aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899");

    let l1_poster = Address::repeat_byte(0xa1);

    let mut steps = Vec::new();

    // Block N — Setup: fund the alias, refund addresses, target, and the
    // EOA that will issue the manual redeem.
    for to in [aliased, fee_refund, value_refund, target, redeemer_addr()] {
        let idx = next_msg_idx();
        let dep = DepositBuilder {
            from: l1_poster,
            to,
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: idx,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Ok(msg) = dep.build() {
            steps.push(message_step(idx, msg, idx));
        }
    }

    // Block N+1 — Submit the underpriced retryable. Same shape as
    // `underpriced_l1_retryable.rs`: gas_fee_cap=1 << basefee, so the
    // auto-redeem is suppressed. `deposit_value` is enough to fund both
    // the submission fee and the would-be auto-redeem gas, so any
    // divergence in the auto-redeem branch shows up as a balance diff
    // rather than a control-flow drop.
    let submit_builder = RetryableSubmitBuilder {
        l1_sender,
        to: target,
        l2_call_value: U256::from(1_000_000_000_000_u64),
        deposit_value: U256::from(50_000_000_000_000_000_u64), // 0.05 ETH; covers gas+submission
        max_submission_fee: U256::from(1_000_000_000_u64),
        excess_fee_refund_address: fee_refund,
        call_value_refund_address: value_refund,
        gas_limit: 500_000,
        max_fee_per_gas: U256::from(1u64),
        data: Bytes::from(vec![0x29, 0xe9, 0x9f, 0x07]),
        l1_block_number: 2,
        timestamp: 1_700_000_001,
        request_id: Some(ticket_id),
    };
    let submit_msg = submit_builder.build().expect("build SubmitRetryable");
    let submit_idx = next_msg_idx();
    steps.push(message_step(submit_idx, submit_msg, submit_idx));

    // Block N+2 — Manual redeem. EOA-signed EIP-1559 tx calling
    // `ArbRetryableTx.redeem(ticket_id)`. The redeem precompile builds the
    // retry tx with `gas_fee_cap = current basefee`, so the basefee gate
    // we added for ArbitrumUnsignedTx / ArbitrumContractTx does not
    // apply: the retry tx looks correctly priced by the time it reaches
    // pre-check. Both nodes execute it.
    let mut redeem_calldata = Vec::with_capacity(36);
    redeem_calldata.extend_from_slice(&REDEEM_SELECTOR);
    redeem_calldata.extend_from_slice(ticket_id.as_slice());

    let redeem_builder = SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
        nonce: 0,
        to: Some(ARB_RETRYABLE_TX),
        value: U256::ZERO,
        data: Bytes::from(redeem_calldata),
        gas_limit: 2_000_000,
        gas_price: 0,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: redeemer_key(),
        l1_block_number: 3,
        timestamp: 1_700_000_002,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    };
    let redeem_msg = redeem_builder.build().expect("build redeem signed tx");
    let redeem_idx = next_msg_idx();
    steps.push(message_step(redeem_idx, redeem_msg, redeem_idx));

    let scen = Scenario {
        name: "multi_block_manual_redeem_underpriced".into(),
        description: "Submit retryable with maxFeePerGas << basefee (no auto-redeem), \
                      then manual redeem from an EOA in a later block. Both nodes \
                      must agree on tx count, gas, balances, and emitted logs."
            .into(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };

    let report = {
        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        nodes.run(&scen).expect("run scenario")
    };

    // Strip pre-existing Nitro vs reth state-trie hashing noise.
    let real_block: Vec<_> = report
        .block_diffs
        .iter()
        .filter(|d| d.field != "state_root" && d.field != "parent_hash")
        .collect();

    if !real_block.is_empty()
        || !report.tx_diffs.is_empty()
        || !report.state_diffs.is_empty()
        || !report.log_diffs.is_empty()
    {
        let payload = serde_json::json!({
            "block_diffs": format!("{:#?}", real_block),
            "tx_diffs":    format!("{:#?}", report.tx_diffs),
            "state_diffs": format!("{:#?}", report.state_diffs),
            "log_diffs":   format!("{:#?}", report.log_diffs),
        });
        let path = std::path::PathBuf::from("/tmp/multi_block_manual_redeem_underpriced.json");
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!(
            "arbreth diverged from Nitro on multi-block manual-redeem of \
             underpriced retryable; see {}",
            path.display()
        );
    }
}
