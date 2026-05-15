//! Directed differential test: an L1→L2 SubmitRetryable whose
//! `max_fee_per_gas` is below the L2 basefee.
//!
//! Background: the 179,288,677 / 179,288,678 bugs were both
//! "underpriced-from-L1 tx that Nitro drops in `state_transition.preCheck`
//! but arbreth ran anyway." We fixed it for the user-tx branch (0x65 / 0x66
//! / standard EIP-1559). RetryTx (0x68) and SubmitRetryable (0x69) take
//! their own earlier code paths and the fix doesn't directly cover them.
//!
//! This test exercises both:
//!   1. Submit a retryable with `max_fee_per_gas = 1 wei` (far below the
//!      L2 basefee, ~0.1 gwei). The user-funded `deposit_value` is enough
//!      to auto-redeem if pricing allowed.
//!   2. Nitro should:
//!        - mint the deposit to the alias (the ArbitrumDepositTx half)
//!        - reject auto-redeem because of ErrFeeCapTooLow → no RetryTx,
//!          the retryable just stays queued or fails validation
//!   3. arbreth must match — same final balances, same tx count in the block
//!      that ingests the SubmitRetryable, same retryable lifecycle state.
//!
//! If arbreth EXECUTES the underpriced auto-redeem, the diff will surface
//! a tx-count / gas_used / balance mismatch and this test fails — exactly
//! the 2% gap I flagged after the 179,288,678 fix.
//!
//! Run with:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     cargo test -p arb-fuzz --test underpriced_l1_retryable \
//!     --release -- --ignored --nocapture

use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    shared_nodes::{fuzz_arbos_version, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        retryable::{apply_l1_to_l2_alias, RetryableSubmitBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;

#[test]
#[ignore]
fn underpriced_submit_retryable_matches_canon() {
    let nodes = shared_dual_exec();

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

    let mut steps = Vec::new();

    // Fund the alias + refund addrs so Nitro's deposits don't error out.
    let l1_poster = Address::repeat_byte(0xa1);
    for to in [aliased, fee_refund, value_refund, target] {
        let idx = arb_fuzz::shared_nodes::next_msg_idx();
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

    // SubmitRetryable with max_fee_per_gas = 1 wei. The chain basefee is
    // ~0.1 gwei = 1e8 wei, so this is ~8 orders of magnitude underpriced.
    // deposit_value is large enough that, IF auto-redeem were allowed at
    // this fee, it would have funds to run.
    let builder = RetryableSubmitBuilder {
        l1_sender,
        to: target,
        l2_call_value: U256::from(1_000_000_000_000_u64), // 1 micro-ETH
        deposit_value: U256::from(10_000_000_000_000_u64), // 10 micro-ETH (covers value + sub fee)
        max_submission_fee: U256::from(1_000_000_000_u64), // 1 gwei (plenty)
        excess_fee_refund_address: fee_refund,
        call_value_refund_address: value_refund,
        gas_limit: 1_000_000,
        max_fee_per_gas: U256::from(1u64), // <<<< the underpriced part
        data: Bytes::from(vec![0x29, 0xe9, 0x9f, 0x07]), // arbitrary selector
        l1_block_number: 1,
        timestamp: 1_700_000_001,
        request_id: None,
    };
    let submit_msg = builder.build().expect("build SubmitRetryable");
    let idx = arb_fuzz::shared_nodes::next_msg_idx();
    steps.push(message_step(idx, submit_msg, idx));

    let scen = Scenario {
        name: "underpriced_submit_retryable".into(),
        description: "L1 SubmitRetryable with max_fee_per_gas << basefee — \
                      Nitro drops auto-redeem; arbreth must match"
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

    let real_block: Vec<_> = report.block_diffs.iter().collect();

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
        let path = std::path::PathBuf::from("/tmp/underpriced_submit_retryable.json");
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!(
            "arbreth diverged from Nitro on underpriced SubmitRetryable; see {}",
            path.display()
        );
    }
}
