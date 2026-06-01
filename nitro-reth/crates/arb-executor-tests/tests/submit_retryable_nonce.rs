//! Isolates whether ArbitrumSubmitRetryableTx increments the sender nonce.
//! Canonical (Nitro) does NOT: only the auto-redeem RetryTx bumps it.

use std::sync::Arc;

use alloy_consensus::transaction::Recovered;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, Address, Bytes, Signature, B256, U256};
use arb_alloy_consensus::tx::ArbSubmitRetryableTx;
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_test_utils::{ArbosHarness, EmptyDb};
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{
    database::{states::account_status::AccountStatus, PlainAccount, State},
    primitives::hardfork::SpecId,
    state::AccountInfo,
};

const ALIAS: Address = address!("526adbf6af5173dc7faa48ae1e9fe7a145dcf37a");
const BENEFICIARY: Address = address!("4159dbf6af5173dc7faa48ae1e9fe7a145dce269");
const RETRY_TO: Address = address!("b6534cb24b925b58dfd811a0090f24c7ad52ca78");
const CHAIN_ID: u64 = 421_614;

fn seed_nonce(state: &mut State<EmptyDb>, addr: Address, nonce: u64, balance: U256) {
    let _ = state.load_cache_account(addr);
    if let Some(c) = state.cache.accounts.get_mut(&addr) {
        c.account = Some(PlainAccount {
            info: AccountInfo {
                balance,
                nonce,
                code_hash: alloy_primitives::keccak256([]),
                code: None,
                account_id: None,
            },
            storage: Default::default(),
        });
        c.status = AccountStatus::InMemoryChange;
    }
}

fn nonce_of(state: &mut State<EmptyDb>, addr: Address) -> u64 {
    state
        .cache
        .accounts
        .get(&addr)
        .and_then(|a| a.account.as_ref())
        .map(|a| a.info.nonce)
        .unwrap_or(0)
}

#[test]
fn submit_retryable_does_not_increment_sender_nonce() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(51)
        .with_chain_id(CHAIN_ID)
        .initialize();

    seed_nonce(
        harness.state(),
        ALIAS,
        5,
        U256::from(10u128).pow(U256::from(19u64)),
    );

    let submit = ArbSubmitRetryableTx {
        chain_id: U256::from(CHAIN_ID),
        request_id: B256::from(U256::from(0x1db544u64).to_be_bytes()),
        from: ALIAS,
        l1_base_fee: U256::from(0x10u64),
        deposit_value: U256::from_str_radix("de0cda8a0a7ce00", 16).unwrap(),
        gas_fee_cap: U256::from(0x7270e00u64),
        gas: 0x194a2,
        retry_to: Some(RETRY_TO),
        retry_value: U256::from_str_radix("de0c25a77d5de00", 16).unwrap(),
        beneficiary: BENEFICIARY,
        max_submission_fee: U256::from(0x31400u64),
        fee_refund_addr: BENEFICIARY,
        retry_data: Bytes::new(),
    };
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::SubmitRetryable(submit),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );

    let before = nonce_of(harness.state(), ALIAS);
    run_tx(&mut harness, 20_000_000, CHAIN_ID, tx, ALIAS);
    let after = nonce_of(harness.state(), ALIAS);

    eprintln!("alias nonce: before={before} after={after} (canonical: unchanged by submit)");
    assert_eq!(
        after, before,
        "submit-retryable must NOT increment the sender nonce (Nitro only bumps it on the redeem)"
    );
}

fn run_tx(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    tx: ArbTransactionSigned,
    sender: Address,
) {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);
    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = chain_id;
    env.cfg_env.disable_base_fee = true;
    env.block_env.timestamp = U256::from(1_774_681_452u64);
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
    executor.arb_ctx.block_timestamp = 1_774_681_452;
    executor.arb_ctx.basefee = U256::from(base_fee);
    executor.arb_ctx.l2_block_number = 1;
    executor.apply_pre_execution_changes().expect("pre");
    let recovered = Recovered::new_unchecked(tx, sender);
    let result = executor
        .execute_transaction_without_commit(recovered)
        .expect("exec");
    executor.commit_transaction(result).expect("commit");
    let _ = executor.finish().expect("finish");
}
