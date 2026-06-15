//! `ArbRetryableTx.cancel()` must move a cancelled ticket's escrow balance to its
//! beneficiary before clearing the ticket.
//!
//! `funded_cancel_clean` / `funded_cancel_positive`: submit a retryable WITH
//! callvalue (funds the escrow) at a gas limit below TxGas so it does not
//! auto-redeem and the escrow persists, then cancel it from the beneficiary;
//! both nodes must agree, the beneficiary must be credited the callvalue, and
//! the escrow must be drained.
//!
//! `zero_value_cancel_control_clean`: the identical scenario with ZERO
//! callvalue. With no escrow to move, both nodes agree — isolating the sweep
//! from the cancel itself.

use alloy_primitives::{Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{interop::interop_eoa, message_step},
    scaffolding::{fund_interop_eoa, selector4, signed, FUZZ_L1_BASE_FEE, INVOKE_GAS_CAP},
    shared_nodes::{next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        apply_l1_to_l2_alias, submit_retryable_ticket_id, DepositBuilder, MessageBuilder,
        RetryableSubmitBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep, StateCheck},
};

const ARBRETRYABLETX: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6e,
]);

const CALLVALUE: u128 = 10u128.pow(18);

fn cancel_calldata(ticket: B256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4("cancel(bytes32)"));
    out.extend_from_slice(ticket.as_slice());
    Bytes::from(out)
}

fn fund_l1_sender(steps: &mut Vec<ScenarioStep>, sender: Address) {
    let idx = next_msg_idx();
    let dep = DepositBuilder {
        from: sender,
        to: apply_l1_to_l2_alias(sender),
        amount: U256::from(10u128).pow(U256::from(19u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("dep");
    steps.push(message_step(idx, dep, idx));
}

/// Submit a value-bearing, zero-gas retryable whose beneficiary is the signable
/// interop EOA, then cancel it from that beneficiary. `call_value` of zero is the
/// control (no escrow); nonzero funds the escrow so the cancel must sweep it to
/// the beneficiary.
fn build_scenario(name: &str, call_value: U256) -> (Scenario, Address, Address) {
    let beneficiary = interop_eoa();
    let l1_sender = Address::repeat_byte(0xa8);

    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    fund_l1_sender(&mut steps, l1_sender);

    // gas_limit 0 keeps usergas < TxGas, so no auto-redeem runs and the escrow
    // (funded by call_value) persists for the cancel. deposit_value covers the
    // callvalue plus the max submission fee.
    let max_submission_fee = U256::from(10u128).pow(U256::from(15u64));
    let deposit_value = call_value + max_submission_fee + U256::from(10u128).pow(U256::from(17u64));

    let idx = next_msg_idx();
    let submit = RetryableSubmitBuilder {
        l1_sender,
        to: Address::repeat_byte(0xc4),
        l2_call_value: call_value,
        deposit_value,
        max_submission_fee,
        excess_fee_refund_address: beneficiary,
        call_value_refund_address: beneficiary,
        gas_limit: 0,
        max_fee_per_gas: U256::ZERO,
        data: Bytes::new(),
        l1_block_number: 3,
        timestamp: 1_700_000_010,
        request_id: Some(B256::repeat_byte(0x18)),
    }
    .build()
    .expect("submit");
    let ticket = submit_retryable_ticket_id(&submit, FUZZ_L2_CHAIN_ID).expect("ticket id");
    steps.push(message_step(idx, submit, idx));

    let cancel = signed(
        0,
        Some(ARBRETRYABLETX),
        cancel_calldata(ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("cancel");
    let idx = next_msg_idx();
    steps.push(message_step(idx, cancel, idx));

    // Trailing no-op deposit seals the cancel's block before we query balances.
    let idx = next_msg_idx();
    let seal = DepositBuilder {
        from: l1_sender,
        to: apply_l1_to_l2_alias(l1_sender),
        amount: U256::from(1u64),
        l1_block_number: 4,
        timestamp: 1_700_000_020,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("seal");
    steps.push(message_step(idx, seal, idx));

    let escrow = arbos::retryables::retryable_escrow_address(ticket);
    let scenario = Scenario {
        name: name.into(),
        description: "cancel escrow transfer to beneficiary".into(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: arb_fuzz::shared_nodes::fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };
    (scenario, beneficiary, escrow)
}

fn checks(beneficiary: Address, escrow: Address) -> Vec<StateCheck> {
    vec![
        StateCheck {
            address: beneficiary,
            slots: Vec::new(),
            check_balance: true,
            check_nonce: false,
            check_code: false,
        },
        StateCheck {
            address: escrow,
            slots: Vec::new(),
            check_balance: true,
            check_nonce: false,
            check_code: false,
        },
    ]
}

#[test]
#[ignore]
fn funded_cancel_clean() {
    let (scenario, beneficiary, escrow) =
        build_scenario("funded_cancel_clean", U256::from(CALLVALUE));
    let state_checks = checks(beneficiary, escrow);

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().unwrap_or_else(|e| e.into_inner());
    let report = nodes
        .run_with_state_checks(&scenario, &state_checks)
        .expect("dual run");

    assert!(
        report.is_clean(),
        "funded cancel must move escrow to the beneficiary and agree across nodes\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

#[test]
#[ignore]
fn funded_cancel_positive() {
    use arb_test_harness::node::{BlockId, ExecutionNode};
    let (scenario, beneficiary, escrow) =
        build_scenario("funded_cancel_positive", U256::from(CALLVALUE));
    let state_checks = checks(beneficiary, escrow);

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().unwrap_or_else(|e| e.into_inner());
    let report = nodes
        .run_with_state_checks(&scenario, &state_checks)
        .expect("dual run");
    assert!(
        report.is_clean(),
        "funded cancel must agree\n  {:#?}",
        report.state_diffs
    );

    let bene_final = nodes
        .left
        .balance(beneficiary, BlockId::Latest)
        .expect("bene bal");
    let escrow_final = nodes
        .left
        .balance(escrow, BlockId::Latest)
        .expect("escrow bal");
    assert!(
        bene_final >= U256::from(CALLVALUE),
        "beneficiary must be credited the escrow callvalue; got {bene_final}",
    );
    assert!(
        escrow_final.is_zero(),
        "escrow must be drained; got {escrow_final}"
    );
}

#[test]
#[ignore]
fn zero_value_cancel_control_clean() {
    let (scenario, beneficiary, escrow) =
        build_scenario("zero_value_cancel_control_clean", U256::ZERO);
    let state_checks = checks(beneficiary, escrow);

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().unwrap_or_else(|e| e.into_inner());
    let report = nodes
        .run_with_state_checks(&scenario, &state_checks)
        .expect("dual run");

    assert!(
        report.is_clean(),
        "zero-callvalue cancel must agree (no escrow to move)\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}
