//! Replays Sepolia block 269,737,870 (v60) through the multi-gas inspector
//! path — a StartBlock InternalTx then an ArbWasm.activateProgram call for a
//! classic Stylus program — and asserts the sender's net balance matches
//! canonical. The base fee sits at the floor, so there is no refund path; this
//! pins ArbWasm.activateProgram's cached-program re-activation gas.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::{collections::BTreeMap, sync::Arc};

use alloy_consensus::transaction::Recovered;
use alloy_eips::Decodable2718;
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, b256, hex, Address, Bytes, Signature, B256, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_evm::{
    config::ArbEvmConfig,
    multi_gas::{MultiGasInspector, MultiGasSink},
};
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
const BLOCK_NUMBER: u64 = 269_737_870;
const BLOCK_TIMESTAMP: u64 = 1_779_283_472;
const HEADER_BASE_FEE: u128 = 20_000_000; // floor
const L1_BLOCK_NUMBER: u64 = 0xa61af2;
const L1_PRICE_PER_UNIT: u64 = 52_223_638;
const PARENT_HASH: B256 = b256!("823eb08ad7f64e8025b60d9f54a3c697cffebe603c78c25c9f3847cbbb7d2e72");
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");
const ARBOS_ADDRESS: Address = address!("00000000000000000000000000000000000A4B05");

const SENDER: Address = address!("61860509a5f05bde458a5318d8e54654f60d0c38");
const CANON_SENDER_POST: U256 = U256::from_limbs([0x07a1c3cd085d54da, 0, 0, 0]);

const PRESTATE_JSON: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269737870/block_start_prestate.json"
));
const TX1: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_269737870/tx1_raw.hex"
));

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

fn seed_l1_price(state: &mut State<EmptyDb>, price: U256) {
    use revm::database::states::bundle_state::BundleRetention;
    {
        let arb_state = arbos::arbos_state::ArbosState::open(
            state,
            arbos::burn::SystemBurner::new(None, false),
        )
        .expect("open arbos state");
        // SAFETY: re-borrow the backing state to write the L1 price.
        let backend = unsafe { arb_state.backing_storage.state_mut() };
        arb_state
            .l1_pricing_state
            .set_price_per_unit(backend, price)
            .expect("set l1 price");
    }
    state.merge_transitions(BundleRetention::Reverts);
}

// The program's existing activation state (version, cached flag, module hash)
// lives in the programs subspace, read by the system activation path rather than
// by an opcode, so it is absent from the prestate trace. Canonical shows the
// program is already activated at version 2 and cached, so the activation is a
// cached re-activation (version 2 -> 3); seed that state.
const PROGRAM_CODE_HASH: B256 =
    b256!("0231f2c7abceefa5239f94514d4bd7662c99b5091ca65f6e04ac9c89c82d3657");

fn seed_program_state(state: &mut State<EmptyDb>) {
    use revm::database::states::bundle_state::BundleRetention;
    {
        let arb_state = arbos::arbos_state::ArbosState::open(
            state,
            arbos::burn::SystemBurner::new(None, false),
        )
        .expect("open arbos state");
        // SAFETY: re-borrow the backing state to write the program state.
        let backend = unsafe { arb_state.backing_storage.state_mut() };
        arb_state
            .programs
            .set_module_hash(backend, PROGRAM_CODE_HASH, B256::repeat_byte(0xab))
            .expect("set module hash");
        arb_state
            .programs
            .set_program(
                backend,
                PROGRAM_CODE_HASH,
                arbos::programs::Program {
                    version: 2,
                    init_cost: 0,
                    cached_cost: 0,
                    footprint: 0,
                    asm_estimate_kb: 0,
                    activated_at: 1,
                    age_seconds: 0,
                    cached: true,
                },
            )
            .expect("set program");
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

#[test]
fn sepolia_269737870_sender_net_charge_matches_canonical() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    let prestate: BTreeMap<String, AccountSnapshot> = serde_json::from_str(PRESTATE_JSON).unwrap();
    seed_prestate(harness.state(), &prestate);
    seed_l1_price(harness.state(), U256::from(L1_PRICE_PER_UNIT));
    seed_program_state(harness.state());

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

    let sink = MultiGasSink::default();
    let evm = evm_factory.create_evm_with_inspector(
        harness.state(),
        env,
        MultiGasInspector::with_sink(sink.clone()),
    );
    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(2),
        parent_hash: PARENT_HASH,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };
    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor.set_multi_gas_sink(sink);
    executor.arb_ctx.block_timestamp = BLOCK_TIMESTAMP;
    executor.arb_ctx.basefee = U256::from(HEADER_BASE_FEE);
    executor.arb_ctx.l2_block_number = BLOCK_NUMBER;
    executor.arb_ctx.l1_block_number = L1_BLOCK_NUMBER;
    executor.apply_pre_execution_changes().expect("pre-exec");

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

    let bytes = hb(TX1);
    let tx = ArbTransactionSigned::decode_2718(&mut bytes.as_slice()).expect("decode 2718");
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
         delta {} wei)",
        CANON_SENDER_POST.abs_diff(got),
    );
}
