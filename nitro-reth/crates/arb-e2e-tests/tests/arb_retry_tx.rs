use std::sync::Arc;

use alloy_consensus::transaction::Recovered;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, Address, Bytes, Signature, B256, U256};
use arb_alloy_consensus::tx::ArbRetryTx;
use arb_e2e_tests::helpers::{
    alice, balance_of, fund_account, ExecutorScaffold, ONE_ETH, ONE_GWEI,
};
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_test_utils::ArbosHarness;
use arbos::retryables::retryable_escrow_address;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

fn run_tx(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    tx: ArbTransactionSigned,
    sender: Address,
) -> Result<bool, String> {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = chain_id;
    env.cfg_env.disable_base_fee = true;
    env.block_env.timestamp = U256::from(1_700_000_000u64);
    env.block_env.basefee = base_fee;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(1u64);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);

    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(harness.state(), env);

    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(1),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, chain_id);
    executor.arb_ctx.block_timestamp = 1_700_000_000;
    executor.arb_ctx.basefee = U256::from(base_fee);
    executor.arb_ctx.l2_block_number = 1;
    executor
        .apply_pre_execution_changes()
        .map_err(|e| format!("pre: {e}"))?;

    let recovered = Recovered::new_unchecked(tx, sender);
    let result = executor
        .execute_transaction_without_commit(recovered)
        .map_err(|e| format!("exec: {e}"))?;
    let success = result.result.result.is_success();
    executor
        .commit_transaction(result)
        .map_err(|e| format!("commit: {e}"))?;
    let _ = executor.finish().map_err(|e| format!("finish: {e}"))?;
    Ok(success)
}

fn zero_sig() -> Signature {
    Signature::new(U256::ZERO, U256::ZERO, false)
}

#[test]
fn retry_tx_redeems_existing_retryable() {
    let mut s = ExecutorScaffold::new();
    let chain_id = s.chain_id;
    let sender = alice();
    let refund_to = address!("0000000000000000000000000000000000000B0B");
    let recipient = address!("00000000000000000000000000000000DEC0DEDA");

    let ticket_id = B256::repeat_byte(0xAB);
    let retry_value = U256::from(ONE_ETH / 2);
    let escrow = retryable_escrow_address(ticket_id);

    fund_account(s.harness.state(), sender, U256::from(10u128 * ONE_ETH));
    fund_account(s.harness.state(), escrow, retry_value);

    s.harness
        .retryable_state()
        .create_retryable(
            ticket_id,
            1_700_000_000 + 604_800,
            sender,
            Some(recipient),
            retry_value,
            refund_to,
            &[],
        )
        .unwrap();

    let retry_tx = ArbRetryTx {
        chain_id: U256::from(chain_id),
        nonce: 0,
        from: sender,
        gas_fee_cap: U256::from(ONE_GWEI),
        gas: 100_000,
        to: Some(recipient),
        value: retry_value,
        data: Bytes::new(),
        ticket_id,
        refund_to,
        max_refund: U256::ZERO,
        submission_fee_refund: U256::ZERO,
    };

    let tx = ArbTransactionSigned::new_unhashed(ArbTypedTransaction::Retry(retry_tx), zero_sig());

    let recipient_before = balance_of(s.harness.state(), recipient);
    let success = run_tx(&mut s.harness, s.base_fee, chain_id, tx, sender).expect("exec");
    assert!(success, "retry tx should succeed");

    let recipient_after = balance_of(s.harness.state(), recipient);
    assert_eq!(
        recipient_after - recipient_before,
        retry_value,
        "recipient should receive the retry value"
    );

    let rs = s.harness.retryable_state();
    assert!(
        rs.open_retryable(ticket_id, 1_700_000_000)
            .unwrap()
            .is_none(),
        "successful retry should delete the retryable"
    );
}

#[test]
fn retry_tx_failure_keeps_retryable_alive() {
    let mut s = ExecutorScaffold::new();
    let chain_id = s.chain_id;
    let sender = alice();
    let refund_to = address!("0000000000000000000000000000000000000B0B");
    let revert_contract = address!("00000000000000000000000000000000DEADFACE");

    let ticket_id = B256::repeat_byte(0xCD);
    let retry_value = U256::from(ONE_ETH / 4);
    let escrow = retryable_escrow_address(ticket_id);

    fund_account(s.harness.state(), sender, U256::from(10u128 * ONE_ETH));
    fund_account(s.harness.state(), escrow, retry_value);
    arb_e2e_tests::helpers::deploy_contract(
        s.harness.state(),
        revert_contract,
        vec![0x60, 0x00, 0x60, 0x00, 0xFD],
        U256::ZERO,
    );

    s.harness
        .retryable_state()
        .create_retryable(
            ticket_id,
            1_700_000_000 + 604_800,
            sender,
            Some(revert_contract),
            retry_value,
            refund_to,
            &[],
        )
        .unwrap();

    let retry_tx = ArbRetryTx {
        chain_id: U256::from(chain_id),
        nonce: 0,
        from: sender,
        gas_fee_cap: U256::from(ONE_GWEI),
        gas: 100_000,
        to: Some(revert_contract),
        value: retry_value,
        data: Bytes::new(),
        ticket_id,
        refund_to,
        max_refund: U256::ZERO,
        submission_fee_refund: U256::ZERO,
    };

    let tx = ArbTransactionSigned::new_unhashed(ArbTypedTransaction::Retry(retry_tx), zero_sig());

    let _ = run_tx(&mut s.harness, s.base_fee, chain_id, tx, sender).expect("exec");

    let rs = s.harness.retryable_state();
    assert!(
        rs.open_retryable(ticket_id, 1_700_000_000)
            .unwrap()
            .is_some(),
        "failed retry must keep the retryable in state"
    );
}
