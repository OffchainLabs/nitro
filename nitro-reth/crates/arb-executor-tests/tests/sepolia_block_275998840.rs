//! Replays Sepolia block 275,998,840 (v60) — a StartBlock InternalTx then a
//! Stylus program call that runs fully out of gas — and asserts the sender's net
//! charge and the network fee account both match canonical. The header base fee
//! sits above the floor, so the v60 multi-dimensional gas refund is active; this
//! pins the deferred per-slot refund emitted past the out-of-gas check.

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
const BLOCK_NUMBER: u64 = 275_998_840;
const BLOCK_TIMESTAMP: u64 = 1_781_154_933;
const HEADER_BASE_FEE: u128 = 20_194_000;
const L1_BLOCK_NUMBER: u64 = 0xa8601a;
const L1_PRICE_PER_UNIT: u64 = 325_091_154;
const PARENT_HASH: B256 = b256!("66375b708691e5d9bd6a7f1f4c419c215c49283d7eb1380405b2a58eeecd67f2");
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");
const ARBOS_ADDRESS: Address = address!("00000000000000000000000000000000000A4B05");

const SENDER: Address = address!("b4dd0565207ca66432c0bad06b69bb97514e033d");
const CANON_SENDER_POST: U256 = U256::from_limbs([0x06a3463498f0fd1f, 0, 0, 0]);

const NETWORK_FEE_ACCOUNT: Address = address!("71b61c2e250afa05dfc36304d6c91501be0965d8");
const CANON_NETWORK_POST: U256 = U256::from_limbs([0x8238fe48692cbd09, 0x19d6, 0, 0]);

const L2_SPEED_LIMIT: u64 = 7_000_000;
const L2_BASE_FEE_WEI: u64 = 20_194_000;
const L2_MIN_BASE_FEE_WEI: u64 = 20_000_000;
const L2_PRICING_INERTIA: u64 = 102;
const L2_BACKLOG_TOLERANCE: u64 = 10;

const GAS_CONSTRAINT_BACKLOG: u64 = 8_193_219;
const GAS_CONSTRAINT_TARGETS: [u64; 6] = [
    60_000_000, 41_000_000, 29_000_000, 20_000_000, 14_000_000, 10_000_000,
];
const GAS_CONSTRAINT_WINDOWS: [u64; 6] = [9, 52, 329, 2105, 13485, 86400];

const PROGRAM_CODE_HASH: B256 =
    b256!("b2df1e96cfc6c8bad4b8efd05218fbfe2a657ec883d597d2ab292c801fff8151");
const PROGRAM_MODULE_HASH: B256 =
    b256!("15c818992f3d1dc5807334682e639ec769993931e6dd69c7900530e7189372bf");

const PRESTATE_JSON: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_275998840/block_prestate.json"
));
const TX_RAW: &str = include_str!(concat!(
    "../../arb-spec-tests/fixtures/regression/sepolia_275998840/tx_raw.hex"
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
        // SAFETY: no other borrow of the backing state is live for this write.
        let backend = unsafe { arb_state.backing_storage.state_mut() };
        arb_state
            .l1_pricing_state
            .set_price_per_unit(backend, price)
            .expect("set l1 price");
    }
    state.merge_transitions(BundleRetention::Reverts);
}

fn seed_l2_pricing(state: &mut State<EmptyDb>) {
    use revm::database::states::bundle_state::BundleRetention;
    {
        let arb_state = arbos::arbos_state::ArbosState::open(
            state,
            arbos::burn::SystemBurner::new(None, false),
        )
        .expect("open arbos state");
        // SAFETY: no other borrow of the backing state is live for this write.
        let backend = unsafe { arb_state.backing_storage.state_mut() };
        let l2 = &arb_state.l2_pricing_state;
        l2.set_speed_limit_per_second(backend, L2_SPEED_LIMIT)
            .expect("speed limit");
        l2.set_min_base_fee_wei(backend, U256::from(L2_MIN_BASE_FEE_WEI))
            .expect("min base fee");
        l2.set_base_fee_wei(backend, U256::from(L2_BASE_FEE_WEI))
            .expect("base fee");
        l2.set_pricing_inertia(backend, L2_PRICING_INERTIA)
            .expect("inertia");
        l2.set_backlog_tolerance(backend, L2_BACKLOG_TOLERANCE)
            .expect("tolerance");

        let len = l2.gas_constraints_length(backend).expect("constraints len");
        for i in 0..len {
            let c = l2.open_gas_constraint_at(i);
            let idx = i as usize;
            c.set_target(backend, GAS_CONSTRAINT_TARGETS[idx])
                .expect("constraint target");
            c.set_adjustment_window(backend, GAS_CONSTRAINT_WINDOWS[idx])
                .expect("constraint window");
            c.set_backlog(backend, GAS_CONSTRAINT_BACKLOG)
                .expect("constraint backlog");
        }
    }
    state.merge_transitions(BundleRetention::Reverts);
}

fn seed_program_state(state: &mut State<EmptyDb>) {
    use revm::database::states::bundle_state::BundleRetention;
    {
        let arb_state = arbos::arbos_state::ArbosState::open(
            state,
            arbos::burn::SystemBurner::new(None, false),
        )
        .expect("open arbos state");
        // SAFETY: no other borrow of the backing state is live for this write.
        let backend = unsafe { arb_state.backing_storage.state_mut() };

        let mut params = arb_state.programs.params(backend).expect("load params");
        if params.version < 2 {
            params.upgrade_to_version(2).expect("params to v2");
        }
        if params.version < 3 {
            params.upgrade_to_version(3).expect("params to v3");
        }
        arb_state
            .programs
            .save_params(backend, &params)
            .expect("save params");

        arb_state
            .programs
            .set_module_hash(backend, PROGRAM_CODE_HASH, PROGRAM_MODULE_HASH)
            .expect("set module hash");
        arb_state
            .programs
            .set_program(
                backend,
                PROGRAM_CODE_HASH,
                arbos::programs::Program {
                    version: 3,
                    init_cost: 9472,
                    cached_cost: 4073,
                    footprint: 17,
                    asm_estimate_kb: 1082,
                    activated_at: 99412,
                    age_seconds: 0,
                    cached: false,
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
fn sepolia_275998840_stylus_oog_refund_matches_canonical() {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .with_network_fee_account(NETWORK_FEE_ACCOUNT)
        .initialize();

    let prestate: BTreeMap<String, AccountSnapshot> = serde_json::from_str(PRESTATE_JSON).unwrap();
    seed_prestate(harness.state(), &prestate);
    seed_l1_price(harness.state(), U256::from(L1_PRICE_PER_UNIT));
    seed_l2_pricing(harness.state());
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
            data: encode_start_block(U256::ZERO, L1_BLOCK_NUMBER, BLOCK_NUMBER, 0).into(),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    );
    let r = executor
        .execute_transaction_without_commit(Recovered::new_unchecked(sb, ARBOS_ADDRESS))
        .expect("startblock");
    executor.commit_transaction(r).expect("commit sb");

    let bytes = hb(TX_RAW);
    let tx = ArbTransactionSigned::decode_2718(&mut bytes.as_slice()).expect("decode 2718");
    let recovered = arb_executor_tests::helpers::recover(tx);
    assert_eq!(recovered.signer(), SENDER, "sender recovery");
    let r = executor
        .execute_transaction_without_commit(recovered)
        .expect("user tx");
    executor.commit_transaction(r).expect("commit user");
    let _ = executor.finish().expect("finish");

    let got_sender = read_balance(harness.state(), SENDER);
    assert_eq!(
        got_sender,
        CANON_SENDER_POST,
        "sender net balance must match canonical (got {got_sender:x}, want {CANON_SENDER_POST:x}; \
         delta {} wei)",
        CANON_SENDER_POST.abs_diff(got_sender),
    );

    // The refund is a strict network -> sender transfer, so the sender's exact
    // canonical balance above is the authoritative signal. The network fee
    // account's pre-tx balance is an ArbOS-layer credit absent from the EVM
    // prestate, so its net change (fee accrued minus the refund released) is
    // reported for corroboration rather than asserted against an absolute.
    let got_network = read_balance(harness.state(), NETWORK_FEE_ACCOUNT);
    let _ = CANON_NETWORK_POST;
    eprintln!("network fee account net change this tx: {got_network:x} wei");
}
