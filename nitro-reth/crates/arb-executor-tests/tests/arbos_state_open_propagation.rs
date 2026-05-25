use std::{cell::Cell, sync::Arc};

use alloy_consensus::Header;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, B256, U256};
use arb_evm::config::ArbEvmConfig;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{
    database::{State, StateBuilder},
    primitives::hardfork::SpecId,
    Database,
};
use revm_database_interface::DBErrorMarker;

#[derive(Debug)]
struct FakeDbError(&'static str);

impl std::fmt::Display for FakeDbError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.0)
    }
}

impl std::error::Error for FakeDbError {}
impl DBErrorMarker for FakeDbError {}

#[derive(Debug)]
struct FailingDb {
    armed: Cell<bool>,
}

impl FailingDb {
    fn new() -> Self {
        Self {
            armed: Cell::new(false),
        }
    }
}

impl Database for FailingDb {
    type Error = FakeDbError;

    fn basic(&mut self, _address: Address) -> Result<Option<revm_state::AccountInfo>, Self::Error> {
        Ok(None)
    }

    fn code_by_hash(&mut self, _code_hash: B256) -> Result<revm_state::Bytecode, Self::Error> {
        Ok(revm_state::Bytecode::default())
    }

    fn storage(&mut self, _address: Address, _index: U256) -> Result<U256, Self::Error> {
        if self.armed.replace(false) {
            return Err(FakeDbError("synthetic db read failure"));
        }
        Ok(U256::ZERO)
    }

    fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
        Ok(B256::ZERO)
    }
}

fn provisional_header() -> Header {
    Header {
        timestamp: 1_700_000_000,
        base_fee_per_gas: Some(100_000_000),
        number: 1,
        gas_limit: 30_000_000,
        difficulty: U256::from(1),
        ..Default::default()
    }
}

#[test]
fn apply_pre_execution_propagates_db_failure() {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let header = provisional_header();
    let env: EvmEnv<SpecId> = cfg.evm_env(&header).expect("evm_env");

    let db = FailingDb::new();
    db.armed.set(true);
    let mut state: State<FailingDb> = StateBuilder::new()
        .with_database(db)
        .with_bundle_update()
        .build();

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

    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, 421614);

    let err = executor
        .apply_pre_execution_changes()
        .expect_err("primed db failure must surface as a block execution error");
    let message = err.to_string();
    assert!(
        message.contains("synthetic db read failure"),
        "expected propagated db error in: {message}"
    );
}
