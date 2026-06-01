use std::sync::Arc;

use alloy_consensus::transaction::Recovered;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, Signature, B256, U256};
use arb_alloy_consensus::tx::ArbDepositTx;
use arb_evm::config::ArbEvmConfig;
use arb_executor_tests::helpers::ExecutorScaffold;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{database::states::bundle_state::BundleRetention, primitives::hardfork::SpecId};

fn zero_sig() -> Signature {
    Signature::new(U256::ZERO, U256::ZERO, false)
}

// Regression for the offline-execute deposit persistence bug. The
// existing `arbitrum_deposit_tx_mints_to_recipient` test checks the
// State cache via `balance_of`, which doesn't catch the failure mode:
// the credit lands in the cache (visible same-block) but never in the
// `BundleState`, so the recipient reads 0 in the next block. This test
// asserts the credit on the *bundle* directly, which is what reth
// writes to plain state.
#[test]
fn arbitrum_deposit_persists_fresh_recipient_to_bundle() {
    let mut s = ExecutorScaffold::new();
    let chain_id = s.chain_id;
    let from = address!("00000000000000000000000000000000d05dd05d");
    let recipient = address!("00000000000000000000000000000000beefbeef");
    let value = U256::from(1_000_000_000_000_000_000u128);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = chain_id;
    env.cfg_env.disable_base_fee = true;
    env.block_env.timestamp = U256::from(1_700_000_000u64);
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(1u64);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));

    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(s.harness.state(), env);
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
    executor.arb_ctx.l2_block_number = 1;
    executor.apply_pre_execution_changes().unwrap();
    executor.arb_ctx.basefee = U256::ZERO;

    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Deposit(ArbDepositTx {
            chain_id: U256::from(chain_id),
            l1_request_id: B256::repeat_byte(0x01),
            from,
            to: recipient,
            value,
        }),
        zero_sig(),
    );
    let recovered: Recovered<ArbTransactionSigned> = Recovered::new_unchecked(tx, from);

    let result = executor
        .execute_transaction_without_commit(recovered)
        .expect("execute");
    let _ = executor.commit_transaction(result).expect("commit");
    let _ = executor.finish().expect("finish");

    let state = s.harness.state();
    state.merge_transitions(BundleRetention::Reverts);
    let bundled = state
        .bundle_state
        .state
        .get(&recipient)
        .and_then(|a| a.info.as_ref())
        .map(|i| i.balance);
    assert_eq!(bundled, Some(value));
}
