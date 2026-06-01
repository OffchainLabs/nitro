//! Installing the multi-gas inspector switches block execution to revm's
//! inspect path. This verifies that path is consensus-equivalent to the plain
//! path: the same contract-call block must produce identical post-state whether
//! or not the inspector is present.

use alloy_consensus::{
    crypto::secp256k1::sign_message, transaction::Recovered, EthereumTxEnvelope,
    SignableTransaction, TxLegacy,
};
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, keccak256, Address, TxKind, B256, U256};
use arb_evm::{
    config::ArbEvmConfig,
    multi_gas::{MultiGasInspector, MultiGasSink},
};
use arb_primitives::ArbTransactionSigned;
use arb_test_utils::{ArbosHarness, EmptyDb};
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{
    context::{BlockEnv, CfgEnv},
    database::{states::account_status::AccountStatus, PlainAccount, State},
    primitives::hardfork::SpecId,
    state::{AccountInfo, Bytecode},
};
use std::sync::Arc;

const SECRET_KEY: [u8; 32] = [
    0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
    0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20,
];
const CHAIN_ID: u64 = 421614;
const CONTRACT: Address = address!("00000000000000000000000000000000c0c0c0c0");

fn sender() -> Address {
    use k256::ecdsa::SigningKey;
    let sk = SigningKey::from_slice(&SECRET_KEY).unwrap();
    let encoded = sk.verifying_key().to_encoded_point(false);
    Address::from_slice(&keccak256(&encoded.as_bytes()[1..])[12..])
}

fn set_account(state: &mut State<EmptyDb>, addr: Address, info: AccountInfo) {
    let _ = state.load_cache_account(addr);
    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
        cached.account = Some(PlainAccount {
            info,
            storage: Default::default(),
        });
        cached.status = AccountStatus::InMemoryChange;
    }
}

fn call_contract_tx() -> Recovered<ArbTransactionSigned> {
    let tx = TxLegacy {
        chain_id: Some(CHAIN_ID),
        nonce: 0,
        gas_price: 1_000_000_000,
        gas_limit: 500_000,
        to: TxKind::Call(CONTRACT),
        value: U256::ZERO,
        input: Default::default(),
    };
    let sig = sign_message(B256::from(SECRET_KEY), tx.signature_hash()).unwrap();
    let envelope = EthereumTxEnvelope::Legacy(tx.into_signed(sig));
    Recovered::new_unchecked(ArbTransactionSigned::from_envelope(envelope), sender())
}

fn block_env() -> EvmEnv<SpecId> {
    let mut env = EvmEnv {
        cfg_env: CfgEnv::default(),
        block_env: BlockEnv::default(),
    };
    env.cfg_env.chain_id = CHAIN_ID;
    env.cfg_env.disable_base_fee = true;
    env.block_env.timestamp = U256::from(1_700_000_000u64);
    env.block_env.basefee = 100_000_000;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(1u64);
    env
}

fn exec_ctx() -> EthBlockExecutionCtx<'static> {
    EthBlockExecutionCtx {
        tx_count_hint: Some(1),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    }
}

/// Stores 1 at slot 0: PUSH1 1; PUSH1 0; SSTORE; STOP.
fn contract_code() -> Vec<u8> {
    vec![0x60, 0x01, 0x60, 0x00, 0x55, 0x00]
}

/// (sender balance, contract balance, contract slot 0).
type PostState = (U256, U256, U256);

fn read_post_state(h: &mut ArbosHarness) -> PostState {
    let read_balance = |h: &mut ArbosHarness, addr: Address| {
        h.state()
            .cache
            .accounts
            .get(&addr)
            .and_then(|a| a.account.as_ref())
            .map(|a| a.info.balance)
            .unwrap_or(U256::ZERO)
    };
    let slot0 = h
        .state()
        .cache
        .accounts
        .get(&CONTRACT)
        .and_then(|a| a.account.as_ref())
        .and_then(|a| a.storage.get(&U256::ZERO).copied())
        .unwrap_or(U256::ZERO);
    (read_balance(h, sender()), read_balance(h, CONTRACT), slot0)
}

fn harness_with_contract() -> ArbosHarness {
    let mut h = ArbosHarness::new()
        .with_arbos_version(30)
        .with_chain_id(CHAIN_ID)
        .initialize();
    let funded = AccountInfo {
        balance: U256::from(10u64).pow(U256::from(19u64)),
        ..Default::default()
    };
    set_account(h.state(), sender(), funded);
    let code = Bytecode::new_raw(contract_code().into());
    set_account(
        h.state(),
        CONTRACT,
        AccountInfo {
            code_hash: code.hash_slow(),
            code: Some(code),
            ..Default::default()
        },
    );
    h
}

fn run_without_inspector() -> PostState {
    let mut h = harness_with_contract();
    let cfg = ArbEvmConfig::new(Arc::new(ChainSpec::default()));
    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(h.state(), block_env());
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx(), CHAIN_ID);
    executor.apply_pre_execution_changes().unwrap();
    let result = executor
        .execute_transaction_without_commit(call_contract_tx())
        .unwrap();
    executor.commit_transaction(result).unwrap();
    executor.finish().unwrap();
    read_post_state(&mut h)
}

fn run_with_inspector() -> PostState {
    let mut h = harness_with_contract();
    let cfg = ArbEvmConfig::new(Arc::new(ChainSpec::default()));
    let sink = MultiGasSink::default();
    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm_with_inspector(
            h.state(),
            block_env(),
            MultiGasInspector::with_sink(sink.clone()),
        );
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx(), CHAIN_ID);
    executor.set_multi_gas_sink(sink);
    executor.apply_pre_execution_changes().unwrap();
    let result = executor
        .execute_transaction_without_commit(call_contract_tx())
        .unwrap();
    executor.commit_transaction(result).unwrap();
    executor.finish().unwrap();
    read_post_state(&mut h)
}

#[test]
fn inspect_path_is_consensus_equivalent() {
    let plain = run_without_inspector();
    let inspected = run_with_inspector();
    assert_eq!(
        plain.2,
        U256::from(1u64),
        "contract SSTORE must have executed"
    );
    assert_eq!(plain, inspected);
}
