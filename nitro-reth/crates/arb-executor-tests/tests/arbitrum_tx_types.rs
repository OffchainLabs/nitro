use std::sync::Arc;

use alloy_consensus::transaction::Recovered;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, Address, Bytes, Signature, B256, U256};
use arb_alloy_consensus::tx::{
    ArbContractTx, ArbDepositTx, ArbInternalTx, ArbSubmitRetryableTx, ArbUnsignedTx,
};
use arb_executor_tests::helpers::{
    balance_of, fund_account, nonce_of, ExecutorScaffold, ONE_ETH, ONE_GWEI, RECIPIENT,
};
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_test_utils::ArbosHarness;
use arbos::internal_tx::encode_start_block;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

const POSTER: Address = address!("a4b0000000000000000000000000000073657175");

fn zero_sig() -> Signature {
    Signature::new(U256::ZERO, U256::ZERO, false)
}

fn run_arb_tx(
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
        .map_err(|e| format!("execute: {e}"))?;
    let success = result.result.result.is_success();
    executor
        .commit_transaction(result)
        .map_err(|e| format!("commit: {e}"))?;
    let _ = executor.finish().map_err(|e| format!("finish: {e}"))?;
    Ok(success)
}

#[test]
fn arbitrum_deposit_tx_mints_to_recipient() {
    let mut s = ExecutorScaffold::new();
    let to = address!("00000000000000000000000000000000DEEDFA11");
    let value = U256::from(3u128) * U256::from(ONE_ETH);

    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Deposit(ArbDepositTx {
            chain_id: U256::from(s.chain_id),
            l1_request_id: B256::repeat_byte(0x01),
            from: POSTER,
            to,
            value,
        }),
        zero_sig(),
    );
    let success = run_arb_tx(&mut s.harness, s.base_fee, s.chain_id, tx, POSTER).expect("execute");
    assert!(success);
    assert_eq!(balance_of(s.harness.state(), to), value);
}

#[test]
fn arbitrum_unsigned_tx_executes_with_poster_as_sender() {
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), POSTER, U256::from(10u128 * ONE_ETH));

    let send_value = U256::from(ONE_ETH);
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Unsigned(ArbUnsignedTx {
            chain_id: U256::from(s.chain_id),
            from: POSTER,
            nonce: 0,
            gas_fee_cap: U256::from(ONE_GWEI),
            gas: 21_000,
            to: Some(RECIPIENT),
            value: send_value,
            data: Bytes::new(),
        }),
        zero_sig(),
    );

    let success = run_arb_tx(&mut s.harness, s.base_fee, s.chain_id, tx, POSTER).expect("execute");
    assert!(success);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send_value);
    assert_eq!(nonce_of(s.harness.state(), POSTER), 1);
}

#[test]
fn arbitrum_contract_tx_executes_from_l1_contract() {
    let mut s = ExecutorScaffold::new();
    let l1_contract: Address = address!("c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0c0");
    fund_account(s.harness.state(), l1_contract, U256::from(10u128 * ONE_ETH));

    let send_value = U256::from(ONE_ETH);
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Contract(ArbContractTx {
            chain_id: U256::from(s.chain_id),
            request_id: B256::repeat_byte(0x42),
            from: l1_contract,
            gas_fee_cap: U256::from(ONE_GWEI),
            gas: 100_000,
            to: Some(RECIPIENT),
            value: send_value,
            data: Bytes::new(),
        }),
        zero_sig(),
    );

    let success =
        run_arb_tx(&mut s.harness, s.base_fee, s.chain_id, tx, l1_contract).expect("execute");
    assert!(success);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send_value);
}

#[test]
fn arbitrum_internal_tx_start_block_executes() {
    let mut s = ExecutorScaffold::new();

    let data = encode_start_block(U256::from(ONE_GWEI), 100, 1, 12);
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(s.chain_id),
            data: data.into(),
        }),
        zero_sig(),
    );
    let arbos_addr = address!("00000000000000000000000000000000000A4B05");
    let success =
        run_arb_tx(&mut s.harness, s.base_fee, s.chain_id, tx, arbos_addr).expect("execute");
    assert!(success);
}

#[test]
fn arbitrum_submit_retryable_creates_ticket_in_state() {
    let mut s = ExecutorScaffold::new();
    let submitter: Address = address!("00000000000000000000000000000000Bee71337");
    fund_account(s.harness.state(), submitter, U256::from(100u128 * ONE_ETH));
    let beneficiary: Address = address!("00000000000000000000000000000000000B0B00");
    let retry_to: Address = address!("11111111111111111111111111111111111111ff");

    let deposit_value = U256::from(10u128 * ONE_ETH);
    let retry_value = U256::from(ONE_ETH);
    let max_submission_fee = U256::from(ONE_ETH);

    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
            chain_id: U256::from(s.chain_id),
            request_id: B256::repeat_byte(0x42),
            from: submitter,
            l1_base_fee: U256::from(ONE_GWEI),
            deposit_value,
            gas_fee_cap: U256::from(ONE_GWEI),
            gas: 0,
            retry_to: Some(retry_to),
            retry_value,
            beneficiary,
            max_submission_fee,
            fee_refund_addr: beneficiary,
            retry_data: Bytes::new(),
        }),
        zero_sig(),
    );
    use alloy_eips::eip2718::Encodable2718;
    let ticket_id: B256 = tx.trie_hash();
    let success =
        run_arb_tx(&mut s.harness, s.base_fee, s.chain_id, tx, submitter).expect("execute");
    assert!(success);

    let rs = s.harness.retryable_state();
    let opened = rs.open_retryable(ticket_id, 1_700_000_000).unwrap();
    assert!(opened.is_some(), "retryable must exist after submit");
}

#[test]
fn deposit_zero_value_still_succeeds() {
    let mut s = ExecutorScaffold::new();
    let to = address!("00000000000000000000000000000000c0ffee00");
    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Deposit(ArbDepositTx {
            chain_id: U256::from(s.chain_id),
            l1_request_id: B256::repeat_byte(0x05),
            from: POSTER,
            to,
            value: U256::ZERO,
        }),
        zero_sig(),
    );
    let success = run_arb_tx(&mut s.harness, s.base_fee, s.chain_id, tx, POSTER).expect("execute");
    assert!(success);
}

#[test]
fn unsigned_tx_with_calldata_to_contract() {
    use arb_executor_tests::helpers::deploy_contract;
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), POSTER, U256::from(10u128 * ONE_ETH));

    let target: Address = address!("00000000000000000000000000000000A11CE000");
    let runtime = vec![0x60, 0x00, 0x60, 0x00, 0xF3];
    deploy_contract(s.harness.state(), target, runtime, U256::ZERO);

    let tx = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Unsigned(ArbUnsignedTx {
            chain_id: U256::from(s.chain_id),
            from: POSTER,
            nonce: 0,
            gas_fee_cap: U256::from(ONE_GWEI),
            gas: 100_000,
            to: Some(target),
            value: U256::from(100u64),
            data: Bytes::from(vec![0xDE, 0xAD, 0xBE, 0xEF]),
        }),
        zero_sig(),
    );
    let success = run_arb_tx(&mut s.harness, s.base_fee, s.chain_id, tx, POSTER).expect("execute");
    assert!(success);
    assert_eq!(balance_of(s.harness.state(), target), U256::from(100u64));
}
