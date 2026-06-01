//! Under v60 the multi-gas inspector drives the per-transaction refund, so any
//! mis-attribution becomes a state difference. The inspector path and the plain
//! path must produce identical post-state across multi-transaction blocks —
//! including blocks whose transactions perform internal creates, whose frames
//! must not bleed multi-gas into the following transaction. The header base fee
//! is set above the `base_fee_wei` floor so the refund is active and exercised.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

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
const ARBOS_VERSION: u64 = 60;
// Header base fee, above the harness's initial `base_fee_wei` floor (0.1 Gwei)
// so the v60 multi-gas refund is active for every transaction.
const HEADER_BASE_FEE: u64 = 150_000_000;
const STORE: Address = address!("00000000000000000000000000000000c0c0c0c0");
const FACTORY: Address = address!("00000000000000000000000000000000fac70217");
const DISPERSE: Address = address!("00000000000000000000000000000000d15b0d15");
const RECIP_EOA: Address = address!("00000000000000000000000000000000eee00001");
const RECIP_NEW: Address = address!("00000000000000000000000000000000eee00002");
const RECIP_CONTRACT: Address = address!("00000000000000000000000000000000c0de0003");
const FAILED_XFER: Address = address!("00000000000000000000000000000000fa11ed00");
const OOG_CALLER: Address = address!("000000000000000000000000000000000060067a");

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

/// Stores 1 at slot 0: PUSH1 1; PUSH1 0; SSTORE; STOP.
fn store_code() -> Vec<u8> {
    vec![0x60, 0x01, 0x60, 0x00, 0x55, 0x00]
}

/// Performs one internal CREATE of a 32-byte-runtime child, then STOP. The
/// create runs as a nested frame — the case whose gas must stay attributed to
/// this transaction.
fn factory_code() -> Vec<u8> {
    vec![
        0x64, 0x60, 0x20, 0x60, 0x00, 0xf3, // PUSH5 <init: PUSH1 0x20 PUSH1 0 RETURN>
        0x60, 0x00, // PUSH1 0
        0x52, // MSTORE (init code at mem[27..32])
        0x60, 0x05, // PUSH1 5   (init size)
        0x60, 0x1b, // PUSH1 27  (init offset)
        0x60, 0x00, // PUSH1 0   (value)
        0xf0, // CREATE
        0x50, // POP
        0x00, // STOP
    ]
}

/// Accepts a value transfer using part of its stipend: PUSH1 1; PUSH1 2; ADD;
/// POP; STOP.
fn stipend_user_code() -> Vec<u8> {
    vec![0x60, 0x01, 0x60, 0x02, 0x01, 0x50, 0x00]
}

/// Sends 1 wei to each recipient via a zero-gas CALL, then STOP. Each call
/// forwards no gas, so the child runs on the bare value-transfer stipend: the
/// two EOAs leave it unspent (one existing, one created by the transfer), the
/// contract spends a little. The unspent stipend is returned to this frame, so
/// none of it may be attributed as the caller's own gas.
fn disperse_code() -> Vec<u8> {
    let mut code = Vec::new();
    for r in [RECIP_EOA, RECIP_NEW, RECIP_CONTRACT] {
        code.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00]);
        code.extend_from_slice(&[0x60, 0x01]); // value
        code.push(0x73); // PUSH20
        code.extend_from_slice(r.as_slice());
        code.extend_from_slice(&[0x60, 0x00]); // gas
        code.extend_from_slice(&[0xf1, 0x50]); // CALL; POP
    }
    code.push(0x00); // STOP
    code
}

/// Attempts a single value transfer it cannot fund (zero balance): the call
/// fails the balance check before a child frame is entered, so its gas is
/// classified from the call opcode delta alone.
fn failed_xfer_code() -> Vec<u8> {
    let mut code = vec![
        0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x01, 0x73,
    ];
    code.extend_from_slice(RECIP_EOA.as_slice());
    code.extend_from_slice(&[0x60, 0x00, 0xf1, 0x50, 0x00]); // gas; CALL; POP; STOP
    code
}

/// Calls [`STORE`] forwarding only 2000 gas, so its cold SSTORE runs out of gas
/// mid-charge. The caller ignores the failed sub-call and returns normally.
fn oog_caller_code() -> Vec<u8> {
    let mut code = vec![0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00];
    code.push(0x73); // PUSH20 STORE
    code.extend_from_slice(STORE.as_slice());
    code.extend_from_slice(&[0x61, 0x07, 0xd0]); // PUSH2 2000 (gas)
    code.extend_from_slice(&[0xf1, 0x50, 0x00]); // CALL; POP; STOP
    code
}

fn call_tx(to: Address, nonce: u64) -> Recovered<ArbTransactionSigned> {
    let tx = TxLegacy {
        chain_id: Some(CHAIN_ID),
        nonce,
        gas_price: 1_000_000_000,
        gas_limit: 500_000,
        to: TxKind::Call(to),
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
    env.block_env.basefee = HEADER_BASE_FEE;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(1u64);
    env
}

fn exec_ctx() -> EthBlockExecutionCtx<'static> {
    EthBlockExecutionCtx {
        tx_count_hint: Some(4),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    }
}

/// (sender balance, STORE slot 0, FACTORY nonce, disperse recipient balances).
type PostState = (U256, U256, u64, U256, U256, U256);

fn read_post_state(h: &mut ArbosHarness) -> PostState {
    let bal = |h: &mut ArbosHarness, a: Address| {
        h.state()
            .cache
            .accounts
            .get(&a)
            .and_then(|c| c.account.as_ref())
            .map(|x| x.info.balance)
            .unwrap_or(U256::ZERO)
    };
    let slot0 = h
        .state()
        .cache
        .accounts
        .get(&STORE)
        .and_then(|a| a.account.as_ref())
        .and_then(|a| a.storage.get(&U256::ZERO).copied())
        .unwrap_or(U256::ZERO);
    let factory_nonce = h
        .state()
        .cache
        .accounts
        .get(&FACTORY)
        .and_then(|a| a.account.as_ref())
        .map(|a| a.info.nonce)
        .unwrap_or(0);
    (
        bal(h, sender()),
        slot0,
        factory_nonce,
        bal(h, RECIP_EOA),
        bal(h, RECIP_NEW),
        bal(h, RECIP_CONTRACT),
    )
}

fn harness() -> ArbosHarness {
    let mut h = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();
    set_account(
        h.state(),
        sender(),
        AccountInfo {
            balance: U256::from(10u64).pow(U256::from(19u64)),
            ..Default::default()
        },
    );
    for (addr, bytes, balance) in [
        (STORE, store_code(), 0u64),
        (FACTORY, factory_code(), 0),
        (RECIP_CONTRACT, stipend_user_code(), 0),
        (DISPERSE, disperse_code(), 1_000),
        (FAILED_XFER, failed_xfer_code(), 0),
        (OOG_CALLER, oog_caller_code(), 0),
    ] {
        let code = Bytecode::new_raw(bytes.into());
        set_account(
            h.state(),
            addr,
            AccountInfo {
                nonce: 1,
                balance: U256::from(balance),
                code_hash: code.hash_slow(),
                code: Some(code),
                ..Default::default()
            },
        );
    }
    set_account(
        h.state(),
        RECIP_EOA,
        AccountInfo {
            balance: U256::from(5u64),
            ..Default::default()
        },
    );
    h
}

/// Runs `targets` as a sequence of single-call transactions from one sender,
/// without the multi-gas inspector.
fn run_plain(targets: &[Address]) -> PostState {
    let mut h = harness();
    let cfg = ArbEvmConfig::new(Arc::new(ChainSpec::default()));
    let factory = cfg.block_executor_factory();
    let evm = factory.evm_factory().create_evm(h.state(), block_env());
    let mut executor = factory.create_arb_executor(evm, exec_ctx(), CHAIN_ID);
    executor.arb_ctx.basefee = U256::from(HEADER_BASE_FEE);
    executor.apply_pre_execution_changes().unwrap();
    for (nonce, target) in targets.iter().enumerate() {
        let result = executor
            .execute_transaction_without_commit(call_tx(*target, nonce as u64))
            .unwrap();
        executor.commit_transaction(result).unwrap();
    }
    executor.finish().unwrap();
    read_post_state(&mut h)
}

/// Same as [`run_plain`] but with the multi-gas inspector installed, switching
/// block execution to revm's inspect path.
fn run_inspected(targets: &[Address]) -> PostState {
    let mut h = harness();
    let cfg = ArbEvmConfig::new(Arc::new(ChainSpec::default()));
    let factory = cfg.block_executor_factory();
    let sink = MultiGasSink::default();
    let evm = factory.evm_factory().create_evm_with_inspector(
        h.state(),
        block_env(),
        MultiGasInspector::with_sink(sink.clone()),
    );
    let mut executor = factory.create_arb_executor(evm, exec_ctx(), CHAIN_ID);
    executor.set_multi_gas_sink(sink);
    executor.arb_ctx.basefee = U256::from(HEADER_BASE_FEE);
    executor.apply_pre_execution_changes().unwrap();
    for (nonce, target) in targets.iter().enumerate() {
        let result = executor
            .execute_transaction_without_commit(call_tx(*target, nonce as u64))
            .unwrap();
        executor.commit_transaction(result).unwrap();
    }
    executor.finish().unwrap();
    read_post_state(&mut h)
}

/// Each ordering: the inspector path must equal the plain path, and the
/// transactions must actually have executed (store written, create performed).
fn assert_equivalent(targets: &[Address], creates: u64) {
    let plain = run_plain(targets);
    let inspected = run_inspected(targets);
    assert_eq!(
        plain, inspected,
        "inspector path diverged from plain path for {targets:?}",
    );
    if targets.contains(&STORE) {
        assert_eq!(plain.1, U256::from(1u64), "STORE must have executed");
    }
    assert_eq!(
        plain.2,
        1 + creates,
        "FACTORY nonce must reflect {creates} create(s)",
    );
    assert!(
        plain.0 < U256::from(10u64).pow(U256::from(19u64)),
        "sender must have been charged",
    );
}

#[test]
fn create_then_call_is_consensus_equivalent() {
    assert_equivalent(&[FACTORY, STORE], 1);
}

#[test]
fn call_then_create_is_consensus_equivalent() {
    assert_equivalent(&[STORE, FACTORY], 1);
}

#[test]
fn repeated_creates_then_call_is_consensus_equivalent() {
    assert_equivalent(&[FACTORY, FACTORY, STORE], 2);
}

#[test]
fn plain_multi_call_is_consensus_equivalent() {
    assert_equivalent(&[STORE, STORE], 0);
}

/// A contract that disperses value through zero-gas CALLs: each child runs on
/// the bare stipend and returns most of it. The returned stipend must not be
/// attributed to the caller, or the inspector over-counts and the refund (and
/// thus the sender's balance) diverges from the plain path.
#[test]
fn disperse_value_transfers_is_consensus_equivalent() {
    let plain = run_plain(&[DISPERSE]);
    let inspected = run_inspected(&[DISPERSE]);
    assert_eq!(
        plain, inspected,
        "value-transfer disperse diverged: inspect vs plain",
    );
    assert_eq!(plain.3, U256::from(6u64), "existing EOA received 1 wei");
    assert_eq!(plain.4, U256::from(1u64), "new account received 1 wei");
    assert_eq!(plain.5, U256::from(1u64), "contract received 1 wei");
    assert!(
        plain.0 < U256::from(10u64).pow(U256::from(19u64)),
        "sender must have been charged",
    );
}

/// A value transfer that fails the balance check enters no child frame. Its
/// frame bookkeeping must still balance — a leaked open frame would withhold
/// the failing transaction's multi-gas and bleed it into the following one,
/// diverging that transaction's refund from the plain path.
#[test]
fn failed_value_transfer_then_call_is_consensus_equivalent() {
    let plain = run_plain(&[FAILED_XFER, STORE]);
    let inspected = run_inspected(&[FAILED_XFER, STORE]);
    assert_eq!(
        plain, inspected,
        "failed transfer diverged: inspect vs plain",
    );
    assert_eq!(
        plain.3,
        U256::from(5u64),
        "recipient unchanged by failed transfer"
    );
    assert_eq!(
        plain.1,
        U256::from(1u64),
        "STORE executed after failed transfer"
    );
}

/// An opcode that runs out of gas is charged less than its nominal cost. Its
/// multi-gas must reflect only the gas consumed, not the full typed cost, or
/// the inspector over-attributes and the refund diverges from the plain path.
#[test]
fn out_of_gas_sstore_is_consensus_equivalent() {
    let plain = run_plain(&[OOG_CALLER]);
    let inspected = run_inspected(&[OOG_CALLER]);
    assert_eq!(
        plain, inspected,
        "out-of-gas sstore diverged: inspect vs plain",
    );
    assert_eq!(
        plain.1,
        U256::ZERO,
        "STORE slot unwritten (sub-call ran out of gas)"
    );
}
