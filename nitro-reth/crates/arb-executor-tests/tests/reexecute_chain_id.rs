use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{B256, U256};
use arb_evm::config::ArbEvmConfig;
use arb_executor_tests::helpers::ExecutorScaffold;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

// The generic execution path (reth `re-execute`) calls the trait
// `create_executor`, which never receives a chain id, so `arb_ctx.chain_id`
// defaults to 0. Pre-execution must source it from the EVM cfg; otherwise the
// retryable auto-redeem tx hash is built with chain_id 0 and diverges from
// canonical. The producer path (`create_arb_executor` with a chain id) is
// unaffected because the fill is gated on a zero value.
#[test]
fn pre_execution_sources_chain_id_from_cfg_when_defaulted() {
    let mut s = ExecutorScaffold::new();
    let chain_id = s.chain_id;
    assert_ne!(chain_id, 0);

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
        tx_count_hint: Some(0),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    // A chain id of 0 mirrors the trait `create_executor` path used by re-execute.
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, 0);
    executor.arb_ctx.l2_block_number = 1;
    executor.apply_pre_execution_changes().unwrap();

    assert_eq!(executor.arb_ctx.chain_id, chain_id);
}
