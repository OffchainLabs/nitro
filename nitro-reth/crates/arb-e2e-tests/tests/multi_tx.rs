use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
use arb_e2e_tests::helpers::{
    alice, alice_key, balance_of, bob, bob_key, charlie, charlie_key, deploy_contract,
    fund_account, nonce_of, recover, sign_legacy, ExecutorScaffold, ONE_ETH, ONE_GWEI, RECIPIENT,
};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

fn run_multi_tx_block(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    txs: Vec<arb_primitives::ArbTransactionSigned>,
) -> Vec<bool> {
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

    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(harness.state(), env);

    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(txs.len()),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };

    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, chain_id);
    executor.apply_pre_execution_changes().expect("pre-exec");

    let mut results = Vec::new();
    for tx in txs {
        let recovered = recover(tx);
        match executor.execute_transaction_without_commit(recovered) {
            Ok(result) => {
                let success = result.result.result.is_success();
                executor.commit_transaction(result).expect("commit");
                results.push(success);
            }
            Err(_) => results.push(false),
        }
    }

    let _ = executor.finish().expect("finish");
    results
}

#[test]
fn three_sequential_transfers_from_same_sender() {
    let mut s = ExecutorScaffold::new();
    let initial = U256::from(10u128 * ONE_ETH);
    fund_account(s.harness.state(), alice(), initial);

    let send = U256::from(ONE_ETH);
    let txs = (0..3u64)
        .map(|nonce| {
            sign_legacy(
                s.chain_id,
                nonce,
                ONE_GWEI,
                21_000,
                TxKind::Call(RECIPIENT),
                send,
                Bytes::new(),
                alice_key(),
            )
        })
        .collect();
    let results = run_multi_tx_block(&mut s.harness, s.base_fee, s.chain_id, txs);

    assert_eq!(results, vec![true, true, true]);
    assert_eq!(nonce_of(s.harness.state(), alice()), 3);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send * U256::from(3u64));
}

#[test]
fn three_transfers_from_three_distinct_senders() {
    let mut s = ExecutorScaffold::new();
    let initial = U256::from(5u128 * ONE_ETH);
    for sender in [alice(), bob(), charlie()] {
        fund_account(s.harness.state(), sender, initial);
    }

    let send = U256::from(ONE_ETH);
    let txs = vec![
        sign_legacy(s.chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
        sign_legacy(s.chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), bob_key()),
        sign_legacy(s.chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), charlie_key()),
    ];
    let results = run_multi_tx_block(&mut s.harness, s.base_fee, s.chain_id, txs);

    assert_eq!(results, vec![true, true, true]);
    assert_eq!(nonce_of(s.harness.state(), alice()), 1);
    assert_eq!(nonce_of(s.harness.state(), bob()), 1);
    assert_eq!(nonce_of(s.harness.state(), charlie()), 1);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send * U256::from(3u64));
}

#[test]
fn nonce_skip_rejects_second_tx() {
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let send = U256::from(ONE_ETH);
    let txs = vec![
        sign_legacy(s.chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
        sign_legacy(s.chain_id, 5, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
    ];
    let results = run_multi_tx_block(&mut s.harness, s.base_fee, s.chain_id, txs);

    assert_eq!(results[0], true);
    assert_eq!(results[1], false);
    assert_eq!(nonce_of(s.harness.state(), alice()), 1);
}

#[test]
fn revert_in_middle_does_not_block_subsequent_txs() {
    let revert_addr = Address::repeat_byte(0xCA);
    let revert_runtime: Vec<u8> = vec![0x60, 0x00, 0x60, 0x00, 0xFD];
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(s.harness.state(), revert_addr, revert_runtime, U256::ZERO);

    let send = U256::from(ONE_ETH);
    let txs = vec![
        sign_legacy(s.chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
        sign_legacy(s.chain_id, 1, ONE_GWEI, 100_000, TxKind::Call(revert_addr), U256::ZERO, Bytes::new(), alice_key()),
        sign_legacy(s.chain_id, 2, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
    ];
    let results = run_multi_tx_block(&mut s.harness, s.base_fee, s.chain_id, txs);

    assert_eq!(results[0], true);
    assert_eq!(results[1], false);
    assert_eq!(results[2], true);
    assert_eq!(nonce_of(s.harness.state(), alice()), 3);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send * U256::from(2u64));
}

#[test]
fn many_small_transfers_in_one_block() {
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(100u128 * ONE_ETH));

    const N: u64 = 20;
    let send = U256::from(ONE_ETH / 10);
    let txs = (0..N)
        .map(|nonce| {
            sign_legacy(
                s.chain_id,
                nonce,
                ONE_GWEI,
                21_000,
                TxKind::Call(RECIPIENT),
                send,
                Bytes::new(),
                alice_key(),
            )
        })
        .collect();
    let results = run_multi_tx_block(&mut s.harness, s.base_fee, s.chain_id, txs);

    assert!(results.iter().all(|&ok| ok));
    assert_eq!(nonce_of(s.harness.state(), alice()), N);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send * U256::from(N));
}

#[test]
fn balance_drains_correctly_when_value_plus_gas_exceeds_balance_mid_block() {
    let mut s = ExecutorScaffold::new();
    let initial = U256::from(2u128 * ONE_ETH);
    fund_account(s.harness.state(), alice(), initial);

    let send = U256::from(ONE_ETH);
    let txs = vec![
        sign_legacy(s.chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
        sign_legacy(s.chain_id, 1, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
        sign_legacy(s.chain_id, 2, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
    ];
    let results = run_multi_tx_block(&mut s.harness, s.base_fee, s.chain_id, txs);

    let successes: usize = results.iter().filter(|&&ok| ok).count();
    assert!(successes >= 1 && successes <= 2, "expected 1 or 2 successes, got {successes}");
    let final_recipient = balance_of(s.harness.state(), RECIPIENT);
    assert!(final_recipient <= send * U256::from(2u64));
}
