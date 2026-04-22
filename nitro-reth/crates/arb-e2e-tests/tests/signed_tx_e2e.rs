use std::sync::Arc;

use alloy_consensus::{
    crypto::secp256k1::sign_message, transaction::SignerRecoverable, EthereumTxEnvelope,
    SignableTransaction, TxLegacy,
};
use alloy_evm::{block::BlockExecutorFactory, eth::EthBlockExecutionCtx, EvmFactory};
use alloy_primitives::{address, keccak256, Address, TxKind, B256, U256};
use arb_evm::config::ArbEvmConfig;
use arb_primitives::ArbTransactionSigned;
use arb_test_utils::{ArbosHarness, EmptyDb};
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{
    context::{BlockEnv, CfgEnv},
    primitives::hardfork::SpecId,
    state::AccountInfo,
};

const SECRET_KEY: [u8; 32] = [
    0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
    0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20,
];

fn derive_address(sk_bytes: [u8; 32]) -> Address {
    use k256::ecdsa::SigningKey;
    let sk = SigningKey::from_slice(&sk_bytes).expect("valid sk");
    let vk = *sk.verifying_key();
    let encoded = vk.to_encoded_point(false);
    let pubkey_bytes = &encoded.as_bytes()[1..];
    let hash = keccak256(pubkey_bytes);
    Address::from_slice(&hash[12..])
}

fn build_signed_legacy_tx(
    chain_id: u64,
    nonce: u64,
    gas_price: u128,
    gas_limit: u64,
    to: Address,
    value: U256,
    sk: [u8; 32],
) -> ArbTransactionSigned {
    let tx = TxLegacy {
        chain_id: Some(chain_id),
        nonce,
        gas_price,
        gas_limit,
        to: TxKind::Call(to),
        value,
        input: Default::default(),
    };
    let sig_hash = tx.signature_hash();
    let sig = sign_message(B256::from(sk), sig_hash).expect("sign");
    let signed = tx.into_signed(sig);
    let envelope = EthereumTxEnvelope::Legacy(signed);
    ArbTransactionSigned::from_envelope(envelope)
}

fn fund_account(state: &mut revm::database::State<EmptyDb>, addr: Address, balance: U256) {
    use revm::database::{states::account_status::AccountStatus, PlainAccount};
    let _ = state.load_cache_account(addr);
    if let Some(cached) = state.cache.accounts.get_mut(&addr) {
        cached.account = Some(PlainAccount {
            info: AccountInfo {
                balance,
                nonce: 0,
                code_hash: keccak256([]),
                code: None,
                account_id: None,
            },
            storage: Default::default(),
        });
        cached.status = AccountStatus::InMemoryChange;
    }
}

#[test]
fn sign_and_recover_legacy_tx_round_trip() {
    let chain_id = 421614;
    let derived = derive_address(SECRET_KEY);
    let tx = build_signed_legacy_tx(
        chain_id,
        0,
        1_000_000_000,
        21_000,
        address!("00000000000000000000000000000000DEADBEEF"),
        U256::from(1u64) * U256::from(10u64).pow(U256::from(18u64)),
        SECRET_KEY,
    );
    let recovered = tx.recover_signer().expect("recover");
    assert_eq!(recovered, derived);
}

#[test]
fn arb_executor_executes_signed_legacy_transfer() {
    use alloy_consensus::transaction::Recovered;
    use alloy_evm::block::BlockExecutor;

    let chain_id = 421614;
    let sender = derive_address(SECRET_KEY);
    let recipient = address!("11111111111111111111111111111111111111ff");
    let send_value = U256::from(1u64) * U256::from(10u64).pow(U256::from(18u64));
    let initial_balance = U256::from(10u64) * U256::from(10u64).pow(U256::from(18u64));

    let mut h = ArbosHarness::new()
        .with_arbos_version(30)
        .with_chain_id(chain_id)
        .initialize();
    fund_account(h.state(), sender, initial_balance);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: CfgEnv::default(),
        block_env: BlockEnv::default(),
    };
    env.cfg_env.chain_id = chain_id;
    env.cfg_env.disable_base_fee = true;
    env.block_env.timestamp = U256::from(1_700_000_000u64);
    env.block_env.basefee = 100_000_000;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(1u64);

    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(h.state(), env);

    let extra = vec![0u8; 32];
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(1),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: extra.into(),
    };

    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, chain_id);

    executor
        .apply_pre_execution_changes()
        .expect("pre-execution");

    let tx = build_signed_legacy_tx(
        chain_id,
        0,
        1_000_000_000,
        21_000,
        recipient,
        send_value,
        SECRET_KEY,
    );
    let recovered = Recovered::new_unchecked(tx, sender);

    let result = executor
        .execute_transaction_without_commit(recovered)
        .expect("tx execution");
    let _ = executor.commit_transaction(result).expect("commit");
    let _ = executor.finish().expect("finish");

    let recipient_balance = h
        .state()
        .cache
        .accounts
        .get(&recipient)
        .and_then(|a| a.account.as_ref())
        .map(|a| a.info.balance)
        .unwrap_or(U256::ZERO);
    assert_eq!(
        recipient_balance, send_value,
        "recipient must have received exactly the transferred value"
    );

    let sender_balance = h
        .state()
        .cache
        .accounts
        .get(&sender)
        .and_then(|a| a.account.as_ref())
        .map(|a| a.info.balance)
        .unwrap_or(U256::ZERO);
    let max_remaining = initial_balance - send_value;
    assert!(
        sender_balance <= max_remaining,
        "sender must have paid at least the value (got balance {sender_balance}, expected <= {max_remaining})"
    );
    let min_remaining =
        initial_balance - send_value - U256::from(21_000u64) * U256::from(1_000_000_000u64);
    assert!(
        sender_balance >= min_remaining,
        "sender balance went below value+gas floor (got {sender_balance}, min {min_remaining})"
    );

    let sender_nonce = h
        .state()
        .cache
        .accounts
        .get(&sender)
        .and_then(|a| a.account.as_ref())
        .map(|a| a.info.nonce)
        .unwrap_or(0);
    assert_eq!(sender_nonce, 1, "nonce must increment after tx execution");
}
