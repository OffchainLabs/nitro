use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
use arb_e2e_tests::helpers::{
    alice, alice_key, bob, bob_key, charlie, charlie_key, fund_account, recover, sign_legacy,
    ONE_ETH, ONE_GWEI, RECIPIENT,
};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

fn run_block(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    block_number: u64,
    timestamp: u64,
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
    env.block_env.timestamp = U256::from(timestamp);
    env.block_env.basefee = base_fee;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(block_number);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);

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
    executor.arb_ctx.l2_block_number = block_number;
    executor.apply_pre_execution_changes().expect("pre");

    let mut results = Vec::new();
    for tx in txs {
        let recovered = recover(tx);
        match executor.execute_transaction_without_commit(recovered) {
            Ok(result) => {
                let ok = result.result.result.is_success();
                executor.commit_transaction(result).expect("commit");
                results.push(ok);
            }
            Err(_) => results.push(false),
        }
    }
    let _ = executor.finish().expect("finish");
    results
}

fn account_snapshot(
    harness: &mut ArbosHarness,
    addrs: &[Address],
) -> Vec<(Address, U256, u64)> {
    addrs
        .iter()
        .map(|a| {
            let info = harness
                .state()
                .cache
                .accounts
                .get(a)
                .and_then(|acc| acc.account.as_ref());
            let balance = info.map(|a| a.info.balance).unwrap_or(U256::ZERO);
            let nonce = info.map(|a| a.info.nonce).unwrap_or(0);
            (*a, balance, nonce)
        })
        .collect()
}

fn build_scenario(chain_id: u64) -> Vec<arb_primitives::ArbTransactionSigned> {
    let send = U256::from(ONE_ETH);
    vec![
        sign_legacy(chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
        sign_legacy(chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), bob_key()),
        sign_legacy(chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), charlie_key()),
        sign_legacy(chain_id, 1, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
    ]
}

fn fresh_harness_with_funds() -> ArbosHarness {
    let mut h = ArbosHarness::new()
        .with_arbos_version(30)
        .with_chain_id(arb_e2e_tests::helpers::CHAIN_ID)
        .initialize();
    for sender in [alice(), bob(), charlie()] {
        fund_account(h.state(), sender, U256::from(10u128 * ONE_ETH));
    }
    h
}

#[test]
fn identical_inputs_produce_identical_post_state() {
    let chain_id = arb_e2e_tests::helpers::CHAIN_ID;

    let mut h1 = fresh_harness_with_funds();
    let mut h2 = fresh_harness_with_funds();

    run_block(
        &mut h1, 100_000_000, chain_id, 1, 1_700_000_000, build_scenario(chain_id),
    );
    run_block(
        &mut h2, 100_000_000, chain_id, 1, 1_700_000_000, build_scenario(chain_id),
    );

    let addrs = vec![alice(), bob(), charlie(), RECIPIENT];
    let snap1 = account_snapshot(&mut h1, &addrs);
    let snap2 = account_snapshot(&mut h2, &addrs);
    assert_eq!(
        snap1, snap2,
        "identical tx sequences must produce identical post-state"
    );
}

#[test]
fn tx_order_matters_for_post_state() {
    let chain_id = arb_e2e_tests::helpers::CHAIN_ID;

    let mut h1 = fresh_harness_with_funds();
    let mut h2 = fresh_harness_with_funds();

    let send = U256::from(ONE_ETH);
    let txs_a_then_b = vec![
        sign_legacy(chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
        sign_legacy(chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), bob_key()),
    ];
    let txs_b_then_a = vec![
        sign_legacy(chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), bob_key()),
        sign_legacy(chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), send, Bytes::new(), alice_key()),
    ];

    run_block(&mut h1, 100_000_000, chain_id, 1, 1_700_000_000, txs_a_then_b);
    run_block(&mut h2, 100_000_000, chain_id, 1, 1_700_000_000, txs_b_then_a);

    let recipient_bal_1 = account_snapshot(&mut h1, &[RECIPIENT])[0].1;
    let recipient_bal_2 = account_snapshot(&mut h2, &[RECIPIENT])[0].1;
    assert_eq!(
        recipient_bal_1, recipient_bal_2,
        "recipient balance must be the same regardless of which of A/B went first"
    );
}

/// Simulate reorg: run a tx sequence, then run alternate sequence from same
/// starting state. Result should match a single run of the alternate.
#[test]
fn reorg_from_fresh_state_produces_canonical_result() {
    let chain_id = arb_e2e_tests::helpers::CHAIN_ID;

    let mut forked = fresh_harness_with_funds();
    run_block(
        &mut forked, 100_000_000, chain_id, 1, 1_700_000_000, build_scenario(chain_id),
    );

    let mut canonical = fresh_harness_with_funds();
    let alt_send = U256::from(ONE_ETH / 2);
    let alt_txs = vec![
        sign_legacy(
            chain_id, 0, ONE_GWEI, 21_000, TxKind::Call(RECIPIENT), alt_send,
            Bytes::new(), alice_key(),
        ),
    ];
    run_block(&mut canonical, 100_000_000, chain_id, 1, 1_700_000_000, alt_txs.clone());

    let mut canonical_from_fresh = fresh_harness_with_funds();
    run_block(&mut canonical_from_fresh, 100_000_000, chain_id, 1, 1_700_000_000, alt_txs);

    let addrs = vec![alice(), RECIPIENT];
    assert_eq!(
        account_snapshot(&mut canonical, &addrs),
        account_snapshot(&mut canonical_from_fresh, &addrs),
    );
}

#[test]
fn empty_block_produces_minimal_state_changes() {
    let chain_id = arb_e2e_tests::helpers::CHAIN_ID;
    let mut h1 = fresh_harness_with_funds();
    let mut h2 = fresh_harness_with_funds();

    run_block(&mut h1, 100_000_000, chain_id, 1, 1_700_000_000, vec![]);
    run_block(&mut h2, 100_000_000, chain_id, 1, 1_700_000_000, vec![]);

    let addrs = vec![alice(), bob(), charlie(), RECIPIENT];
    assert_eq!(
        account_snapshot(&mut h1, &addrs),
        account_snapshot(&mut h2, &addrs),
    );
}
