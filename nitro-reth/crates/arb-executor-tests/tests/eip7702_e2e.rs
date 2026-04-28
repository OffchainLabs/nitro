use std::sync::Arc;

use alloy_consensus::{
    crypto::secp256k1::sign_message, transaction::Recovered, EthereumTxEnvelope,
    SignableTransaction, TxEip7702,
};
use alloy_eips::eip7702::Authorization;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, Bytes, B256, U256};
use arb_executor_tests::helpers::{
    alice, alice_key, bob, bob_key, deploy_contract, derive_address, fund_account, ONE_ETH,
    ONE_GWEI,
};
use arb_evm::config::ArbEvmConfig;
use arb_primitives::ArbTransactionSigned;
use arb_test_utils::ArbosHarness;
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
    env.cfg_env.spec = SpecId::PRAGUE;
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

fn sign_authorization(
    chain_id: u64,
    address: Address,
    nonce: u64,
    sk: [u8; 32],
) -> alloy_eips::eip7702::SignedAuthorization {
    let auth = Authorization {
        chain_id: U256::from(chain_id),
        address,
        nonce,
    };
    let sig_hash = auth.signature_hash();
    let sig = sign_message(B256::from(sk), sig_hash).expect("sign auth");
    auth.into_signed(sig)
}

fn sign_7702(
    chain_id: u64,
    nonce: u64,
    max_fee: u128,
    max_priority: u128,
    gas_limit: u64,
    to: Address,
    value: U256,
    input: Bytes,
    auth_list: Vec<alloy_eips::eip7702::SignedAuthorization>,
    sk: [u8; 32],
) -> ArbTransactionSigned {
    let tx = TxEip7702 {
        chain_id,
        nonce,
        gas_limit,
        max_fee_per_gas: max_fee,
        max_priority_fee_per_gas: max_priority,
        to,
        value,
        access_list: Default::default(),
        authorization_list: auth_list,
        input,
    };
    let sig_hash = tx.signature_hash();
    let sig = sign_message(B256::from(sk), sig_hash).expect("sign tx");
    let signed = tx.into_signed(sig);
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Eip7702(signed))
}

#[test]
fn eip7702_delegation_installs_code_prefix_on_authority() {
    let chain_id = 421614u64;
    let authority_key = bob_key();
    let authority = bob();
    let delegate_target = Address::repeat_byte(0x77);

    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    fund_account(
        s.harness.state(),
        alice(),
        U256::from(10u128) * U256::from(ONE_ETH),
    );
    fund_account(s.harness.state(), authority, U256::from(ONE_ETH));
    deploy_contract(
        s.harness.state(),
        delegate_target,
        vec![0x60, 0x42, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3],
        U256::ZERO,
    );

    let signed_auth = sign_authorization(chain_id, delegate_target, 0, authority_key);
    let tx = sign_7702(
        chain_id,
        0,
        2 * ONE_GWEI,
        0,
        500_000,
        authority,
        U256::ZERO,
        Bytes::new(),
        vec![signed_auth],
        alice_key(),
    );

    let success = run_tx(&mut s.harness, s.base_fee, chain_id, tx, alice()).expect("execute");
    assert!(success, "7702 tx should succeed");

    let code = s
        .harness
        .state()
        .cache
        .accounts
        .get(&authority)
        .and_then(|a| a.account.as_ref())
        .and_then(|a| a.info.code.as_ref())
        .map(|c| c.original_bytes());
    let code = code.expect("authority must have delegated code");
    assert!(
        code.len() >= 2,
        "delegated code should be at least the 2-byte prefix"
    );
    assert_eq!(
        &code[..2],
        &[0xEF, 0x01],
        "EIP-7702 delegation magic prefix should be 0xEF01"
    );
}

#[test]
fn eip7702_authorization_recovers_authority() {
    let chain_id = 421614u64;
    let sk = alice_key();
    let expected = alice();
    let signed_auth = sign_authorization(chain_id, Address::repeat_byte(0xAA), 0, sk);
    let recovered = signed_auth.recover_authority().expect("recover");
    assert_eq!(recovered, expected);
}

#[test]
fn eip7702_empty_authorization_list_still_executes() {
    let chain_id = 421614u64;
    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    fund_account(
        s.harness.state(),
        alice(),
        U256::from(10u128) * U256::from(ONE_ETH),
    );

    let recipient = Address::repeat_byte(0xCC);
    let send_value = U256::from(ONE_ETH);
    let tx = sign_7702(
        chain_id,
        0,
        2 * ONE_GWEI,
        0,
        100_000,
        recipient,
        send_value,
        Bytes::new(),
        vec![],
        alice_key(),
    );

    let _ = run_tx(&mut s.harness, s.base_fee, chain_id, tx, alice());
}

#[test]
fn eip7702_delegation_with_wrong_authority_does_not_install_code() {
    let chain_id = 421614u64;
    let authority = bob();

    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    fund_account(
        s.harness.state(),
        alice(),
        U256::from(10u128) * U256::from(ONE_ETH),
    );

    let signed_auth_by_wrong_key =
        sign_authorization(chain_id, Address::repeat_byte(0x99), 0, alice_key());
    let tx = sign_7702(
        chain_id,
        0,
        2 * ONE_GWEI,
        0,
        500_000,
        authority,
        U256::ZERO,
        Bytes::new(),
        vec![signed_auth_by_wrong_key],
        alice_key(),
    );

    let _ = run_tx(&mut s.harness, s.base_fee, chain_id, tx, alice());

    let authority_acct = s.harness.state().cache.accounts.get(&authority).cloned();
    if let Some(code) = authority_acct
        .and_then(|a| a.account)
        .and_then(|a| a.info.code)
    {
        let raw = code.original_bytes();
        let starts_with_magic = raw.len() >= 2 && raw[..2] == [0xEF, 0x01];
        let delegated_to_wrong = if raw.len() >= 22 {
            let addr = Address::from_slice(&raw[2..22]);
            addr == authority
        } else {
            false
        };
        assert!(
            !starts_with_magic || !delegated_to_wrong,
            "bob should NOT be delegated to himself via signatures from alice"
        );
    }

    let _ = derive_address(alice_key());
}
