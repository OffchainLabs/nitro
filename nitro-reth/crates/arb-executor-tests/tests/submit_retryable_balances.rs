//! A submit-retryable plus its auto-redeem where the redeemer is the block
//! coinbase and is first seen during this block. The redeemer's gas is prepaid
//! at submit time, so it pays nothing net; the prepaid-gas amount is minted to
//! the redeemer only for within-tx visibility and must not leak into the
//! persisted changeset. The redeemer's revert baseline must therefore be its
//! (absent) parent-block state, not the transient prepaid mint.

use std::sync::Arc;

use alloy_consensus::transaction::{Recovered, SignerRecoverable};
use alloy_eips::eip2718::{Decodable2718, Encodable2718};
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, Address, Signature, B256, U256};
use arb_alloy_consensus::tx::ArbSubmitRetryableTx;
use arb_evm::config::ArbEvmConfig;
use arb_executor_tests::helpers::{balance_of, deploy_contract};
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{database::states::bundle_state::BundleRetention, primitives::hardfork::SpecId};

const CHAIN_ID: u64 = 421_614;
const BASE_FEE: u64 = 100_000_000;

const SENDER: Address = address!("00000000000000000000000000000000d52983fa");
const BENEFICIARY: Address = address!("00000000000000000000000000000000beef8888");
const RETRY_TO: Address = address!("00000000000000000000000000000000DEC0DEDA");

// retryData drives the submission fee = (1400 + 6 * len) * l1BaseFee.
const RETRY_DATA: [u8; 196] = alloy_primitives::hex!(
    "4201f985\
0000000000000000000000000000000000000000000000000000000000000040\
0000000000000000000000000000000000000000000000000000000000000080\
0000000000000000000000000000000000000000000000000000000000000001\
0000000000000000000000007b79995e5f793a07bc00c21412e50ecae098e7f9\
0000000000000000000000000000000000000000000000000000000000000001\
000000000000000000000000cfb1f08a4852699a979909e22c30263ca249556d"
);

#[test]
fn auto_redeem_redeemer_is_coinbase_changeset_baseline() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(10)
        .with_chain_id(CHAIN_ID)
        .initialize();

    // A redeem target that burns gas (one SSTORE) so a refund is routed.
    deploy_contract(
        harness.state(),
        RETRY_TO,
        vec![0x60, 0x01, 0x60, 0x00, 0x55, 0x00],
        U256::ZERO,
    );
    // The redeemer is absent before this block.

    let submit = ArbSubmitRetryableTx {
        chain_id: U256::from(CHAIN_ID),
        request_id: B256::from(U256::from(0x14u64).to_be_bytes()),
        from: SENDER,
        l1_base_fee: U256::from(0x1cceu64),
        deposit_value: U256::from(0xfa8bb94c940u64),
        gas_fee_cap: U256::from(0x11e1a300u64),
        gas: 0xe02f,
        retry_to: Some(RETRY_TO),
        retry_value: U256::ZERO,
        beneficiary: BENEFICIARY,
        max_submission_fee: U256::from(0x0487dc40u64),
        fee_refund_addr: BENEFICIARY,
        retry_data: RETRY_DATA.into(),
    };
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::SubmitRetryable(submit),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    // The submit's ticket id is the tx trie hash; the escrow address derives
    // from it.
    let submit_ticket_id = tx.trie_hash();

    let cfg = ArbEvmConfig::new(Arc::new(ChainSpec::default()));
    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = CHAIN_ID;
    env.cfg_env.disable_base_fee = true;
    env.block_env.timestamp = U256::from(1_700_000_000u64);
    env.block_env.basefee = BASE_FEE;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(1u64);
    // The coinbase is the redeemer (an L1->L2 message block is mined by the L1
    // sender, which is also the retryable's redeemer).
    env.block_env.beneficiary = SENDER;

    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(harness.state(), env);
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(2),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor.arb_ctx.block_timestamp = 1_700_000_000;
    executor.arb_ctx.basefee = U256::from(BASE_FEE);
    executor.arb_ctx.l2_block_number = 1;
    executor.arb_ctx.coinbase = SENDER;
    executor.apply_pre_execution_changes().expect("pre");

    // Submit, then drain and execute the auto-redeem as the producer does.
    let submit_res = executor
        .execute_transaction_without_commit(Recovered::new_unchecked(tx, SENDER))
        .expect("submit exec");
    assert!(
        submit_res.result.result.is_success(),
        "submit should succeed"
    );
    executor
        .commit_transaction(submit_res)
        .expect("submit commit");

    let scheduled = executor.drain_scheduled_txs();
    assert_eq!(scheduled.len(), 1, "submit should schedule one auto-redeem");
    let redeem = ArbTransactionSigned::decode_2718(&mut &scheduled[0][..]).expect("decode redeem");
    let redeem_res = executor
        .execute_transaction_without_commit(redeem.try_into_recovered().expect("recover"))
        .expect("redeem exec");
    assert!(
        redeem_res.result.result.is_success(),
        "redeem should succeed"
    );
    executor
        .commit_transaction(redeem_res)
        .expect("redeem commit");

    let _ = executor.finish().expect("finish");

    // The redeemer pays nothing net (the gas was prepaid at submit).
    assert_eq!(
        balance_of(harness.state(), SENDER),
        U256::ZERO,
        "redeemer (== coinbase) must net zero"
    );

    harness.state().merge_transitions(BundleRetention::Reverts);
    let escrow = arbos::retryables::retryable_escrow_address(submit_ticket_id);
    // The escrow persists present-empty (resurrected by the same-block redeem),
    // not dropped.
    let esc = harness.state().bundle_state.state.get(&escrow);
    let escrow_present = esc
        .and_then(|a| a.info.as_ref())
        .map(|i| (i.balance, i.nonce));
    assert_eq!(
        escrow_present,
        Some((U256::ZERO, 0)),
        "retryable escrow must persist present-empty (a zombie), not be dropped"
    );
    let redeemer = harness.state().bundle_state.state.get(&SENDER);
    let present = redeemer
        .and_then(|a| a.info.as_ref())
        .map(|i| i.balance)
        .unwrap_or(U256::ZERO);
    let baseline = redeemer
        .and_then(|a| a.original_info.as_ref())
        .map(|i| i.balance);
    assert_eq!(present, U256::ZERO, "persisted redeemer balance must be 0");
    assert!(
        baseline.is_none() || baseline == Some(U256::ZERO),
        "redeemer changeset revert baseline = {baseline:?}; must be the (absent) \
         parent-block state, not the transient prepaid-gas mint"
    );

    // The escrow's revert baseline is the absent parent state: unwinding the
    // block must delete it, not resurrect a present-empty account.
    let escrow_baseline = harness
        .state()
        .bundle_state
        .state
        .get(&escrow)
        .and_then(|a| a.original_info.clone());
    assert!(
        escrow_baseline.is_none(),
        "escrow revert baseline must be absent, got {escrow_baseline:?}"
    );
    harness.state().bundle_state.revert(usize::MAX);
    let escrow_after = harness
        .state()
        .bundle_state
        .state
        .get(&escrow)
        .and_then(|a| a.info.clone());
    assert!(
        escrow_after.is_none(),
        "after unwinding the block the escrow must be absent, not {escrow_after:?}"
    );
}
