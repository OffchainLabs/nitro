use std::sync::Arc;

use alloy_consensus::Header;
use alloy_evm::{block::BlockExecutorFactory, eth::EthBlockExecutionCtx, EvmFactory};
use alloy_primitives::{Address, B64, B256, U256};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::{ArbosHarness, EmptyDb};
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{
    context::{BlockEnv, CfgEnv},
    database::{State, StateBuilder},
    primitives::hardfork::SpecId,
};

fn fresh_state() -> State<EmptyDb> {
    StateBuilder::new()
        .with_database(EmptyDb)
        .with_bundle_update()
        .build()
}

fn provisional_header() -> Header {
    Header {
        parent_hash: B256::ZERO,
        ommers_hash: alloy_consensus::EMPTY_OMMER_ROOT_HASH,
        beneficiary: Address::ZERO,
        state_root: B256::ZERO,
        transactions_root: B256::ZERO,
        receipts_root: B256::ZERO,
        withdrawals_root: None,
        logs_bloom: Default::default(),
        timestamp: 1_700_000_000,
        mix_hash: B256::ZERO,
        nonce: B64::ZERO,
        base_fee_per_gas: Some(100_000_000),
        number: 1,
        gas_limit: 30_000_000,
        difficulty: U256::from(1),
        gas_used: 0,
        extra_data: Default::default(),
        parent_beacon_block_root: None,
        blob_gas_used: None,
        excess_blob_gas: None,
        requests_hash: None,
    }
}

#[test]
fn arb_evm_config_constructs_with_real_chain_spec() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let _ = cfg.block_executor_factory();
}

#[test]
fn arb_evm_config_builds_evm_env_from_header() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let header = provisional_header();
    let env: EvmEnv<SpecId> = cfg.evm_env(&header).expect("evm_env");
    assert_eq!(env.block_env.timestamp, U256::from(header.timestamp));
}

#[test]
fn evm_factory_creates_evm_against_state() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let header = provisional_header();
    let env: EvmEnv<SpecId> = cfg.evm_env(&header).expect("evm_env");
    let mut state = fresh_state();
    let _evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(&mut state, env);
}

#[test]
fn arb_executor_constructs_via_factory() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let header = provisional_header();
    let env: EvmEnv<SpecId> = cfg.evm_env(&header).expect("evm_env");
    let mut state = fresh_state();
    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(&mut state, env);
    let extra = vec![0u8; 32];
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(0),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: extra.into(),
    };
    let _executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, 421614);
}

#[test]
fn harness_state_can_back_a_constructed_evm() {
    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: CfgEnv::default(),
        block_env: BlockEnv::default(),
    };
    env.cfg_env.chain_id = h.chain_id();
    env.block_env.timestamp = U256::from(1_700_000_000u64);

    let _evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(h.state(), env);
}

#[test]
fn arb_executor_apply_pre_execution_on_harness_state() {
    use alloy_evm::block::BlockExecutor;

    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
    let chain_id = h.chain_id();
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: CfgEnv::default(),
        block_env: BlockEnv::default(),
    };
    env.cfg_env.chain_id = chain_id;
    env.block_env.timestamp = U256::from(1_700_000_000u64);
    env.block_env.basefee = 100_000_000;
    env.block_env.gas_limit = 30_000_000;

    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(h.state(), env);

    let extra = vec![0u8; 32];
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(0),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: extra.into(),
    };

    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, chain_id);

    let _ = executor.apply_pre_execution_changes();
}
