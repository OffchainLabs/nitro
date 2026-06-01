//! Replays Sepolia block 269,589,702 (v60) from captured canonical pre-state:
//! the StartBlock InternalTx followed by the user EIP-1559 transaction, and
//! asserts the sender's net balance matches canonical. The block charges the
//! sender at the header base fee (20,026,000) but the canonical net debit is
//! at the floor (20,000,000) — the difference is the v60 multi-gas refund.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::{collections::BTreeMap, sync::Arc};

use alloy_consensus::{transaction::Recovered, EthereumTxEnvelope, SignableTransaction, TxEip1559};
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, b256, hex, Address, Bytes, Signature, TxKind, B256, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::config::ArbEvmConfig;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arb_storage::{set_account_code, set_account_nonce, write_storage_at};
use arb_test_utils::{ArbosHarness, EmptyDb};
use arbos::internal_tx::encode_start_block;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{database::State, primitives::hardfork::SpecId};
use serde::Deserialize;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const BLOCK_NUMBER: u64 = 269_589_702;
const BLOCK_TIMESTAMP: u64 = 0x6a0c9712;
const HEADER_BASE_FEE: u128 = 0x1319290; // 20,026,000
const L1_BLOCK_NUMBER: u64 = 0xa606cd;
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");

const SENDER: Address = address!("bbbbedc42dc53842141be8f70df9efe4d08538a4");
const TARGET: Address = address!("2b6897a9e0d78c59a35564d93fd0a2ae745d0654");
const TX_NONCE: u64 = 3212;
const TX_GAS_LIMIT: u64 = 304_331;
const TX_MAX_FEE: u128 = 24_000_000;
const TX_MAX_PRIO: u128 = 0;

const CANON_SENDER_POST: U256 = U256::from_limbs([0x024fc7fb330f38c0, 0, 0, 0]);

const ARBOS_ADDRESS: Address = address!("00000000000000000000000000000000000A4B05");

const TX_INPUT_HEX: &str =
    include_str!("../../arb-spec-tests/fixtures/regression/sepolia_269589702/tx1_input.hex");
const PRESTATE_JSON: &str = include_str!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269589702/block_start_prestate.json"
);

#[derive(Debug, Deserialize)]
struct AccountSnapshot {
    #[serde(default)]
    balance: Option<String>,
    #[serde(default)]
    nonce: Option<u64>,
    #[serde(default)]
    code: Option<String>,
    #[serde(default)]
    storage: BTreeMap<String, String>,
}

fn hu(s: &str) -> U256 {
    let s = s.strip_prefix("0x").unwrap_or(s);
    if s.is_empty() {
        U256::ZERO
    } else {
        U256::from_str_radix(s, 16).unwrap()
    }
}
fn hb(s: &str) -> Vec<u8> {
    hex::decode(s.trim().trim_start_matches("0x")).unwrap()
}
fn addr(s: &str) -> Address {
    Address::from_slice(&hb(s))
}

fn seed_prestate(state: &mut State<EmptyDb>, snap: &BTreeMap<String, AccountSnapshot>) {
    use revm::database::states::bundle_state::BundleRetention;
    for (a, acct) in snap {
        let ad = addr(a);
        if let Some(b) = acct.balance.as_deref() {
            let v = hu(b);
            if !v.is_zero() {
                arb_executor_tests::helpers::fund_account(state, ad, v);
            }
        }
        if let Some(c) = acct.code.as_deref() {
            let by = hb(c);
            if !by.is_empty() {
                set_account_code(state, ad, Bytes::from(by));
            }
        }
        if let Some(n) = acct.nonce {
            if n > 0 {
                set_account_nonce(state, ad, n);
            }
        }
        for (slot, val) in &acct.storage {
            write_storage_at(state, ad, hu(slot), hu(val)).unwrap();
        }
    }
    state.merge_transitions(BundleRetention::Reverts);
}

fn read_balance(state: &mut State<EmptyDb>, a: Address) -> U256 {
    state
        .cache
        .accounts
        .get(&a)
        .and_then(|c| c.account.as_ref())
        .map(|x| x.info.balance)
        .unwrap_or(U256::ZERO)
}

fn build_user_tx() -> ArbTransactionSigned {
    let tx = TxEip1559 {
        chain_id: CHAIN_ID,
        nonce: TX_NONCE,
        gas_limit: TX_GAS_LIMIT,
        max_fee_per_gas: TX_MAX_FEE,
        max_priority_fee_per_gas: TX_MAX_PRIO,
        to: TxKind::Call(TARGET),
        value: U256::ZERO,
        access_list: Default::default(),
        input: Bytes::from(hb(TX_INPUT_HEX)),
    };
    let sig = Signature::new(
        U256::from_be_bytes(
            b256!("2e4c557af1d19b6306ad4bf8ac0c344a65a5a52d1acf8b81494a769866d1c895").0,
        ),
        U256::from_be_bytes(
            b256!("222310565dcdb5c49bc71ea069cc9f867e0d8762e428409391718b7e0ff0ff84").0,
        ),
        true,
    );
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Eip1559(tx.into_signed(sig)))
}

#[test]
fn sepolia_269589702_sender_net_charge_matches_canonical() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    let prestate: BTreeMap<String, AccountSnapshot> = serde_json::from_str(PRESTATE_JSON).unwrap();
    seed_prestate(harness.state(), &prestate);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = CHAIN_ID;
    env.cfg_env.disable_base_fee = true;
    env.cfg_env.tx_gas_limit_cap = Some(u64::MAX);
    env.block_env.timestamp = U256::from(BLOCK_TIMESTAMP);
    env.block_env.basefee = HEADER_BASE_FEE as u64;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);
    env.block_env.beneficiary = SEQUENCER;

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(
        ARBOS_VERSION,
        BLOCK_TIMESTAMP,
        BLOCK_NUMBER,
        L1_BLOCK_NUMBER,
        false,
    );
    evm_factory.stage_ctx(Arc::new(arb_context::ArbPrecompileCtx::with_block(
        Arc::new(block_ctx),
    )));
    let evm = evm_factory.create_evm(harness.state(), env);
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(2),
        parent_hash: b256!("8762beb83df569953b74bd02978932f761ec204748c60544d2cb20641ea369f5"),
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor.arb_ctx.block_timestamp = BLOCK_TIMESTAMP;
    executor.arb_ctx.basefee = U256::from(HEADER_BASE_FEE);
    executor.arb_ctx.l2_block_number = BLOCK_NUMBER;
    executor.arb_ctx.l1_block_number = L1_BLOCK_NUMBER;
    executor.apply_pre_execution_changes().expect("pre-exec");

    // StartBlock InternalTx (drains base_fee_wei to the next block's value).
    let sb = ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(CHAIN_ID),
            data: encode_start_block(U256::ZERO, L1_BLOCK_NUMBER, BLOCK_NUMBER, 1).into(),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    let r = executor
        .execute_transaction_without_commit(Recovered::new_unchecked(sb, ARBOS_ADDRESS))
        .expect("startblock");
    executor.commit_transaction(r).expect("commit sb");

    // User tx.
    let tx = build_user_tx();
    let recovered = arb_executor_tests::helpers::recover(tx);
    assert_eq!(recovered.signer(), SENDER, "sender recovery");
    let r = executor
        .execute_transaction_without_commit(recovered)
        .expect("user tx");
    executor.commit_transaction(r).expect("commit user");
    let _ = executor.finish().expect("finish");

    let got = read_balance(harness.state(), SENDER);
    assert_eq!(
        got,
        CANON_SENDER_POST,
        "sender net balance must match canonical (got {got:x}, want {CANON_SENDER_POST:x}; \
         delta {} wei vs canonical)",
        CANON_SENDER_POST.abs_diff(got),
    );
}
